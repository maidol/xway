package service

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	logrus_logstash "github.com/bshuster-repo/logrus-logstash-hook"
	"github.com/gorilla/mux"
	logrus "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"xway/api"
	en "xway/engine"
	"xway/engine/etcd3"
	"xway/middleware"
	"xway/plugin"
	"xway/proxy"
	"xway/router"
	"xway/router/xrouter"
	"xway/utils/mq"
	"xway/utils/mysql"
	"xway/utils/redis"
	"xway/utils/xlog/kafka"
	"xway/utils/xlog/redis"
)

const (
	retryPeriod       = 5 * time.Second
	changesBufferSize = 2000
)

type Service struct {
	options  Options
	registry *plugin.Registry
	ng       en.Engine
	// ngiSvc the gateway
	ngiSvc         *negroni.Negroni
	router         router.Router
	watcherCancelC chan struct{}
	watcherErrorC  chan struct{}
	watcherWg      sync.WaitGroup
}

// var appLogger *logrus.Entry

// func init() {
// 	logger := logrus.New()
// 	logger.Level = logrus.InfoLevel
// 	logger.Formatter = new(logrus.TextFormatter)
// 	// logger.Out = os.Stdout

// 	// conn, err := net.Dial("tcp", "logstash.mycompany.net:8911")
// 	// if err != nil {
// 	// 	logrus.Fatal(err)
// 	// }

// 	// hook := logrus_logstash.New(conn, logrus_logstash.DefaultFormatter(logrus.Fields{"type": "xway"}))

// 	// if err != nil {
// 	// 	logrus.Fatal(err)
// 	// }

// 	// logger.Hooks.Add(hook)

// 	stdHook := logrus_logstash.New(os.Stdout, logrus_logstash.DefaultFormatter(logrus.Fields{"type": "xway"}))
// 	logger.Hooks.Add(stdHook)

// 	appLogger = logger.WithFields(logrus.Fields{"name": "app"})
// }

func NewService(options Options, registry *plugin.Registry) *Service {
	return &Service{
		options:  options,
		registry: registry,
	}
}

func (s *Service) initLogger() {
	logrus.SetLevel(s.options.LogSeverity.S)

	if s.options.LogFormatter != nil {
		logrus.SetOutput(os.Stdout)
		logrus.SetFormatter(s.options.LogFormatter)
		return
	}
	if s.options.Log == "console" {
		logrus.SetOutput(os.Stdout)
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
		return
	}
	if s.options.Log == "json" {
		logrus.SetOutput(os.Stdout)
		logrus.SetFormatter(&logrus.JSONFormatter{})
		return
	}
	// TODO: 容错: 连接断开重连的问题, 日志发送失败重试, 日志临时保存在内存或本地文件, 日志服务恢复时需重发......
	var err error
	var hostname string
	var e error
	if hostname, e = os.Hostname(); e != nil {
		hostname = "localhost"
	}
	if s.options.Log == "logstash" {
		logrus.SetOutput(ioutil.Discard)
		logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableSorting: true, DisableTimestamp: true})
		fm := logrus_logstash.DefaultFormatter(logrus.Fields{"type": "gateway", "hostname": hostname, "logproject": "epaper", "logstore": "gateway"})
		// TODO: 建议: 考虑并发写conn的情况(conn.Write是阻塞的)
		// 连接的使用优化成连接池+goroutine池并且以队列方式异步处理(队列中, 多个goroutine处理多个连接)
		var conn net.Conn
		// conn, err = net.Dial("udp", "192.168.2.155:64100")
		conn, err = net.Dial("tcp", "192.168.2.155:64756")
		if err != nil {
			log.Fatal(err)
		}
		hook := logrus_logstash.New(conn, fm)
		logrus.AddHook(hook)
		// go已经对conn.Write做了线程同步的支持
		logrus.StandardLogger().SetNoLock()
		return
	}
	if s.options.Log == "redis" {
		hid := strconv.FormatInt(time.Now().Unix(), 10)
		levels := logrus.AllLevels
		// fm := &logrus.JSONFormatter{}
		fm := logrus_logstash.DefaultFormatter(logrus.Fields{"type": "gateway", "hostname": hostname, "logproject": "epaper", "logstore": "gateway"})
		p := s.registry.GetRedisPool()
		// TODO: 优化成以队列方式异步处理
		var hook *redislogrus.Hook
		hook, err = redislogrus.NewHook(hid, levels, fm, p, "gateway", true)
		if err == nil {
			logrus.SetOutput(ioutil.Discard)
			logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableSorting: true, DisableTimestamp: true})
			logrus.AddHook(hook)
			logrus.StandardLogger().SetNoLock()
			return
		}
	}
	if s.options.Log == "kafka" {
		// path := "/dev/null" // suport linux
		// if runtime.GOOS == "windows" {
		// 	path = "NUL" // suport windows
		// }
		// var devNull *os.File
		// devNull, err = os.OpenFile(path, os.O_WRONLY, 0)
		// if err == nil {}
		hid := strconv.FormatInt(time.Now().Unix(), 10)
		levels := logrus.AllLevels
		// fm := &logrus.JSONFormatter{}
		fm := logrus_logstash.DefaultFormatter(logrus.Fields{"type": "gateway", "hostname": hostname, "logproject": "epaper", "logstore": "gateway"})
		p := s.registry.GetMQProducer()
		// TODO: 建议: 考虑并发写的情况
		// 优化成连接池(多个MQProducer)+goroutine池并且以队列方式异步处理(队列中, 多个goroutine处理多个MQProducer)
		var hook *kafkalogrus.Hook
		hook, err = kafkalogrus.NewHook(hid, levels, fm, p, "gateway", true)
		if !s.options.EnableMQ {
			err = errors.New("mq was disabled")
		}
		if err == nil {
			// logrus.SetOutput(devNull)
			logrus.SetOutput(ioutil.Discard)
			logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableSorting: true, DisableTimestamp: true})
			logrus.AddHook(hook)
			logrus.StandardLogger().SetNoLock()
			return
		}
	}
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.Errorf("[initLogger] Failed to initialized logger. Fallback to default: logger=%s, err=(%v)", s.options.Log, err)
}

