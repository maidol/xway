package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/urfave/negroni"

	"xway/enum"
	xerror "xway/error"
)

// Router ...
type Router struct {
}

func (rt *Router) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// 处理路由匹配
	fmt.Printf("-> url router for: %v, %v\n", r.Host, r.URL)
	if !rt.IsValid(r.URL.Path) {
		DefaultNotFound(rw, r)
		return
	}
	next(rw, r)
}

// Handle ...
func (rt *Router) Handle(string, http.Handler) error {
	return nil
}

// IsValid ...
func (rt *Router) IsValid(path string) bool {
	return strings.HasPrefix(path, "/gateway/")
}

// New ...
func New() negroni.Handler {
	return &Router{}
}

// DefaultNotFound is an HTTP handler that returns simple 404 Not Found response.
var DefaultNotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	e := xerror.NewRequestError(enum.RetProxyError, enum.ECodeRouteNotFound, "代理路由异常")
	e.Write(w)
})
