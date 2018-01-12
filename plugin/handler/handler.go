package handler

import (
	"net/http"
	"xway/context"
	"xway/enum"
	xerror "xway/error"
)

type ErrorHandler interface {
	RequestError(w http.ResponseWriter, req *http.Request, err error)
}

var DefaultErrorHandler ErrorHandler = &StdErrorHandler{}

type StdErrorHandler struct {
}

func (s *StdErrorHandler) RequestError(w http.ResponseWriter, req *http.Request, err error) {
	HandleRequestError(w, req, err)
}

// HandleRequestError process err
// 中间件产生的错误统一由HandleRequestError处理
// xwayCtx.Map["error"] = err 用以判断, 阻断下一层的处理(例如路由xrouter下一层的代理proxy)
func HandleRequestError(w http.ResponseWriter, req *http.Request, err error) {
	xwayCtx := xwaycontext.DefaultXWayContext(req.Context())
	xwayCtx.Map["error"] = err
	if e, ok := err.(*xerror.Error); ok {
		e.Write(w)
		return
	}
	e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, err.Error())
	e.Write(w)
}

// type ErrorHandlerFunc func(http.ResponseWriter, *http.Request, error)

// // ServeHTTP calls f(w, r).
// func (f ErrorHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request, err error) {
// 	f(w, r, err)
// }
