package authtoken

import (
	"net/http"
	"xway/context"
	"xway/enum"
	xerror "xway/error"

	"github.com/urfave/negroni"
)

type AuthToken struct {
	opt Options
}

type Options struct {
}

// New ...
// 创建中间件实例
func New(opt interface{}) negroni.Handler {
	o, ok := opt.(Options)
	if !ok {
		o = Options{}
	}
	return &AuthToken{
		opt: o,
	}
}

func (at *AuthToken) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
	_, pwd, ok := r.BasicAuth()
	// fmt.Printf("Auth %+v %v\n", a, ok)
	if !ok || pwd != "123456" {
		// TODO: 产生错误退出
		err := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeUnauthorized, "no auth")
		xwayCtx.Map["error"] = err
		err.Write(rw)
		return
	}
	next(rw, r)
}
