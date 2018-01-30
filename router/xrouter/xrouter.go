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
	"xway/plugin"
	"xway/proxy/frontend"
)

// Router ...
type Router struct {
	// snp       *en.Snapshot
	registry      *plugin.Registry
	frontendMap   map[string]*en.Frontend
	frontendMWMap map[string]*frontend.T
	frontends     []*en.Frontend
}

func (rt *Router) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// TODO: x-way等个性化的头信息元数据可考虑移到在一个中间件配置
	rw.Header().Add("x-xway", "beta-1.0.0")
	// 处理路由匹配
	// fmt.Printf("[MW:xrouter] -> url router for: r.Host %v, r.URL %v\n", r.Host, r.URL)
	match, fe := rt.IsMatch(r)
	if !match {
		DefaultNotFound(rw, r)
		return
	}

	// TODO: match中间件处理
	// fmt.Printf("match frontend %+v\n", fe)
	f := fe.(*en.Frontend)
	ff := rt.frontendMWMap[f.RouteId]
	ff.ServeHTTP(rw, r, next)
}

// Remove ...
func (rt *Router) Remove(f interface{}) error {
	var frontends []*en.Frontend
	var frontendsTemp []en.Frontend
	rid := f.(string)
	fmt.Printf("[xrouter.Remove] frontend %v\n", rid)
	delete(rt.frontendMap, rid)
	delete(rt.frontendMWMap, rid)

	// 重新排序(重载路由表)
	for _, v := range rt.frontendMap {
		frontendsTemp = append(frontendsTemp, *v)
	}
	sort.Sort(frontendSlice(frontendsTemp))

	fmt.Printf("[重新加载路由表]\n")
	for _, v := range frontendsTemp {
		if v.RouteId == "" || v.Status != 0 {
			delete(rt.frontendMap, v.RouteId)
			delete(rt.frontendMWMap, v.RouteId)
			continue
		}
		fmt.Printf("[加载路由] %v\n", v)
		// 变量v在第一次定义时, 地址是确定的(默认不变) fmt.Printf("%p %v\n", &v, &v)
		// 防止每次传递变量地址(&v)一样导致的bug, 需赋值f:=v
		// append(frontends, &v) => append(frontends, &f)
		f := v
		frontends = append(frontends, &f)
	}
	fmt.Printf("[重载路由表完成]\n")

	rt.frontends = frontends
	return nil
}

// Handle add/update
func (rt *Router) Handle(f interface{}) error {
	// add/update
	var frontends []*en.Frontend
	var frontendsTemp []en.Frontend
	fr := f.(en.Frontend)
	fmt.Printf("[xrouter.Handle] frontend %v\n", fr)
	rt.frontendMap[fr.RouteId] = &fr
	rt.frontendMWMap[fr.RouteId] = frontend.New(fr, rt.registry)

	// 重新排序(重载路由表)
	for _, v := range rt.frontendMap {
		frontendsTemp = append(frontendsTemp, *v)
	}
	sort.Sort(frontendSlice(frontendsTemp))

	fmt.Printf("[重新加载路由表]\n")
	for _, v := range frontendsTemp {
		if v.RouteId == "" || v.Status != 0 {
			delete(rt.frontendMap, v.RouteId)
			delete(rt.frontendMWMap, v.RouteId)
			continue
		}
		fmt.Printf("[加载路由] %v\n", v)
		// 变量v在第一次定义时, 地址是确定的(默认不变) fmt.Printf("%p %v\n", &v, &v)
		// 防止每次传递变量地址(&v)一样导致的bug, 需赋值f:=v
		// append(frontends, &v) => append(frontends, &f)
		f := v
		frontends = append(frontends, &f)
	}
	fmt.Printf("[重载路由表完成]\n")

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

func (rt *Router) IsValid(expr string) bool {
	return true
}

// IsMatch 验证路由匹配
func (rt *Router) IsMatch(r *http.Request) (bool, interface{}) {
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
		// forwardURL和v.RouteUrl末尾必须带"/"进行匹配, 若不带"/", /v5/userinfo/create 会匹配到 /v5/user, /v5/useri, /v5/userin, ... , /v5/userinfo, /v5/userinfo/create, /v5/userinfo/created, ...
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
func New(snp *en.Snapshot, registry *plugin.Registry, newRouterC chan bool) negroni.Handler {
	defer func() {
		// fmt.Println("close(newRouterC)")
		close(newRouterC)
	}()

	var frontends []*en.Frontend
	var frontendsTemp []en.Frontend
	frontendMap := make(map[string]*en.Frontend)
	frontendMWMap := make(map[string]*frontend.T)

	// 排序
	for _, v := range snp.FrontendSpecs {
		f := v.Frontend
		frontendsTemp = append(frontendsTemp, f)
	}
	sort.Sort(frontendSlice(frontendsTemp))

	fmt.Printf("[开始加载路由表]\n")
	for _, v := range frontendsTemp {
		if v.RouteId == "" || v.Status != 0 {
			continue
		}
		fmt.Printf("[加载路由] %v\n", v)
		// 变量v在第一次定义时, 地址是确定的(默认不变) fmt.Printf("%p %v\n", &v, &v)
		// 防止每次传递变量地址(&v)一样导致的bug, 需赋值f:=v
		// append(frontends, &v) => append(frontends, &f)
		f := v
		// 加载前端的处理中间件
		fe := frontend.New(f, registry)
		frontendMWMap[f.RouteId] = fe
		frontends = append(frontends, &f)
		frontendMap[f.RouteId] = &f
	}
	fmt.Printf("[加载路由表完成]\n")
	// 加载成功
	newRouterC <- true
	return &Router{
		// snp:       snp,
		registry:      registry,
		frontendMap:   frontendMap,
		frontendMWMap: frontendMWMap,
		frontends:     frontends,
	}
}

// DefaultNotFound is an HTTP handler that returns simple 404 Not Found response.
var DefaultNotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	e := xerror.NewRequestError(enum.RetProxyError, enum.ECodeRouteNotFound, "代理路由异常")
	e.Write(w)
})
