package xwaymw

import (
	"context"
	"net/http"

	"github.com/urfave/negroni"

	"xway/context"
)

type ContextMWOption struct {
	Key xwaycontext.ContextKey
}

func DefaultXWayContext() negroni.HandlerFunc {
	return XWayContext(ContextMWOption{})
}

// XWayContext ...
func XWayContext(opt ContextMWOption) negroni.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		xwayCtx := xwaycontext.XWayContext{Map: map[interface{}]interface{}{}}
		ctx := context.WithValue(r.Context(), opt.Key, &xwayCtx)
		// ctx := context.WithValue(r.Context(), xwaycontext.ContextKey{Key: "xway"}, &xwayCtx)
		// ctx = context.WithValue(ctx, xwaycontext.ContextKey{Key: "cwg"}, map[interface{}]interface{}{"mykey": "cwg"})
		mr := r.WithContext(ctx)
		next(w, mr)
	}
}
