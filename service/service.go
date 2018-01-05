package service

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	logrus_logstash "github.com/bshuster-repo/logrus-logstash-hook"
	etcd "github.com/coreos/etcd/client"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	en "xway/engine"
	"xway/engine/etcd3"
	"xway/middleware"
	"xway/proxy"
	"xway/router"
	"xway/router/xrouter"
)

const (
	retryPeriod       = 5 * time.Second
	changesBufferSize = 2000
)

type Service struct {
	client         etcd.Client
	options        Options
	ng             en.Engine
	ngiSvc         *negroni.Negroni
	router         router.Router
	watcherCancelC chan struct{}
	watcherErrorC  chan struct{}
	watcherWg      sync.WaitGroup
}

var appLogger *logrus.Entry

func init() {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel
	logger.Formatter = new(logrus.TextFormatter)
	// logger.Out = os.Stdout

	// conn, err := net.Dial("tcp", "logstash.mycompany.net:8911")
	// if err != nil {
	// 	logrus.Fatal(err)
	// }

	// hook := logrus_logstash.New(conn, logrus_logstash.DefaultFormatter(logrus.Fields{"type": "xway"}))

	// if err != nil {
	// 	logrus.Fatal(err)
	// }

	// logger.Hooks.Add(hook)

	stdHook := logrus_logstash.New(os.Stdout, logrus_logstash.DefaultFormatter(logrus.Fields{"type": "xway"}))
	logger.Hooks.Add(stdHook)

	appLogger = logger.WithFields(logrus.Fields{"name": "app"})
}

func NewService(options Options) *Service {
	return &Service{
		options: options,
	}
}

func (s *Service) initEngine() error {
	// init engine
	if len(s.options.EtcdNodes) == 0 {
		// 默认值
		s.options.EtcdNodes.Set("http://localhost:2379")
	}
	if s.options.EtcdApiVersion == 2 {
		return errors.New("Unsupport etcdApiVersion=2")
	}

	ng, err := etcd3.New(s.options.EtcdNodes, s.options.EtcdKey, etcd3.Options{})
	if err != nil {
		return err
	}
	s.ng = ng

	return nil
}

func (s *Service) processChange(ch interface{}) error {
	switch change := ch.(type) {
	case *en.FrontendUpserted:
		return s.router.Handle(change.Frontend)

	case *en.FrontendDeleted:
		return s.router.Remove(change.FrontendKey.Id)
	}
	return fmt.Errorf("unsupported change: %v", ch)
}

func (s *Service) initProxy() error {
	// TODO: 初始化代理服务
	// 1. 获取快照/启动goroutine监听router等信息的变化
	// 2. 加载路由匹配中间件/加载代理服务
	// 初始化代理失败需要安全退出所有goroutine和关闭channel
	// 3. 启动goroutine处理router等信息的变化

	// 获取快照/监听router等信息的变化
	snp, err := s.ng.GetSnapshot()
	if err != nil {
		return err
	}
	// fmt.Printf("GetSnapshot -> %#v\n", snp)
	// TODO: 需要处理发生错误时, goroutine退出
	cancelWatcher := true // 标记是否取消监听router等信息的变化
	changes := make(chan interface{}, changesBufferSize)
	s.watcherCancelC = make(chan struct{})
	s.watcherErrorC = make(chan struct{})
	// s.watcherWg关联开启的goroutine
	s.watcherWg.Add(1)
	go func() {
		defer s.watcherWg.Done() // 执行顺序 2
		defer close(changes)     // 执行顺序 1
		if err := s.ng.Subscribe(changes, snp.Index, s.watcherCancelC); err != nil {
			fmt.Printf("[engine.Subscribe] watcher failed: '%v'\n", err)
			s.watcherErrorC <- struct{}{}
			return
		}
		// s.watcherCancelC <- struct{}{}
		fmt.Println("[engine.Subscribe] watcher shutdown")
	}()
	// 初始化代理服务失败时需要等待对router进行监听的goroutine完全退出才能退出initProxy
	// Make sure watcher goroutine [close(changes)] is stopped if initialization fails.
	defer func() {
		if cancelWatcher {
			close(s.watcherCancelC)
			s.watcherWg.Wait() // 阻塞并等待所有goroutine退出
		}
	}()

	// 加载路由匹配中间件/加载代理服务
	// negroni
	n := negroni.New()
	// context
	n.UseFunc(xwaymw.DefaultXWayContext())
	// router
	r := xrouter.New(snp)
	s.router = r.(router.Router)
	n.Use(r)
	// proxy
	p, err := proxy.NewDo()
	if err != nil {
		return err
	}
	n.UseHandlerFunc(p)
	s.ngiSvc = n

	// 服务初始化后, cancelWatcher置为false
	// server has been successfully started therefore we do not need
	// to cancel the watcher.
	cancelWatcher = false

	// 处理router等信息的变化
	// This goroutine will listen for changes arriving to the changes channel
	// and reconfigure the given server router.
	s.watcherWg.Add(1)
	go func() {
		defer s.watcherWg.Done()
		for change := range changes {
			// fmt.Printf("/xway/ change %v\n", change)
			if err := s.processChange(change); err != nil {
				fmt.Printf("failed to process, change=%#v, err=%s\n", change, err)
			}
		}
		fmt.Println("change processor shutdown")
	}()

	return nil
}

func (s *Service) load() error {
	if err := s.initEngine(); err != nil {
		return err
	}

	if err := s.initProxy(); err != nil {
		return err
	}
	return nil
}

// Run ...
func Run() error {
	fmt.Println("Running......")
	// 加载配置
	options, err := ParseCommandLine()
	if err != nil {
		return fmt.Errorf("failed to parse command line: %s", err)
	}
	// fmt.Printf("加载配置options: %v\n", options)
	// fmt.Println("初始化......")

	// appLogger.Info("Starting......")
	s := NewService(options)
	if err := s.load(); err != nil {
		return fmt.Errorf("service start failure: %s", err)
	}
	// start server
	s.ngiSvc.Run(":" + fmt.Sprint(s.options.Port))
	return nil
}
