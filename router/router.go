package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/urfave/negroni"

	xerror "xway/error"
)

// Router ...
type Router struct {
}

func (rt *Router) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// 处理路由匹配
	fmt.Printf("%v\n", r.URL)
	if !rt.IsValid(r.URL.Path) {
		e := xerror.NewRequestError(xerror.Normal, xerror.EcodeRouteNotFound, "")
		e.Write(rw)
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
