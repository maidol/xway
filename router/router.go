package router

import (
	"net/http"
)

// Router ...
type Router interface {
	// // Sets the not-found handler (this handler is called when no other handlers/routes in the routing library match
	// SetNotFound(http.Handler) error

	// // Gets the not-found handler that is currently in use by this router.
	// GetNotFound() http.Handler

	// // Removes a route. The http.Handler associated with it, will be discarded.
	// Remove(string) error

	// Adds a new route->handler combination. The route is a string which provides the routing expression. http.Handler is called when this expression matches a request.
	Handle(string, http.Handler) error

	// Validates whether this is an acceptable route expression
	IsValid(string) bool

	// ServiceHTTP is the http.Handler implementation that allows callers to route their calls to sub-http.Handlers based on route matches.
	ServeHTTP(http.ResponseWriter, *http.Request)
}
