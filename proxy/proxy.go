package proxy

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"xway/context"

	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/testutils"

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
		xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
		originalRequest := xwayCtx.GetOriginalRequest()
		fmt.Printf("-> url original request: %v, %v\n", originalRequest.Host, originalRequest.URL)

		u, err := url.Parse("http://192.168.2.102:8708" + r.URL.String())
		fmt.Printf("-> url forward to: %v, %v\n", u.Host, u)
		if err != nil {
			fmt.Printf("url.Parse err %v\n", err)
			er := xerror.NewRequestError(xemun.RetProxyError, xemun.ECodeProxyFailed, err.Error())
			er.Write(w)
			return
		}

		outReq := new(http.Request)
		*outReq = *r           // includes shallow copies of maps, but we handle this in Director
		outReq.RequestURI = "" // Request.RequestURI can't be set in client requests.
		outReq.URL = u
		outReq.Host = u.Host

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
		fmt.Printf("-> response data: %v\n", string(body))
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
		xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
		// cwgCtx := r.Context().Value(xwaycontext.ContextKey{Key: "cwg"})
		originalRequest := xwayCtx.GetOriginalRequest()
		fmt.Println("-> url proxy:", originalRequest.Host, originalRequest.URL)
		// r.URL = testutils.ParseURI("https://eapi.ciwong.com/gateway/")
		r.URL = testutils.ParseURI("http://192.168.2.102:8708")
		fmt.Println("-> url forward:", r.URL)
		fwd.ServeHTTP(w, r)
	})
	return redirect, nil
}