func (s *Service) initDB() error {
	p := redis.Pool(redis.Options{Address: s.options.RedisHost, Password: s.options.RedisPassword, DB: s.options.RedisDB, Wait: s.options.RedisWait, MaxActive: s.options.RedisMaxActive, MaxIdle: s.options.RedisMaxIdle, IdleTimeout: s.options.RedisConnIdleTimeout})
	s.registry.SetRedisPool(p)
	// fmt.Println("[registry redis success]")

	db, err := mysql.NewPool(mysql.Options{UserName: s.options.DBUserName, Password: s.options.DBPassword, Address: s.options.DBHost, DBName: s.options.GatewayDBName, MaxOpen: s.options.DBMaxOpen, MaxIdle: s.options.DBMaxIdle, MaxLifetime: s.options.DBConnMaxLifetime}) // 注意: mysql客户端MaxLifetime的大小不能大于mysql服务端设置的连接会话超时时间
	if err != nil {
		return err
	}
	s.registry.SetDBPool(db)
	// fmt.Println("[registry mysql success]")

	return nil
}

// ResetDB mysql
func (s *Service) ResetDB() error {
	db, err := mysql.NewPool(mysql.Options{UserName: s.options.DBUserName, Password: s.options.DBPassword, Address: s.options.DBHost, DBName: s.options.GatewayDBName, MaxOpen: s.options.DBMaxOpen, MaxIdle: s.options.DBMaxIdle, MaxLifetime: s.options.DBConnMaxLifetime})
	if err != nil {
		return err
	}
	odb := s.registry.GetDBPool()
	if odb != nil {
		odb.Close()
	}
	s.registry.SetDBPool(db)
	return nil
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

	ng, err := etcd3.New(s.options.EtcdNodes, s.options.EtcdKey, s.registry, etcd3.Options{})
	if err != nil {
		return err
	}
	s.ng = ng

	return nil
}

