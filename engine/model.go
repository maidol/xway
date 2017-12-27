package engine

import "xway/router"

type Snapshot struct {
	Index         uint64
	FrontendSpecs []FrontendSpec
}

type FrontendSpec struct {
	Frontend Frontend
}

type Frontend struct {
	Id        string
	Route     string
	Type      string
	BackendId string

	Settings interface{} `json:",omitempty"`

	RouteId      string `json:"routeId,omitempty"`
	DomainHost   string `json:"domainHost,omitempty"`
	RouteUrl     string `json:"routeUrl"`
	RedirectHost string `json:"redirectHost,omitempty"`
	ForwardURL   string `json:"forwardUrl,omitempty"`
	// Type         string      `json:"type,omitempty"`
	Config interface{} `json:"config,omitempty"`
}

func NewHTTPFrontend(router router.Router, id, backendId string, routeExpr string, settings HTTPFrontendSettings) (*Frontend, error) {
	return &Frontend{
		Type:     HTTP,
		Route:    routeExpr,
		Settings: settings,
	}, nil
}
