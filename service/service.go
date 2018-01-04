package service

import (
	"errors"
	"fmt"
	"os"

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

type Service struct {
	client  etcd.Client
	options Options
	ng      en.Engine
	ngiSvc  *negroni.Negroni
	router  router.Router
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
	// 获取快照
	// 加载路由匹配中间件
	// 加载代理

	// 获取快照
	snp, err := s.ng.GetSnapshot()
	if err != nil {
		return err
	}
	// fmt.Printf("GetSnapshot -> %#v\n", snp)
	// TODO: 需要处理发生错误时, goruntine退出
	changes := make(chan interface{})
	cancelC := make(chan struct{})
	go s.ng.Subscribe(changes, snp.Index, cancelC)
	go func() {
		for change := range changes {
			// fmt.Printf("/xway/ change %v\n", change)
			s.processChange(change)
		}
	}()

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
	s.ngiSvc.Run(":" + fmt.Sprint(s.options.Port))

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

	return nil
}