func (s *Service) initMQ() error {
	cfg := &mq.MqConfig{}
	mq.LoadJsonConfig(cfg, s.options.KafkaConfigPath)
	if s.options.KafkaAK != "" {
		cfg.Ak = s.options.KafkaAK
	}
	if s.options.KafkaPassword != "" {
		cfg.Password = s.options.KafkaPassword
	}

	producer := mq.NewProducer(cfg)
	s.registry.SetMQProducer(producer)

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
		return fmt.Errorf("s.ng.GetSnapshot failure: %v", err)
	}
	// fmt.Printf("GetSnapshot -> %#v\n", snp)
	// 需要处理发生错误时, goroutine退出
	cancelWatcher := true // 标记是否取消监听router等信息的变化
	changes := make(chan interface{}, changesBufferSize)
	s.watcherCancelC = make(chan struct{})
	s.watcherErrorC = make(chan struct{})
	// 控制watching的开启
	newRouterC := make(chan bool)
	// s.watcherWg关联开启的goroutine
	s.watcherWg.Add(1)
	go func() {
		defer s.watcherWg.Done() // 执行顺序 2
		defer close(changes)     // 执行顺序 1
		// 优化, 在snapshot获取后, 创建路由表xrouter.New(snp)成功后, 才进行watching, 否则不进行watching, 直接退出goroutine
		b := <-newRouterC
		if !b {
			return
		}
		// fmt.Println("start watching")
		if err := s.ng.Subscribe(changes, snp.Index+1, s.watcherCancelC); err != nil {
			fmt.Printf("[s.ng.Subscribe] watcher failure: '%v'\n", err)
			s.watcherErrorC <- struct{}{}
			return
		}
		// 发信息取消watch, close(s.watcherCancelC)
		fmt.Println("[s.ng.Subscribe] watcher shutdown")
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
	// negroni for gateway
	n := negroni.New()
	// recovery
	n.Use(negroni.NewRecovery())
	// context
	n.UseFunc(xwaymw.XWayContext(xwaymw.ContextMWOption{Registry: s.registry}))
	// router
	r := xrouter.New(snp, s.registry, newRouterC)
	s.router = r.(router.Router)
	s.registry.SetRouter(s.router)
	n.Use(r)
	// proxy
	tr := &http.Transport{
		// Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: s.options.ProxyConnKeepAlive,
			DualStack: true,
		}).DialContext,
		// ResponseHeaderTimeout: 60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          s.options.ProxyMaxIdleConns, // Zero means no limit.
		MaxIdleConnsPerHost:   s.options.ProxyMaxIdleConnsPerHost,
		IdleConnTimeout:       s.options.ProxyIdleConnTimeout,
	}
	s.registry.SetTransport(tr)
	p, err := proxy.NewDo(tr)
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
	if s.options.PidPath != "" {
		ioutil.WriteFile(s.options.PidPath, []byte(fmt.Sprint(os.Getpid())), 0644)
	}

	if err := s.initDB(); err != nil {
		return fmt.Errorf("initDB failure: %v", err)
	}
	fmt.Println("[initDB success]")

	if s.options.EnableMQ {
		if err := s.initMQ(); err != nil {
			return fmt.Errorf("initMQ failure: %v", err)
		}
		fmt.Println("[initMQ success]")
	}

	if err := s.initEngine(); err != nil {
		return fmt.Errorf("initEngine failure: %v", err)
	}
	fmt.Println("[initEngine success]")

	if err := s.initProxy(); err != nil {
		return fmt.Errorf("initProxy failure: %v", err)
	}
	fmt.Println("[initProxy success]")

	// 放最后
	s.initLogger()

	return nil
}

func (s *Service) startAPIServer() {
	// TODO: 优化, 处理goroutine的安全退出, 且要在*Service的数据初始化完毕后才启动
	go func() {
		router := mux.NewRouter()
		api.InitProxyController(s.ng, nil, router)
		address := s.options.ApiInterface + ":" + strconv.Itoa(s.options.ApiPort)
		fmt.Printf("[api server] listening on %v\n", address)
		log.Fatal(http.ListenAndServe(address, router))
	}()
}

func (s *Service) startGWServer() {
	address := s.options.Interface + ":" + strconv.Itoa(s.options.Port)
	l := log.New(os.Stdout, "[gateway server] ", 0)
	l.Printf("listening on %s", address)
	l.Fatal(http.ListenAndServe(address, s.ngiSvc))
}

// Run ...
func Run(registry *plugin.Registry) error {
	defer func() {
		// 启动发生错误, 或程序退出时的处理
		registry.Release()
	}()

	fmt.Println("[app running......]")
	// 加载配置
	options, err := ParseCommandLine()
	if err != nil {
		return fmt.Errorf("failed to parse command line: %s", err)
	}
	// fmt.Printf("加载配置options: %v\n", options)
	// fmt.Println("初始化......")
	registry.SetSvcOptions(options)

	// appLogger.Info("Starting......")
	s := NewService(options, registry)
	if err := s.load(); err != nil {
		return fmt.Errorf("service.load failure: %s", err)
	}
	registry.SetSvc(s)

	// start server
	fmt.Println("[start server]")
	// api server
	s.startAPIServer()
	// gateway server
	s.startGWServer()

	return nil
}
