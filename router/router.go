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

	// Validates whether this is an acceptable route expression
	IsValid(*http.Request) (bool, interface{})

	// ServiceHTTP is the negroni.Handler implementation
	ServeHTTP(http.ResponseWriter, *http.Request, http.HandlerFunc)
}
