package router

import (
	"net/http"
)

// Router ...
type Router interface {

	// Removes a route
	Remove(interface{}) error

	// Add/update a route
	Handle(interface{}) error

	// GetFrontends
	GetFrontends() interface{}

	// Validates whether this is an acceptable route expression
	IsValid(string) bool

	// IsMatch whether this is a matched route
	IsMatch(*http.Request) (bool, interface{})

	// ServiceHTTP is the negroni.Handler implementation
	ServeHTTP(http.ResponseWriter, *http.Request, http.HandlerFunc)
}
