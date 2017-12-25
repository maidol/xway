package xwaycontext

import (
	"context"
	"errors"
	"net/http"
)

type ContextKey struct {
	Key string
}

type XWayContext struct {
	originalRequest *http.Request
	Map             map[interface{}]interface{}
	UserName        string
	Password        string
}

func (xc *XWayContext) GetOriginalRequest() *http.Request {
	return xc.originalRequest
}

func (xc *XWayContext) SetOriginalRequest(val *http.Request) error {
	if xc.originalRequest != nil {
		return errors.New("originalRequest had been set")
	}
	xc.originalRequest = val
	return nil
}

func GetXWayContext(ctx context.Context, xwayKey string) *XWayContext {
	return ctx.Value(ContextKey{Key: xwayKey}).(*XWayContext)
}

func SetXWayContext(ctx context.Context, xwayKey string, value *XWayContext) context.Context {
	return context.WithValue(ctx, ContextKey{Key: xwayKey}, value)
}
