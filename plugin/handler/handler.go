package handler

import (
	"net/http"
	"xway/context"
	"xway/enum"
	xerror "xway/error"
)

type ErrorHandler interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request, err error)
}

var DefaultHandler ErrorHandler = &StdHandler{}

type StdHandler struct {
}

// errorReqHandler process err
// 前端中间件产生的错误必须统一由errorReHandler处理, xwayCtx.Map["error"] = err用以阻断路由xrouter下一层的处理
func errorReqHandler(rw http.ResponseWriter, r *http.Request, err *xerror.Error) {
	xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
	xwayCtx.Map["error"] = err
	err.Write(rw)
}

func (s *StdHandler) ServeHTTP(w http.ResponseWriter, req *http.Request, err error) {
	xwayCtx := xwaycontext.DefaultXWayContext(req.Context())
	xwayCtx.Map["error"] = err
	if e, ok := err.(*xerror.Error); ok {
		e.Write(w)
		return
	}
	e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, err.Error())
	e.Write(w)
}

type ErrorHandlerFunc func(http.ResponseWriter, *http.Request, error)

// ServeHTTP calls f(w, r).
func (f ErrorHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request, err error) {
	f(w, r, err)
}
