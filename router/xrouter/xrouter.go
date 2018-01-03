package xrouter

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/urfave/negroni"

	"xway/context"
	en "xway/engine"
	"xway/enum"
	xerror "xway/error"
)

// Router ...
type Router struct {
	// snp       *en.Snapshot
	frontends []en.Frontend
}

func (rt *Router) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// 处理路由匹配
	fmt.Printf("[MW:xrouter] -> url router for: r.Host %v, r.URL %v\n", r.Host, r.URL)
	match, fe := rt.IsValid(r)
	if !match {
		DefaultNotFound(rw, r)
		return
	}
	// TODO: match中间件处理
	if fe == nil {
		fmt.Printf("match frontend %+v\n", fe)
	}

	next(rw, r)
}

// Handle ...
func (rt *Router) Handle(string, http.Handler) error {
	return nil
}

// frontendSlice 排序
type frontendSlice []en.Frontend

func (fs frontendSlice) Len() int {
	return len(fs)
}

func (fs frontendSlice) Swap(i, j int) {
	fs[i], fs[j] = fs[j], fs[i]
}

func (fs frontendSlice) Less(i, j int) bool {
	// 降序
	return len(strings.Split(strings.Trim(fs[j].RouteUrl, "/"), "/")) < len(strings.Split(strings.Trim(fs[i].RouteUrl, "/"), "/"))
}

// IsValid ...
func (rt *Router) IsValid(r *http.Request) (bool, *en.Frontend) {
	if !strings.HasPrefix(r.URL.Path, "/gateway/") {
		return false, nil
	}

	forwardURL := strings.Replace(r.URL.Path, "/gateway/", "/", 1)
	var matchers []en.Frontend
	// TODO: 优化匹配逻辑
	for _, v := range rt.frontends {
		if v.DomainHost == r.Host && strings.HasPrefix(forwardURL, v.RouteUrl) {
			matchers = append(matchers, v)
		}
	}

	if len(matchers) <= 0 {
		return false, nil
	}

	sort.Sort(frontendSlice(matchers))
	res := matchers[0]

	xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
	xwayCtx.Map["matchRouteFrontend"] = &res

	return true, &res
}

// New ...
func New(snp *en.Snapshot) negroni.Handler {
	var frontends []en.Frontend
	for _, v := range snp.FrontendSpecs {
		frontends = append(frontends, v.Frontend)
	}
	return &Router{
		// snp:       snp,
		frontends: frontends,
	}
}

// DefaultNotFound is an HTTP handler that returns simple 404 Not Found response.
var DefaultNotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	e := xerror.NewRequestError(enum.RetProxyError, enum.ECodeRouteNotFound, "代理路由异常")
	e.Write(w)
})
