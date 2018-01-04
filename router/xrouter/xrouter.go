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
	frontendMap map[string]*en.Frontend
	frontends   []*en.Frontend
	// frontendMapTemp map[string]*en.Frontend
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

// Remove ...
func (rt *Router) Remove(f interface{}) error {
	var frontends []*en.Frontend
	var frontendsTemp []en.Frontend
	rid := f.(string)
	fmt.Printf("[xrouter.Remove] frontend %v\n", rid)
	delete(rt.frontendMap, rid)

	// 重新排序(重载路由表)
	for _, v := range rt.frontendMap {
		frontendsTemp = append(frontendsTemp, *v)
	}
	sort.Sort(frontendSlice(frontendsTemp))

	fmt.Printf("[重新加载路由表......]\n")
	for _, v := range frontendsTemp {
		fmt.Printf("[加载路由] %v\n", v)
		// 变量v在第一次定义时, 地址是确定的(默认不变) fmt.Printf("%p %v\n", &v, &v)
		// 防止每次传递变量地址(&v)一样导致的bug, 需赋值f:=v
		// append(frontends, &v) => append(frontends, &f)
		f := v
		frontends = append(frontends, &f)
	}

	rt.frontends = frontends
	return nil
}

// Handle ...
func (rt *Router) Handle(f interface{}) error {
	// add/update
	var frontends []*en.Frontend
	var frontendsTemp []en.Frontend
	fr := f.(en.Frontend)
	fmt.Printf("[xrouter.Handle] frontend %v\n", fr)
	rt.frontendMap[fr.RouteId] = &fr

	// 重新排序(重载路由表)
	for _, v := range rt.frontendMap {
		frontendsTemp = append(frontendsTemp, *v)
	}
	sort.Sort(frontendSlice(frontendsTemp))

	fmt.Printf("[重新加载路由表......]\n")
	for _, v := range frontendsTemp {
		fmt.Printf("[加载路由] %v\n", v)
		// 变量v在第一次定义时, 地址是确定的(默认不变) fmt.Printf("%p %v\n", &v, &v)
		// 防止每次传递变量地址(&v)一样导致的bug, 需赋值f:=v
		// append(frontends, &v) => append(frontends, &f)
		f := v
		frontends = append(frontends, &f)
	}

	rt.frontends = frontends
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
func (rt *Router) IsValid(r *http.Request) (bool, interface{}) {
	if !strings.HasPrefix(r.URL.Path, "/gateway/") {
		return false, nil
	}

	xwayCtx := xwaycontext.DefaultXWayContext(r.Context())

	forwardURL := strings.Replace(r.URL.Path, "/gateway/", "/", 1)
	forwardURL = strings.ToLower(strings.TrimRight(forwardURL, "/"))
	xwayCtx.Map["forwardURL"] = forwardURL // 传递的forwardURL末尾不带"/"

	forwardURL += "/"
	var matchers []en.Frontend
	// 优化匹配逻辑
	for _, v := range rt.frontends {
		rurl := strings.ToLower(strings.TrimRight(v.RouteUrl, "/")) + "/"
		if v.DomainHost == r.Host && strings.HasPrefix(forwardURL, rurl) {
			matchers = append(matchers, *v)
			break // 路由表已经在初始化和重载时预先做了排序("/"分割的字符串数组长度降序), 第一条匹配到的路由已是最优
		}
	}

	if len(matchers) <= 0 {
		return false, nil
	}

	// 优化, 整个路由表的优化排序放在路由初始化/重载时
	// sort.Sort(frontendSlice(matchers))
	// fmt.Printf("matchers %v\n", matchers)
	res := matchers[0]
	xwayCtx.Map["matchRouteFrontend"] = &res

	return true, &res
}

// New ...
func New(snp *en.Snapshot) negroni.Handler {
	var frontends []*en.Frontend
	var frontendsTemp []en.Frontend
	frontendMap := make(map[string]*en.Frontend)

	// 排序
	for _, v := range snp.FrontendSpecs {
		f := v.Frontend
		frontendsTemp = append(frontendsTemp, f)
	}
	sort.Sort(frontendSlice(frontendsTemp))

	fmt.Printf("[加载路由表......]\n")
	for _, v := range frontendsTemp {
		fmt.Printf("[加载路由] %v\n", v)
		// 变量v在第一次定义时, 地址是确定的(默认不变) fmt.Printf("%p %v\n", &v, &v)
		// 防止每次传递变量地址(&v)一样导致的bug, 需赋值f:=v
		// append(frontends, &v) => append(frontends, &f)
		f := v
		frontends = append(frontends, &f)
		frontendMap[f.RouteId] = &f
	}
	return &Router{
		// snp:       snp,
		frontendMap: frontendMap,
		frontends:   frontends,
	}
}

// DefaultNotFound is an HTTP handler that returns simple 404 Not Found response.
var DefaultNotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	e := xerror.NewRequestError(enum.RetProxyError, enum.ECodeRouteNotFound, "代理路由异常")
	e.Write(w)
})
