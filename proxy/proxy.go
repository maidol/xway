package proxy

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

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
		u, _ := url.Parse("http://192.168.1.46:800" + r.URL.String())
		// u, _ := url.Parse("https://eapi.ciwong.com" + r.URL.String())
		// u := testutils.ParseURI("https://eapi.ciwong.com" + r.URL.String())
		fmt.Printf("forward to url -->> %v, %v\n", u.Host, u)

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
			// fmt.Printf("%v %v\n", k, v)
			for _, s := range v {
				w.Header().Add(k, s)
			}
		}

		// body = append(body, ([]byte("abc"))...)
		// w.Header().Set("Content-Length", string(len(body)))

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
		r.URL = testutils.ParseURI("https://eapi.ciwong.com")
		fwd.ServeHTTP(w, r)
	})
	return redirect, nil
}
