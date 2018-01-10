package authtoken

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"xway/context"
	"xway/enum"
	xerror "xway/error"

	"github.com/garyburd/redigo/redis"
	"github.com/mholt/binding"
	"github.com/urfave/negroni"
)

type AuthToken struct {
	opt Options
}

type Options struct {
}

type QueryData struct {
	ClientId string
	Token    string
}

// func (qd *QueryData) Validate(req *http.Request) error {
// 	if qd.Token == "" {
// 		return binding.Errors{
// 			binding.NewError([]string{"accessToken"}, "EmptyError", "accessToken 不能为空"),
// 		}
// 	}
// 	return nil
// }

func (qd *QueryData) FieldMap(req *http.Request) binding.FieldMap {
	return binding.FieldMap{
		// &qd.ClientId: binding.Field{
		// 	Form:         "clientId",
		// 	Required:     true,
		// 	ErrorMessage: "clitenId不能为空",
		// },
		&qd.ClientId: "clientId",
		&qd.Token: binding.Field{
			Form:         "accessToken",
			Required:     true,
			ErrorMessage: "accessToken不能为空",
		},
	}
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
	qd := new(QueryData)
	if errs := binding.URL(r, qd); errs != nil {
		xwayCtx.Map["error"] = errs
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeParamsError, errs.Error())
		e.Write(rw)
		return
	}
	p := xwayCtx.Registry.GetRedisPool()
	// fmt.Println(p.ActiveCount(), p.IdleCount(), p.Stats())
	rdc := p.Get()
	defer func() {
		// 重要: 释放客户端
		if err := rdc.Close(); err != nil {
			// TODO: 处理错误
			fmt.Printf("[AuthToken.ServeHTTP] rdc.Close err: %v\n", err)
		}
	}()
	// 读取token, 验证权限
	tk := "cw:gateway:token:" + qd.Token
	v, err := rdc.Do("HGETALL", tk)
	if err != nil {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, err.Error())
		xwayCtx.Map["error"] = e
		e.Write(rw)
		return
	}
	m, err := redis.StringMap(v, err)
	if err != nil {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, err.Error())
		xwayCtx.Map["error"] = e
		e.Write(rw)
		return
	}
	if m == nil || len(m) == 0 {
		err := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeUnauthorized, "未找到有效token")
		xwayCtx.Map["error"] = err
		err.Write(rw)
		return
	}
	expireDate, err := strconv.Atoi(m["expireDate"])
	if err != nil {
		err := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, err.Error()+` [strconv.Atoi(m["expireDate"])转换失败]`)
		xwayCtx.Map["error"] = err
		err.Write(rw)
		return
	}
	if int64(expireDate) < time.Now().Unix() {
		err := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeUnauthorized, "token已过期")
		xwayCtx.Map["error"] = err
		err.Write(rw)
		return
	}
	r.SetBasicAuth(m["userId"], "123456")
	next(rw, r)
}
