package xwaymw

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/urfave/negroni"

	"xway/context"
	"xway/plugin"
)

type ContextMWOption struct {
	Key      xwaycontext.ContextKey
	Registry *plugin.Registry
}

// func DefaultXWayContext() negroni.HandlerFunc {
// 	return XWayContext(ContextMWOption{})
// }

// XWayContext ...
func XWayContext(opt ContextMWOption) negroni.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		xwayCtx := xwaycontext.XWayContext{Map: map[interface{}]interface{}{}, Registry: opt.Registry}
		xwayCtx.RequestId = strconv.FormatInt(time.Now().UnixNano(), 10)
		w.Header().Set("x-xway-request-id", xwayCtx.RequestId)
		ctx := context.WithValue(r.Context(), opt.Key, &xwayCtx)
		// ctx := context.WithValue(r.Context(), xwaycontext.ContextKey{Key: "xway"}, &xwayCtx)
		// ctx = context.WithValue(ctx, xwaycontext.ContextKey{Key: "cwg"}, map[interface{}]interface{}{"mykey": "cwg"})
		mr := r.WithContext(ctx)
		next(w, mr)
	}
}
