package proxy

import (
	"log"
	"net"
	"net/http"
	"time"

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
		r.URL = testutils.ParseURI("http://192.168.2.101:8708")
		fwd.ServeHTTP(w, r)
	})
	return redirect, nil
}
