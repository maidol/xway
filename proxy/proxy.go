package proxy

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/testutils"

	"xway/context"
	en "xway/engine"
	xemun "xway/enum"
	xerror "xway/error"
)

var errLog = log.New(os.Stderr, "[MW:proxy]", 0)

// NewDo ...
func NewDo(tr *http.Transport) (http.HandlerFunc, error) {
	// 默认设置client 30s请求超时
	reqTimeout := 30 * time.Second
	client := &http.Client{Transport: tr, Timeout: reqTimeout}

	pr := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// u, err := url.Parse("http://192.168.2.102:8708" + r.URL.String())
		xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
		matchRouteFrontend := xwayCtx.Map["matchRouteFrontend"].(*en.Frontend)
		forwardURL := xwayCtx.Map["forwardURL"].(string)
		forwardURL = strings.Replace(forwardURL, strings.ToLower(strings.TrimRight(matchRouteFrontend.RouteUrl, "/")), strings.ToLower(strings.TrimRight(matchRouteFrontend.ForwardURL, "/")), 1)
		u, err := url.Parse("http://" + matchRouteFrontend.RedirectHost + forwardURL + "?" + r.URL.RawQuery)

		// fmt.Printf("[MW:proxy] -> url forward to: %v, %v\n", u.Host, u)

		// TODO: 优化, 异步日志
		// pool := xgrpool.Default()
		// pool.JobQueue <- func() {
		// 	fmt.Printf("-> url forward to: %v, %v\n", u.Host, u)
		// 	fmt.Printf("-> url forward to: %v, %v\n", u.Host, u)
		// }

		if err != nil {
			fmt.Printf("[MW:proxy] url.Parse err: %v\n", err)
			e := xerror.NewRequestError(xemun.RetProxyError, xemun.ECodeProxyFailed, err.Error())
			e.Write(w)
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

		// TODO: 优化并精简错误处理代码logProxyError
		resp, err := client.Do(outReq)
		if err != nil {
			// TODO: 需要处理client.Do的请求错误, 根据错误类型
			switch ue := err.(type) {
			case *url.Error:
				if ue.Timeout() {
					// client.Do请求超时
					ue.Err = fmt.Errorf("request was timeout(%s): %s", reqTimeout, ue.Err.Error())
					break
				}
				if ue.Err == context.Canceled {
					ue.Err = fmt.Errorf("request was canceled: %s", ue.Err.Error())
					break
				}
			// case *url.EscapeError:
			// case *url.InvalidHostError:
			default:
			}
			var d []byte
			var respmsg string
			if resp != nil {
				d, _ = ioutil.ReadAll(resp.Body)
				respmsg = fmt.Sprintf("[resp状态码]:%d, resp.body:%s", resp.StatusCode, d)
			}
			// fmt.Printf("[MW:proxy] client.Do err: %v\n", err)
			msg := fmt.Sprintf("client.Do err: %v", err.Error()+respmsg)
			cause := formatRequestCause(u, msg)
			e := xerror.NewRequestError(xemun.RetProxyError, xemun.ECodeProxyFailed, cause)
			e.Write(w)
			logProxyError(outReq, e)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			// fmt.Printf("[MW:proxy] ioutil.ReadAll err: %v\n", err)
			msg := fmt.Sprintf("ioutil.ReadAll(resp.Body) err: %v", err.Error())
			cause := formatRequestCause(u, msg)
			e := xerror.NewRequestError(xemun.RetProxyError, xemun.ECodeInternal, cause)
			e.Write(w)
			logProxyError(outReq, e)
			return
		}

		// TODO: 处理resp.StatusCode
		statusCode := resp.StatusCode
		b, rexerr := regexp.MatchString("^[2|3]", strconv.Itoa(statusCode))
		if rexerr != nil {
			// fmt.Printf("[MW:proxy] regexp.MatchString err: %v\n", rexerr)
			msg := fmt.Sprintf("resp.StatusCode regexp.MatchString(\"^[2|3]\", %v) err: %v", statusCode, err.Error())
			cause := formatRequestCause(u, msg)
			e := xerror.NewRequestError(xemun.RetProxyError, xemun.ECodeProxyFailed, cause)
			e.Write(w)
			logProxyError(outReq, e)
			return
		}
		if !b {
			// 处理4xx, 5xx ...
			msg := fmt.Sprintf("源服务器错误,[状态码]:%v, body:%s", statusCode, body)
			cause := formatRequestCause(u, msg)
			// TODO: 增加源服务器错误码26代替此处xemun.ECodeProxyFailed错误码, 兼容原网关
			e := xerror.NewRequestError(xemun.RetProxyError, xemun.ECodeOriginalServerError, cause)
			e.Write(w)
			logProxyError(outReq, e)
			return
		}

		sendResponse(w, resp.Header, statusCode, body)
	})
	return pr, nil
}

func sendResponse(w http.ResponseWriter, header http.Header, statusCode int, body []byte) {
	for k, v := range header {
		for _, s := range v {
			w.Header().Add(k, s)
		}
	}
	// body = append(body, ([]byte("abc"))...)
	// w.Header().Set("Content-Length", string(len(body)))
	// fmt.Printf("-> response data: %v\n", string(body))

	// 顺序执行, w.WriteHeader的操作必须放在w.Header().Add的设置后面, 避免影响header
	w.WriteHeader(statusCode)
	w.Write(body)
}

func formatRequestCause(url *url.URL, cause string) string {
	return fmt.Sprintf("url forward to %v: %v", url, cause)
}

func logProxyError(r *http.Request, err error) {
	xwayCtx := xwaycontext.DefaultXWayContext(r.Context())

	tk := "cw:gateway:err:" + xwayCtx.RequestId
	msg := "[MW:proxy:logProxyError] " + err.Error()
	logrus.WithFields(logrus.Fields{"key": tk}).Error(msg)
	// logrus.WithFields(logrus.Fields{"topic": "gateway-error", "key": tk}).Error(msg)

	// // TODO: 优化日志记录, 精简请求头和body数据
	// errLog.Printf("======http proxy occur err: begin======\n")
	// errLog.Printf("request option: %+v\n", r)
	// errLog.Printf("err message: %v\n", err)
	// body, e := ioutil.ReadAll(r.Body)
	// if e == nil {
	// 	errLog.Printf("request body: %s\n", body)
	// } else {
	// 	errLog.Printf("request body ioutil.ReadAll err: %v\n", e)
	// }
	// errLog.Printf("======http proxy occur err: end======\n")
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
