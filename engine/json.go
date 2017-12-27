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
	Id        string
	Route     string
	Type      string
	BackendId string
	Settings  json.RawMessage
}

type HTTPFrontendSettings struct {
	Hostname string
}

func FrontendFromJSON(router router.Router, in []byte) (*Frontend, error) {
	var rf *rawFrontend
	if err := json.Unmarshal(in, &rf); err != nil {
		return nil, err
	}

	if rf.Type != HTTP {
		return nil, fmt.Errorf("Unsupported fronted type: %v", rf.Type)
	}

	var s HTTPFrontendSettings
	if rf.Settings != nil {
		if err := json.Unmarshal(rf.Settings, &s); err != nil {
			return nil, fmt.Errorf("Invalid HTTPFrontendSettings json format: %v", err.Error())
		}
	}

	f, err := NewHTTPFrontend(router, "", "", rf.Route, s)
	if err != nil {
		return nil, err
	}
	return f, nil
}
