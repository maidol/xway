// package forwarder implements http handler that forwards requests to remote server
// and serves back the response
// websocket proxying support based on https://github.com/yhat/wsutil
package forward

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/vulcand/oxy/utils"
)

// ReqRewriter can alter request headers and body
type ReqRewriter interface {
	Rewrite(r *http.Request)
}

type optSetter func(f *Forwarder) error

// PassHostHeader specifies if a client's Host header field should
// be delegated
func PassHostHeader(b bool) optSetter {
	return func(f *Forwarder) error {
		f.httpForwarder.passHost = b
		return nil
	}
}

// RoundTripper sets a new http.RoundTripper
// Forwarder will use http.DefaultTransport as a default round tripper
func RoundTripper(r http.RoundTripper) optSetter {
	return func(f *Forwarder) error {
		f.httpForwarder.roundTripper = r
		return nil
	}
}

// Rewriter defines a request rewriter for the HTTP forwarder
func Rewriter(r ReqRewriter) optSetter {
	return func(f *Forwarder) error {
		f.httpForwarder.rewriter = r
		return nil
	}
}

// PassHostHeader specifies if a client's Host header field should
// be delegated
func WebsocketTLSClientConfig(tcc *tls.Config) optSetter {
	return func(f *Forwarder) error {
		f.httpForwarder.tlsClientConfig = tcc
		return nil
	}
}

// ErrorHandler is a functional argument that sets error handler of the server
func ErrorHandler(h utils.ErrorHandler) optSetter {
	return func(f *Forwarder) error {
		f.errHandler = h
		return nil
	}
}

// Stream specifies if HTTP responses should be streamed.
func Stream(stream bool) optSetter {
	return func(f *Forwarder) error {
		f.stream = stream
		return nil
	}
}

// Logger defines the logger the forwarder will use.
//
// It defaults to logrus.StandardLogger(), the global logger used by logrus.
func Logger(l *log.Logger) optSetter {
	return func(f *Forwarder) error {
		f.log = l
		return nil
	}
}

func StateListener(stateListener UrlForwardingStateListener) optSetter {
	return func(f *Forwarder) error {
		f.stateListener = stateListener
		return nil
	}
}

func ResponseModifier(responseModifier func(*http.Response) error) optSetter {
	return func(f *Forwarder) error {
		f.httpForwarder.modifyResponse = responseModifier
		return nil
	}
}

func StreamingFlushInterval(flushInterval time.Duration) optSetter {
	return func(f *Forwarder) error {
		f.httpForwarder.flushInterval = flushInterval
		return nil
	}
}

type ErrorHandlingRoundTripper struct {
	http.RoundTripper
	errorHandler utils.ErrorHandler
}

func (rt ErrorHandlingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := rt.RoundTripper.RoundTrip(req)
	if err != nil {
		// We use the recorder from httptest because there isn't another `public` implementation of a recorder.
		recorder := httptest.NewRecorder()
		rt.errorHandler.ServeHTTP(recorder, req, err)
		res = recorder.Result()
		err = nil
	}
	return res, err
}

// Forwarder wraps two traffic forwarding implementations: HTTP and websockets.
// It decides based on the specified request which implementation to use
type Forwarder struct {
	*httpForwarder
	*handlerContext
	stateListener UrlForwardingStateListener
	stream        bool
}

// handlerContext defines a handler context for error reporting and logging
type handlerContext struct {
	errHandler utils.ErrorHandler
}

// httpForwarder is a handler that can reverse proxy
// HTTP traffic
type httpForwarder struct {
	roundTripper   http.RoundTripper
	rewriter       ReqRewriter
	passHost       bool
	flushInterval  time.Duration
	modifyResponse func(*http.Response) error

	tlsClientConfig *tls.Config

	log *log.Logger
}

const (
	defaultFlushInterval = time.Duration(100) * time.Millisecond
	StateConnected       = iota
	StateDisconnected
)

type UrlForwardingStateListener func(*url.URL, int)

// New creates an instance of Forwarder based on the provided list of configuration options
func New(setters ...optSetter) (*Forwarder, error) {
	f := &Forwarder{
		httpForwarder:  &httpForwarder{log: log.StandardLogger()},
		handlerContext: &handlerContext{},
	}
	for _, s := range setters {
		if err := s(f); err != nil {
			return nil, err
		}
	}

	if !f.stream {
		f.flushInterval = 0
	} else if f.flushInterval == 0 {
		f.flushInterval = defaultFlushInterval
	}

	if f.httpForwarder.rewriter == nil {
		h, err := os.Hostname()
		if err != nil {
			h = "localhost"
		}
		f.httpForwarder.rewriter = &HeaderRewriter{TrustForwardHeader: true, Hostname: h}
	}

	if f.httpForwarder.roundTripper == nil {
		f.httpForwarder.roundTripper = http.DefaultTransport
	}

	if f.errHandler == nil {
		f.errHandler = utils.DefaultHandler
	}

	if f.tlsClientConfig == nil {
		f.tlsClientConfig = f.httpForwarder.roundTripper.(*http.Transport).TLSClientConfig
	}

	f.httpForwarder.roundTripper = ErrorHandlingRoundTripper{
		RoundTripper: f.httpForwarder.roundTripper,
		errorHandler: f.errHandler,
	}

	return f, nil
}

