package main

import (
	"os"
	"runtime"

	logrus_logstash "github.com/bshuster-repo/logrus-logstash-hook"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"xway/middleware"
	"xway/proxy"
	"xway/router"
)

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

func main() {
	runtime.GOMAXPROCS(4)

	appLogger.Info("初始化......")

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
	p, _ := proxy.New()
	n.UseHandlerFunc(p)

	n.Run(":9799")
}
