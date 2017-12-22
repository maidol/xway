package main

import (
	"net/http"
	"os"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"cw-gateway/proxy"
	"cw-gateway/router"
)

type resp struct {
	data []byte
	rw   http.ResponseWriter
}

// var logger logrus.Logger
var appLogger *logrus.Entry

func init() {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel
	logger.Formatter = new(logrus.JSONFormatter)
	logger.Out = os.Stdout

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

	// router
	r := router.New()

	// proxy
	p, _ := proxy.New()

	n.Use(r)
	n.UseHandlerFunc(p)

	n.Run(":8799")
}