// ServeHTTP decides which forwarder to use based on the specified
// request and delegates to the proper implementation
func (f *Forwarder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if f.log.Level >= log.DebugLevel {
		logEntry := f.log.WithField("Request", utils.DumpHttpRequest(req))
		logEntry.Debug("vulcand/oxy/forward: begin ServeHttp on request")
		defer logEntry.Debug("vulcand/oxy/forward: competed ServeHttp on request")
	}

	if f.stateListener != nil {
		f.stateListener(req.URL, StateConnected)
		defer f.stateListener(req.URL, StateDisconnected)
	}
	if IsWebsocketRequest(req) {
		f.httpForwarder.serveWebSocket(w, req, f.handlerContext)
	} else {
		f.httpForwarder.serveHTTP(w, req, f.handlerContext)
	}
}

// Modify the request to handle the target URL
func (f *httpForwarder) modifyRequest(outReq *http.Request, target *url.URL) {
	outReq.URL = utils.CopyURL(outReq.URL)
	outReq.URL.Scheme = target.Scheme
	outReq.URL.Host = target.Host
	outReq.URL.Opaque = outReq.RequestURI
	// raw query is already included in RequestURI, so ignore it to avoid dupes
	outReq.URL.RawQuery = ""
	// Do not pass client Host header unless optsetter PassHostHeader is set.
	if !f.passHost {
		outReq.Host = target.Host
	}

	outReq.Proto = "HTTP/1.1"
	outReq.ProtoMajor = 1
	outReq.ProtoMinor = 1

	if f.rewriter != nil {
		f.rewriter.Rewrite(outReq)
	}
}

// serveHTTP forwards websocket traffic
func (f *httpForwarder) serveWebSocket(w http.ResponseWriter, req *http.Request, ctx *handlerContext) {
	if f.log.Level >= log.DebugLevel {
		logEntry := f.log.WithField("Request", utils.DumpHttpRequest(req))
		logEntry.Debug("vulcand/oxy/forward/websocket: begin ServeHttp on request")
		defer logEntry.Debug("vulcand/oxy/forward/websocket: competed ServeHttp on request")
	}

	outReq := f.copyWebSocketRequest(req)

	dialer := websocket.DefaultDialer
	if outReq.URL.Scheme == "wss" && f.tlsClientConfig != nil {
		dialer.TLSClientConfig = f.tlsClientConfig.Clone()
		// WebSocket is only in http/1.1
		dialer.TLSClientConfig.NextProtos = []string{"http/1.1"}
	}
	targetConn, resp, err := dialer.Dial(outReq.URL.String(), outReq.Header)
	if err != nil {
		if resp == nil {
			ctx.errHandler.ServeHTTP(w, req, err)
		} else {
			log.Errorf("vulcand/oxy/forward/websocket: Error dialing %q: %v with resp: %d %s", outReq.Host, err, resp.StatusCode, resp.Status)
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				log.Errorf("vulcand/oxy/forward/websocket: %s can not be hijack", reflect.TypeOf(w))
				ctx.errHandler.ServeHTTP(w, req, err)
				return
			}

			conn, _, errHij := hijacker.Hijack()
			if errHij != nil {
				log.Errorf("vulcand/oxy/forward/websocket: Failed to hijack responseWriter")
				ctx.errHandler.ServeHTTP(w, req, errHij)
				return
			}
			defer conn.Close()

			errWrite := resp.Write(conn)
			if errWrite != nil {
				log.Errorf("vulcand/oxy/forward/websocket: Failed to forward response")
				ctx.errHandler.ServeHTTP(w, req, errWrite)
				return
			}
		}
		return
	}

	// Only the targetConn choose to CheckOrigin or not
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
		return true
	}}

	utils.RemoveHeaders(resp.Header, WebsocketUpgradeHeaders...)
	underlyingConn, err := upgrader.Upgrade(w, req, resp.Header)
	if err != nil {
		log.Errorf("vulcand/oxy/forward/websocket: Error while upgrading connection : %v", err)
		return
	}
	defer underlyingConn.Close()
	defer targetConn.Close()

	errc := make(chan error, 2)
	replicate := func(dst io.Writer, src io.Reader, dstName string, srcName string) {
		_, errCopy := io.Copy(dst, src)
		if errCopy != nil {
			f.log.Errorf("vulcand/oxy/forward/websocket: Error when copying from %s to %s using io.Copy: %v", srcName, dstName, errCopy)
		} else {
			f.log.Infof("vulcand/oxy/forward/websocket: Copying from %s to %s using io.Copy completed without error.", srcName, dstName)
		}
		errc <- errCopy
	}

	go replicate(targetConn.UnderlyingConn(), underlyingConn.UnderlyingConn(), "backend", "client")

	// Try to read the first message
	msgType, msg, err := targetConn.ReadMessage()
	if err != nil {
		log.Errorf("vulcand/oxy/forward/websocket: Couldn't read first message : %v", err)
	} else {
		underlyingConn.WriteMessage(msgType, msg)
	}

	go replicate(underlyingConn.UnderlyingConn(), targetConn.UnderlyingConn(), "client", "backend")
	<-errc
}

