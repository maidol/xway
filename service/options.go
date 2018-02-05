package service

import (
	"flag"
	"fmt"
	"strings"
	"time"

	logrus "github.com/sirupsen/logrus"
)

type Options struct {
	ApiPort      int
	ApiInterface string

	PidPath   string
	Port      int
	Interface string

	Log          string
	LogSeverity  SeverityFlag
	LogFormatter logrus.Formatter

	EtcdApiVersion int
	EtcdNodes      listOptions
	EtcdKey        string

	DBHost     string
	DBUserName string
	DBPassword string
	DBMaxOpen  int
	DBMaxIdle  int
	// DBConnMaxLifetime second
	DBConnMaxLifetime time.Duration

	GatewayDBName string

	RedisHost      string
	RedisPassword  string
	RedisDB        int
	RedisWait      bool
	RedisMaxActive int
	RedisMaxIdle   int
	// RedisConnIdleTimeout second
	RedisConnIdleTimeout time.Duration

	ProxyMaxIdleConns        int
	ProxyMaxIdleConnsPerHost int
	// ProxyConnKeepAlive second
	ProxyConnKeepAlive   time.Duration
	ProxyIdleConnTimeout time.Duration

	EnableMQ        bool
	KafkaConfigPath string
	KafkaAK         string
	KafkaPassword   string

	Topic string
}

type SeverityFlag struct {
	S logrus.Level
}

func (s *SeverityFlag) Get() interface{} {
	return &s.S
}

func (s *SeverityFlag) Set(value string) error {
	sev, err := logrus.ParseLevel(strings.ToLower(value))
	if err != nil {
		return err
	}
	s.S = sev
	return nil
}

func (s *SeverityFlag) String() string {
	return s.S.String()
}

type listOptions []string

func (o *listOptions) String() string {
	return fmt.Sprint(*o)
}

func (o *listOptions) Set(value string) error {
	*o = append(*o, value)
	return nil
}

func validateOptions(o Options) (Options, error) {
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "readTimeout" {
			fmt.Printf("!!!!!! WARN: Using deprecated readTimeout flag, use serverReadTimeout instead\n\n")
		}
		if f.Name == "writeTimeout" {
			fmt.Printf("!!!!!! WARN: Using deprecated writeTimeout flag, use serverWriteTimeout instead\n\n")
		}
	})
	return o, nil
}

// ParseCommandLine ...
func ParseCommandLine() (options Options, err error) {
	flag.Var(&options.EtcdNodes, "etcd", "Etcd discovery service API endpoints")
	flag.IntVar(&options.EtcdApiVersion, "etcdApiVer", 3, "Etcd Client API version (When 2, use Etcd 2.x API. All other values default to v3.x)")
	flag.StringVar(&options.EtcdKey, "etcdKey", "xway", "Etcd key for storing configuration")

	flag.StringVar(&options.PidPath, "pidPath", "", "Path to write PID file to")
	flag.IntVar(&options.Port, "port", 9799, "Port to listen on")
	flag.IntVar(&options.ApiPort, "apiPort", 9788, "Port to provide api on")

	flag.StringVar(&options.Interface, "interface", "", "Interface to bind to")
	flag.StringVar(&options.ApiInterface, "apiInterface", "127.0.0.1", "Interface to for API to bind to")

	flag.StringVar(&options.Log, "log", "console", "Logging to use (console, json, redis, kafka, syslog or logstash)")
	options.LogSeverity.S = logrus.ErrorLevel // default
	flag.Var(&options.LogSeverity, "logSeverity", "log at or above(debug,info,warn,error,fatal,panic) this level to the logging output(default >=error)")

	// db
	flag.StringVar(&options.DBHost, "dbHost", "127.0.0.1:3306", "db server")
	flag.StringVar(&options.DBUserName, "dbUserName", "", "db username")
	flag.StringVar(&options.DBPassword, "dbPassword", "", "db password")
	flag.IntVar(&options.DBMaxOpen, "dbMaxOpen", 0, "db maxopen")
	flag.IntVar(&options.DBMaxIdle, "dbMaxIdle", 1000, "db maxidle")
	flag.DurationVar(&options.DBConnMaxLifetime, "dbConnMaxLifetime", 60*time.Second, "db Conn MaxLifetime(second)")

	// gateway db
	flag.StringVar(&options.GatewayDBName, "gatewayDBName", "cw_api_gateway", "gateway dbname")

	//redis
	flag.StringVar(&options.RedisHost, "redisHost", "127.0.0.1:6379", "redis server")
	flag.StringVar(&options.RedisPassword, "redisPassword", "", "redis password")
	flag.IntVar(&options.RedisDB, "redisDB", 0, "redis db num")
	flag.BoolVar(&options.RedisWait, "redisWait", false, "redis db wait")
	flag.IntVar(&options.RedisMaxActive, "redisMaxActive", 0, "redis db maxactive")
	flag.IntVar(&options.RedisMaxIdle, "redisMaxIdle", 1000, "redis db maxidle")
	flag.DurationVar(&options.RedisConnIdleTimeout, "redisConnIdleTimeout", 240*time.Second, "redis db Conn IdleTimeout(second)")

	// proxy
	flag.IntVar(&options.ProxyMaxIdleConns, "proxyMaxIdleConns", 0, "proxy MaxIdleConns")
	flag.IntVar(&options.ProxyMaxIdleConnsPerHost, "proxyMaxIdleConnsPerHost", 1000, "proxy MaxIdleConnsPerHost")
	flag.DurationVar(&options.ProxyConnKeepAlive, "proxyConnKeepAlive", 30*time.Second, "proxy Conn KeepAlive(second)")
	flag.DurationVar(&options.ProxyIdleConnTimeout, "proxyIdleConnTimeout", 90*time.Second, "proxy IdleConnTimeout(second)")

	flag.BoolVar(&options.EnableMQ, "enablemq", false, "enable mq")
	// kafka
	flag.StringVar(&options.KafkaConfigPath, "kafkaConfigPath", "mq.json", "kafka config path")
	flag.StringVar(&options.KafkaAK, "kafkaAK", "", "kafka access key")
	flag.StringVar(&options.KafkaPassword, "kafkaPassword", "", "kafka password")

	flag.StringVar(&options.Topic, "topic", "xway", "log topic")

	flag.Parse()

	options, err = validateOptions(options)
	if err != nil {
		return options, err
	}
	return options, nil
}
