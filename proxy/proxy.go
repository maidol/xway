package proxy

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/testutils"

	"xway/context"
	en "xway/engine"
	xemun "xway/enum"
	xerror "xway/error"
)

// NewDo ...
func NewDo() (http.HandlerFunc, error) {
	tr := &http.Transport{
		// Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 30 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		// MaxIdleConns:          0, // Zero means no limit.
		MaxIdleConnsPerHost: 1000,
	}

	client := &http.Client{Transport: tr}

	pr := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// u, err := url.Parse("http://192.168.2.102:8708" + r.URL.String())

		xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
		matchRouteFrontend := xwayCtx.Map["matchRouteFrontend"].(*en.Frontend)
		forwardURL := xwayCtx.Map["forwardURL"].(string)
		forwardURL = strings.Replace(forwardURL, strings.ToLower(strings.TrimRight(matchRouteFrontend.RouteUrl, "/")), strings.ToLower(strings.TrimRight(matchRouteFrontend.ForwardURL, "/")), 1)
		u, err := url.Parse("http://" + matchRouteFrontend.RedirectHost + forwardURL)

		fmt.Printf("[MW:proxy] -> url forward to: %v, %v\n", u.Host, u)
		// TODO: 优化, 异步日志
		// pool := xgrpool.Default()
		// pool.JobQueue <- func() {
		// 	fmt.Printf("-> url forward to: %v, %v\n", u.Host, u)
		// 	fmt.Printf("-> url forward to: %v, %v\n", u.Host, u)
		// }

		if err != nil {
			fmt.Printf("url.Parse err %v\n", err)
			er := xerror.NewRequestError(xemun.RetProxyError, xemun.ECodeProxyFailed, err.Error())
			er.Write(w)
			return
		}

		outReq := new(http.Request) // or use outReq := r.WithContext(r.Context())
		*outReq = *r                // includes shallow copies of maps, but we handle this in Director
		outReq.RequestURI = ""      // Request.RequestURI can't be set in client requests.
		outReq.URL = u
		outReq.Host = u.Host
		// TODO: 需要优化, 处理outReq.Header和outReq.Close, 保持http.client连接 (可参考net/http/httputil/reverseproxy.go ServeHTTP)
		// fmt.Printf("outReq.Close %v, r.Close %v\n", outReq.Close, r.Close)
		// r.WithContext(r.Context())/new(http.Request) 的创建方式, outReq.Close默认值无效, 连接没有复用
		// outReq.Close = false 必须强制重设, 具体原因待探讨
		outReq.Close = false

		// outReq, _ := http.NewRequest("GET", "http://192.168.2.102:8708"+r.URL.String(), nil)

		resp, err := client.Do(outReq)
		if err != nil {
			fmt.Printf("client.Do err %v\n", err)
			er := xerror.NewRequestError(xemun.RetProxyError, xemun.ECodeProxyFailed, err.Error())
			er.Write(w)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("ioutil.ReadAll %v\n", err)
			er := xerror.NewRequestError(xemun.RetProxyError, xemun.ECodeInternal, err.Error())
			er.Write(w)
			return
		}

		for k, v := range resp.Header {
			for _, s := range v {
				w.Header().Add(k, s)
			}
		}

		// body = append(body, ([]byte("abc"))...)
		// w.Header().Set("Content-Length", string(len(body)))
		// fmt.Printf("-> response data: %v\n", string(body))
		w.Write(body)
	})
	return pr, nil
}

// New ...
func New() (http.HandlerFunc, error) {
	tr := &http.Transport{
		// Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 30 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		// MaxIdleConns:          0, // Zero means no limit.
		MaxIdleConnsPerHost: 1000,
	}

	fwd, err := forward.New(forward.RoundTripper(tr))

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	redirect := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
		// fmt.Printf("%v\n", xwayCtx.UserName)

		r.URL = testutils.ParseURI("http://192.168.2.102:8708")

		fwd.ServeHTTP(w, r)
	})
	return redirect, nil
}
