package proxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"xway/context"

	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/testutils"
)

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
		fmt.Println("proxy url -->>", originalRequest.Host, originalRequest.URL)
		r.URL = testutils.ParseURI("https://eapi.ciwong.com/gateway/")
		fmt.Println("forward url -->>", r.URL)
		fwd.ServeHTTP(w, r)
	})
	return redirect, nil
}
