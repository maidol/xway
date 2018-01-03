package etcd3

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/etcd/mvcc/mvccpb"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/sirupsen/logrus"

	"xway/engine"
)

type ng struct {
	nodes         []string
	etcdKey       string
	client        *etcd.Client
	context       context.Context
	cancelFunc    context.CancelFunc
	logsev        logrus.Level
	options       Options
	requireQuorum bool
}

type Options struct {
	EtcdConsistency         string
	EtcdSyncIntervalSeconds int64
}

func New(nodes []string, etcdKey string, options Options) (engine.Engine, error) {
	n := &ng{
		nodes:   nodes,
		etcdKey: "/" + etcdKey,
		options: options,
	}

	if err := n.reconnect(); err != nil {
		return nil, err
	}

	return n, nil
}

func (n *ng) reconnect() error {
	var client *etcd.Client
	cfg := n.getEtcdClientConfig()
	var err error
	if client, err = etcd.New(cfg); err != nil {
		return err
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	n.context = ctx
	n.cancelFunc = cancelFunc

	if n.client != nil {
		// 关闭
		n.client.Close()
	}

	n.client = client
	n.requireQuorum = true
	if n.options.EtcdConsistency == "WEAK" {
		n.requireQuorum = false
	}
	return nil
}

func (n *ng) getEtcdClientConfig() etcd.Config {
	return etcd.Config{
		Endpoints: n.nodes,
	}
}

func (n *ng) parseFrontends(kvs []*mvccpb.KeyValue) ([]engine.FrontendSpec, error) {
	frontendSpecs := []engine.FrontendSpec{}
	for _, kv := range kvs {
		// fmt.Println("-> frontend kv", string(kv.Key), string(kv.Value))
		// frontend, err := engine.FrontendFromJSON(n.registry.GetRouter(), []byte(kv.Value))
		frontend, err := engine.FrontendFromJSON(nil, []byte(kv.Value))
		if err != nil {
			return nil, err
		}
		// fmt.Println("-> frontend:", frontend)
		frontendSpec := engine.FrontendSpec{
			Frontend: *frontend,
		}
		frontendSpecs = append(frontendSpecs, frontendSpec)
	}
	return frontendSpecs, nil
}

func (n *ng) GetSnapshot() (*engine.Snapshot, error) {
	response, err := n.client.Get(n.context, n.etcdKey, etcd.WithPrefix(), etcd.WithSort(etcd.SortByKey, etcd.SortAscend))
	if err != nil {
		return nil, err
	}

	frontends, err := n.parseFrontends(filterByPrefix(response.Kvs, n.etcdKey+"/frontends"))
	if err != nil {
		return nil, err
	}

	s := &engine.Snapshot{Index: uint64(response.Header.Revision), FrontendSpecs: frontends}

	return s, nil
}

func filterByPrefix(kvs []*mvccpb.KeyValue, prefix string) []*mvccpb.KeyValue {
	returnValue := make([]*mvccpb.KeyValue, 0, 10)
	for _, kv := range kvs {
		if strings.Index(string(kv.Key), prefix) == 0 {
			returnValue = append(returnValue, kv)
		}
	}
	return returnValue
}

func (n *ng) Subscribe(changes chan interface{}, afterIdx uint64, cancelC chan struct{}) error {
	watcher := etcd.NewWatcher(n.client)
	defer watcher.Close()
	rch := watcher.Watch(n.context, n.etcdKey+"/", etcd.WithPrefix())
	for wresp := range rch {
		if wresp.Canceled {
			fmt.Println("Stop watching: graceful shutdown")
			return nil
		}
		if err := wresp.Err(); err != nil {
			fmt.Printf("Stop watching: error: %v\n", err)
			return err
		}

		for _, ev := range wresp.Events {
			// fmt.Printf("n.client.Watch %s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			change, err := n.parseChange(ev)
			if err != nil {
				continue
			}
			if change != nil {
				fmt.Printf("ng.Subscribe chagne %v\n", change)
				select {
				case changes <- change:
				case <-cancelC:
					return nil
				}
			}
		}
	}
	return nil
}

type MatcherFn func(*etcd.Event) (interface{}, error)

func (n *ng) parseChange(e *etcd.Event) (interface{}, error) {
	matchers := []MatcherFn{
		n.parseFrontendChange,
	}
	for _, matcher := range matchers {
		m, err := matcher(e)
		if m != nil || err != nil {
			return m, err
		}
	}
	return nil, nil
}

func (n *ng) parseFrontendChange(e *etcd.Event) (interface{}, error) {
	switch e.Type {
	case etcd.EventTypePut:
		frontend, err := engine.FrontendFromJSON(nil, e.Kv.Value)
		if err != nil {
			return e, err
		}
		return frontend, nil
	case etcd.EventTypeDelete:
		return e, nil
	}
	return nil, nil
}
