package chi

import (
	"context"
	"net/http"
)

// Router interface consisting of the core routing methods used by Chi's Mux.
type Router interface {
	http.Handler
	Routes

	Use(middlewares ...func(http.Handler) http.Handler)
	With(middlewares ...func(http.Handler) http.Handler) Router
	Group(fn func(r Router)) Router
	Route(pattern string, fn func(r Router)) Router
	Mount(pattern string, h http.Handler)
	Handle(pattern string, h http.Handler)
	HandleFunc(pattern string, h http.HandlerFunc)
	Method(method, pattern string, h http.Handler)
	MethodFunc(method, pattern string, h http.HandlerFunc)

	Connect(pattern string, h http.HandlerFunc)
	Delete(pattern string, h http.HandlerFunc)
	Get(pattern string, h http.HandlerFunc)
	Head(pattern string, h http.HandlerFunc)
	Options(pattern string, h http.HandlerFunc)
	Patch(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
	Put(pattern string, h http.HandlerFunc)
	Trace(pattern string, h http.HandlerFunc)

	NotFound(h http.HandlerFunc)
	MethodNotAllowed(h http.HandlerFunc)
}

// Routes interface to access the tree of routes in the router.
type Routes interface {
	Routes() []Route
	Middlewares() Middlewares
	Match(rctx *Context, method, path string) bool
}

// Route describes a branch of routing tree.
type Route struct {
	Pattern  string
	Handlers map[string]http.Handler
}

// Middlewares type slice.
type Middlewares []func(http.Handler) http.Handler

// NewRouter returns a new Mux object.
func NewRouter() *Mux {
	return NewMux()
}

// URLParam returns the url parameter from a http.Request context.
func URLParam(r *http.Request, key string) string {
	if rctx := RouteContext(r.Context()); rctx != nil {
		return rctx.URLParam(key)
	}
	return ""
}

// URLParamFromCtx returns the url parameter from a context.Context.
func URLParamFromCtx(ctx context.Context, key string) string {
	if rctx := RouteContext(ctx); rctx != nil {
		return rctx.URLParam(key)
	}
	return ""
}
