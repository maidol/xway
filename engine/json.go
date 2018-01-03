package engine

import (
	"encoding/json"
	"fmt"

	"xway/router"
)

const (
	HTTP = "http"
)

type rawFrontend struct {
	RouteId      string          `json:"routeId,omitempty"`
	DomainHost   string          `json:"domainHost,omitempty"`
	RouteUrl     string          `json:"routeUrl"`
	RedirectHost string          `json:"redirectHost,omitempty"`
	ForwardURL   string          `json:"forwardUrl,omitempty"`
	Type         string          `json:"type,omitempty"`
	Config       json.RawMessage `json:"config,omitempty"`
}

type rawFrontends struct {
	Frontends []json.RawMessage
}

func FrontendFromJSON(router router.Router, in []byte) (*Frontend, error) {
	var rf *rawFrontend
	if err := json.Unmarshal(in, &rf); err != nil {
		return nil, err
	}

	if rf.Type != HTTP {
		return nil, fmt.Errorf("Unsupported frontend type: %v", rf.Type)
	}

	var s HTTPFrontendSettings
	if rf.Config != nil {
		if err := json.Unmarshal(rf.Config, &s); err != nil {
			return nil, fmt.Errorf("Invalid HTTPFrontendSettings json format: %v", err.Error())
		}
	}

	f, err := NewHTTPFrontend(router, rf.RouteId, rf.DomainHost, rf.RedirectHost, rf.ForwardURL, rf.RouteUrl, s)
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
