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
)

type Service struct {
	client  etcd.Client
	options Options
	ng      en.Engine
	ngiSvc  *negroni.Negroni
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

func (s *Service) load() error {
	// init engine
	if s.options.EtcdApiVersion == 2 {
		return errors.New("Unsupport etcdApiVersion=2")
	}

	ng, err := etcd3.New(s.options.EtcdNodes, s.options.EtcdKey, etcd3.Options{})
	if err != nil {
		return err
	}
	s.ng = ng
	snp, err := s.ng.GetSnapshot()
	if err != nil {
		return err
	}

	fmt.Printf("loadNG GetSnapshot -> %#v\n", snp)

	// TODO: 初始化服务
	// 加载路由匹配中间件
	// 加载代理

	// negroni
	n := negroni.New()
	// context
	n.UseFunc(xwaymw.DefaultXWayContext())
	// router
	n.Use(router.New())
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

// Run ...
func Run() error {
	// 加载配置
	options, err := ParseCommandLine()
	if err != nil {
		return fmt.Errorf("failed to parse command line: %s", err)
	}

	appLogger.Info("初始化......")
	// fmt.Printf("options: %v\n", options)
	s := NewService(options)
	if err := s.load(); err != nil {
		return fmt.Errorf("service start failure: %s", err)
	}

	return nil
}
