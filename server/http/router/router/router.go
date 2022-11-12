// Package router ...
package router

import (
	"net/http"
)

// NewRouterFunc can be used to create a new router.
type NewRouterFunc func() Router

// TODO: check all comments

// The Router interface as found here is a copy of the Chi router interface.
// It was selected for it's simplicity compliance with the stdlib.
// Some methods were removded as they were considered unnecessary for go-micro.
// If you are missing anything please open an issue.

// TODO: proper comment

// Router is a servermux.
type Router interface { //nolint:interfacebloat
	http.Handler
	Routes

	// Handle and HandleFunc adds routes for `pattern` that matches
	// all HTTP methods.
	Handle(pattern string, h http.Handler)
	HandleFunc(pattern string, h http.HandlerFunc)

	// Method and MethodFunc adds routes for `pattern` that matches
	// the `method` HTTP method.
	Method(method, pattern string, h http.Handler)
	MethodFunc(method, pattern string, h http.HandlerFunc)

	// HTTP-method routing along `pattern`
	Connect(pattern string, h http.HandlerFunc)
	Delete(pattern string, h http.HandlerFunc)
	Get(pattern string, h http.HandlerFunc)
	Head(pattern string, h http.HandlerFunc)
	Options(pattern string, h http.HandlerFunc)
	Patch(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
	Put(pattern string, h http.HandlerFunc)
	Trace(pattern string, h http.HandlerFunc)

	// Use appends one or more middlewares onto the Router stack.
	Use(middlewares ...func(http.Handler) http.Handler)

	// Mount attaches another http.Handler along ./pattern/*
	// This can be used to mount extra routers in subpaths.
	Mount(pattern string, h http.Handler)

	// NotFound defines a handler to respond whenever a route could
	// not be found.
	NotFound(h http.HandlerFunc)

	// MethodNotAllowed defines a handler to respond whenever a method is
	// not allowed.
	MethodNotAllowed(h http.HandlerFunc)
}

// Routes interface adds two methods for router traversal, which is also
// used by the `docgen` subpackage to generation documentation for Routers.
type Routes interface {
	// Routes returns the routing tree in an easily traversable structure.
	// Routes() []Route

	// Middlewares returns the list of middlewares in use by the router.
	Middlewares() []func(http.Handler) http.Handler
}

// Route describes the details of a routing handler.
// Handlers map key is an HTTP method.
type Route struct {
	SubRoutes Routes
	Handlers  map[string]http.Handler
	Pattern   string
}
