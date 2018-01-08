package frontend

import (
	"net/http"

	"xway/context"
	en "xway/engine"
	"xway/enum"
	xerror "xway/error"

	"github.com/urfave/negroni"
)

// T represents a frontend instance.
type T struct {
	cfg     en.Frontend
	handler http.Handler
}

// New ...
func New(cfg en.Frontend) *T {
	fe := T{
		cfg: cfg,
	}
	ngi := negroni.New()
	switch cfg.Type {
	case en.HTTP:
		config := cfg.Config.(en.HTTPFrontendSettings)
		for _, a := range config.Auth {
			ngi.UseFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
				xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
				_, pwd, ok := r.BasicAuth()
				if a != "" {

				}
				// fmt.Printf("Auth %+v %v\n", a, ok)
				if !ok || pwd != "123456" {
					// TODO: 产生错误退出
					err := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeUnauthorized, "no auth")
					xwayCtx.Map["error"] = err
					err.Write(rw)
					return
				}
				next(rw, r)
			})
		}

	}
	fe.handler = ngi
	return &fe
}

// ServeHTTP implements negroni.Handler.
func (fe *T) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	fe.handler.ServeHTTP(w, r)
	xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
	// next := xwayCtx.Map["next"].(http.HandlerFunc)
	err := xwayCtx.Map["error"]
	if err != nil {
		// TODO: 中间件验证不通过, 返回
		return
	}
	next(w, r)
}
