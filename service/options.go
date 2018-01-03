package service

import (
	"flag"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Options struct {
	ApiPort      int
	ApiInterface string

	PidPath   string
	Port      int
	Interface string

	Log          string
	LogSeverity  SeverityFlag
	LogFormatter log.Formatter

	EtcdApiVersion int
	EtcdNodes      listOptions
	EtcdKey        string
}

type SeverityFlag struct {
	S log.Level
}

func (s *SeverityFlag) Get() interface{} {
	return &s.S
}

func (s *SeverityFlag) Set(value string) error {
	sev, err := log.ParseLevel(strings.ToLower(value))
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
	flag.StringVar(&options.ApiInterface, "apiInterface", "", "Interface to for API to bind to")

	flag.StringVar(&options.Log, "log", "console", "Logging to use (console, json, syslog or logstash)")
	options.LogSeverity.S = log.WarnLevel
	flag.Var(&options.LogSeverity, "logSeverity", "log at or above this level to the loggint output")

	flag.Parse()

	options, err = validateOptions(options)
	if err != nil {
		return options, err
	}
	return options, nil
}
