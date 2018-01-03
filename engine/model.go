package engine

import "xway/router"

type Snapshot struct {
	Index         uint64
	FrontendSpecs []FrontendSpec
}

type FrontendSpec struct {
	Frontend Frontend
	// Middlewares []Middleware
}

type Frontend struct {
	// Id        string
	// Route     string
	// Type      string
	// BackendId string

	// Settings interface{} `json:"config,omitempty"`

	RouteId      string      `json:"routeId,omitempty"`
	DomainHost   string      `json:"domainHost,omitempty"`
	RouteUrl     string      `json:"routeUrl"`
	RedirectHost string      `json:"redirectHost,omitempty"`
	ForwardURL   string      `json:"forwardUrl,omitempty"`
	Type         string      `json:"type,omitempty"`
	Config       interface{} `json:"config,omitempty"`
}

type HTTPFrontendSettings struct {
	Hostname string
	Auth     []string `json:"auth,omitempty"`
}

func NewHTTPFrontend(router router.Router, routeId, domainHost, redirectHost, forwardURL string, routeExpr string, settings HTTPFrontendSettings) (*Frontend, error) {
	return &Frontend{
		RouteId:      routeId,
		DomainHost:   domainHost,
		RedirectHost: redirectHost,
		ForwardURL:   forwardURL,
		Type:         HTTP,
		RouteUrl:     routeExpr,
		Config:       settings,
	}, nil
}