// copyWebsocketRequest makes a copy of the specified request.
func (f *httpForwarder) copyWebSocketRequest(req *http.Request) (outReq *http.Request) {
	outReq = new(http.Request)
	*outReq = *req // includes shallow copies of maps, but we handle this below

	outReq.URL = utils.CopyURL(req.URL)
	outReq.URL.Scheme = req.URL.Scheme

	// sometimes backends might be registered as HTTP/HTTPS servers so translate URLs to websocket URLs.
	switch req.URL.Scheme {
	case "https":
		outReq.URL.Scheme = "wss"
	case "http":
		outReq.URL.Scheme = "ws"
	}

	if requestURI, err := url.ParseRequestURI(outReq.RequestURI); err == nil {
		if requestURI.RawPath != "" {
			outReq.URL.Path = requestURI.RawPath
		} else {
			outReq.URL.Path = requestURI.Path
		}
		outReq.URL.RawQuery = requestURI.RawQuery
	}

	outReq.URL.Host = req.URL.Host

	outReq.Header = make(http.Header)
	// gorilla websocket use this header to set the request.Host tested in checkSameOrigin
	outReq.Header.Set("Host", outReq.Host)
	utils.CopyHeaders(outReq.Header, req.Header)
	utils.RemoveHeaders(outReq.Header, WebsocketDialHeaders...)

	if f.rewriter != nil {
		f.rewriter.Rewrite(outReq)
	}
	return outReq
}

// serveHTTP forwards HTTP traffic using the configured transport
func (f *httpForwarder) serveHTTP(w http.ResponseWriter, inReq *http.Request, ctx *handlerContext) {
	if f.log.Level >= log.DebugLevel {
		logEntry := f.log.WithField("Request", utils.DumpHttpRequest(inReq))
		logEntry.Debug("vulcand/oxy/forward/http: begin ServeHttp on request")
		defer logEntry.Debug("vulcand/oxy/forward/http: completed ServeHttp on request")
	}

	pw := &utils.ProxyWriter{
		W: w,
	}
	start := time.Now().UTC()

	outReq := new(http.Request)
	*outReq = *inReq // includes shallow copies of maps, but we handle this in Director

	revproxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			f.modifyRequest(req, inReq.URL)
		},
		Transport:      f.roundTripper,
		FlushInterval:  f.flushInterval,
		ModifyResponse: f.modifyResponse,
	}
	revproxy.ServeHTTP(pw, outReq)

	if inReq.TLS != nil {
		f.log.Infof("vulcand/oxy/forward/http: Round trip: %v, code: %v, Length: %v, duration: %v tls:version: %x, tls:resume:%t, tls:csuite:%x, tls:server:%v",
			inReq.URL, pw.Code, pw.Length, time.Now().UTC().Sub(start),
			inReq.TLS.Version,
			inReq.TLS.DidResume,
			inReq.TLS.CipherSuite,
			inReq.TLS.ServerName)
	} else {
		f.log.Infof("vulcand/oxy/forward/http: Round trip: %v, code: %v, Length: %v, duration: %v",
			inReq.URL, pw.Code, pw.Length, time.Now().UTC().Sub(start))
	}
}

// isWebsocketRequest determines if the specified HTTP request is a
// websocket handshake request
func IsWebsocketRequest(req *http.Request) bool {
	containsHeader := func(name, value string) bool {
		items := strings.Split(req.Header.Get(name), ",")
		for _, item := range items {
			if value == strings.ToLower(strings.TrimSpace(item)) {
				return true
			}
		}
		return false
	}
	return containsHeader(Connection, "upgrade") && containsHeader(Upgrade, "websocket")
}
