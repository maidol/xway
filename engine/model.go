package engine

import (
	"xway/router"
)

type StatsProvider interface {
	FrontendStats(FrontendKey) (*RoundTripStats, error)

	TopFrontends() ([]Frontend, error)
}

type RoundTripStats struct{}

type Snapshot struct {
	Index         uint64
	FrontendSpecs []FrontendSpec
}

type FrontendSpec struct {
	Frontend Frontend
	// Middlewares []Middleware
}

type Frontend struct {
	RouteId      string      `json:"routeId,omitempty"`
	DomainHost   string      `json:"domainHost,omitempty"`
	RouteUrl     string      `json:"routeUrl"`
	RedirectHost string      `json:"redirectHost,omitempty"` //需考虑分离到单独的host类型里
	ForwardURL   string      `json:"forwardUrl,omitempty"`
	BackendType  string      `json:"backendType,omitempty"` // 后端微服务http/rpc/..., 需考虑分离到单独的backend类型里
	Type         string      `json:"type,omitempty"`        // 前端请求类型http/websocket/...
	Config       interface{} `json:"config,omitempty"`
	Status       int         `json:"status"`
}

type HTTPFrontendSettings struct {
	Hostname string
	Auth     []string `json:"auth,omitempty"`
}

func NewHTTPFrontend(router router.Router, routeId, domainHost, redirectHost, forwardURL string, routeExpr string, status int, settings HTTPFrontendSettings) (*Frontend, error) {
	return &Frontend{
		RouteId:      routeId,
		DomainHost:   domainHost,
		RedirectHost: redirectHost,
		ForwardURL:   forwardURL,
		Type:         HTTP,
		RouteUrl:     routeExpr,
		Config:       settings,
		Status:       status,
	}, nil
}
