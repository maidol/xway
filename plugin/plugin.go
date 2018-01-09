package plugin

import (
	"xway/router"
)

// Registry contains common obj.
type Registry struct {
	router router.Router
}

// New ...
func New() *Registry {
	return &Registry{}
}

func (r *Registry) SetRouter(router router.Router) error {
	r.router = router
	return nil
}

func (r *Registry) GetRouter() router.Router {
	return r.router
}
