package engine

import (
	"encoding/json"
	"fmt"

	"xway/router"
)

const (
	HTTP      = "http"
	WEBSOCKET = "websocket"
	RPC       = "rpc"
)

type rawFrontend struct {
	RouteId      string          `json:"routeId,omitempty"`
	DomainHost   string          `json:"domainHost,omitempty"`
	RouteUrl     string          `json:"routeUrl"`
	RedirectHost string          `json:"redirectHost,omitempty"` //需考虑分离到单独的host类型里
	ForwardURL   string          `json:"forwardUrl,omitempty"`
	BackendType  string          `json:"backendType,omitempty"` // 后端微服务http/rpc/..., 需考虑分离到单独的backend类型里
	Type         string          `json:"type,omitempty"`        // 前端请求类型http/websocket
	Config       json.RawMessage `json:"config,omitempty"`
	Status       int             `json:"status,string"`
}

type rawFrontends struct {
	Frontends []json.RawMessage
}

func FrontendFromJSON(router router.Router, in []byte) (*Frontend, error) {
	var rf *rawFrontend
	if err := json.Unmarshal(in, &rf); err != nil {
		// TODO: 转换失败处理
		// fmt.Printf("[FrontendFromJSON failure] Invalid Frontend json format, json.Unmarshal err: %v, %s", err, in)
		return nil, fmt.Errorf("Invalid Frontend json format, json.Unmarshal err: %v, data: %s", err, in)
	}

	// TODO: 处理多种rf.Type(http, websocket)
	// 目前只支持http类型的前端请求
	if rf.Type != HTTP {
		return nil, fmt.Errorf("Unsupported frontend type: %v", rf.Type)
	}

	var s HTTPFrontendSettings
	if rf.Config != nil {
		if err := json.Unmarshal(rf.Config, &s); err != nil {
			return nil, fmt.Errorf("Invalid HTTPFrontendSettings json format, json.Unmarshal err: %v, config data: %s; it's frontend: %s", err, rf.Config, in)
		}
	}

	f, err := NewHTTPFrontend(router, rf.RouteId, rf.DomainHost, rf.RedirectHost, rf.ForwardURL, rf.RouteUrl, rf.Status, s)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func FrontendsFromJSON(router router.Router, in []byte) ([]Frontend, error) {
	var rfs *rawFrontends
	if err := json.Unmarshal(in, &rfs); err != nil {
		return nil, err
	}

	out := make([]Frontend, len(rfs.Frontends))
	for i, raw := range rfs.Frontends {
		f, err := FrontendFromJSON(router, raw)
		if err != nil {
			return nil, err
		}
		out[i] = *f
	}

	return out, nil
}
