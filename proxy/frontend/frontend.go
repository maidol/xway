package frontend

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"xway/context"
	en "xway/engine"
	"xway/plugin"

	"github.com/urfave/negroni"
)

// T represents a frontend instance.
// 聚合前端的所有中间件, 每个前端有特定的处理配置
type T struct {
	cfg     en.Frontend
	handler http.Handler // 聚合前端的所有中间件
}

// New ...
func New(cfg en.Frontend, registry *plugin.Registry) *T {
	fe := T{
		cfg: cfg,
	}
	// TODO: 加载前端中间件
	ngi := negroni.New()
	switch cfg.Type {
	case en.HTTP:
		// TODO: 根据定义的配置创建中间件实例
		config := cfg.Config.(en.HTTPFrontendSettings)
		for _, mwcfg := range config.Auth {
			spec := registry.GetMW(mwcfg)
			if spec != nil {
				ngi.Use(spec.MW(mwcfg))
			}
		}

	}
	fe.handler = ngi
	return &fe
}

// ServeHTTP implements negroni.Handler.
func (fe *T) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	fe.handler.ServeHTTP(w, r)
	if hasError(r) {
		// 中间件错误验证不通过, 返回
		return
	}
	next(w, r)
}

func hasError(r *http.Request) bool {
	xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
	err := xwayCtx.Map["error"]
	if err != nil {
		// TODO: 需要优化错误日志
		e, ok := err.(error)
		if ok {
			tk := "cw:gateway:err:" + xwayCtx.RequestId
			msg := fmt.Sprintf("[MW:frontend:hasError] %v: %v", r.URL, e.Error())
			logrus.WithFields(logrus.Fields{"key": tk}).Error(msg)
		}
		return true
	}
	return false
}

// func hasError(r *http.Request) bool {
// 	xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
// 	err := xwayCtx.Map["error"]
// 	if err != nil {
// 		// TODO: 需要优化错误日志
// 		e, ok := err.(error)
// 		if ok {
// 			tk := "cw:gateway:err:" + xwayCtx.RequestId
// 			msg:="[MW:frontend:hasError] " + e.Error()
// 			l := xwayCtx.Registry.GetRedisPool().Get()
// 			defer l.Close()
// 			_, re := l.Do("SET", tk, msg)
// 			if re != nil {
// 				fmt.Println("[MW:frontend:hasError] redis l.Do(SET) err:", re)
// 			}
// 		}
// 		return true
// 	}
// 	return false
// }
