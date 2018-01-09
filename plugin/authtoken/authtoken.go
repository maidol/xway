package authtoken

import (
	"fmt"
	"net/http"
	"time"
	"xway/context"
	"xway/enum"
	xerror "xway/error"

	rd "github.com/garyburd/redigo/redis"
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
	p := xwayCtx.Registry.GetRedisPool()
	// fmt.Println(p.ActiveCount(), p.IdleCount(), p.Stats())
	rdc := p.Get()
	defer func() {
		if err := rdc.Close(); err != nil {
			// TODO: 处理错误
			fmt.Printf("[AuthToken.ServeHTTP] rdc.Close err: %v\n", err)
		}
	}()
	_, err := rdc.Do("SET", "GO_AuthToken", time.Now().String())
	if err != nil {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, err.Error())
		xwayCtx.Map["error"] = e
		e.Write(rw)
		return
	}
	v, err := rd.String(rdc.Do("GET", "GO_AuthToken"))
	if err != nil {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, err.Error())
		xwayCtx.Map["error"] = e
		e.Write(rw)
		return
	}
	if v != "" {
	}
	// fmt.Printf("GO_AuthToken %v\n", v)
	// TODO: 读取token
	_, pwd, ok := r.BasicAuth()
	if !ok || pwd != "123456" {
		// TODO: 产生错误退出
		err := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeUnauthorized, "no auth")
		xwayCtx.Map["error"] = err
		err.Write(rw)
		return
	}
	next(rw, r)
}
