package chi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

var _ Router = &Mux{}

// Mux is a HTTP route multiplexer.
type Mux struct {
	tree                    *node
	middlewares             []func(http.Handler) http.Handler
	notFoundHandler         http.HandlerFunc
	methodNotAllowedHandler http.HandlerFunc
	pool                    sync.Pool
	parent                  *Mux
}

func NewMux() *Mux {
	mux := &Mux{tree: &node{}}
	mux.pool.New = func() interface{} {
		return NewRouteContext()
	}
	return mux
}

func (mx *Mux) Use(middlewares ...func(http.Handler) http.Handler) {
	mx.middlewares = append(mx.middlewares, middlewares...)
}

func (mx *Mux) With(middlewares ...func(http.Handler) http.Handler) Router {
	sub := &Mux{
		tree:                    mx.tree,
		middlewares:             append(mx.middlewares[:len(mx.middlewares):len(mx.middlewares)], middlewares...),
		notFoundHandler:         mx.notFoundHandler,
		methodNotAllowedHandler: mx.methodNotAllowedHandler,
		parent:                  mx,
	}
	return sub
}

func (mx *Mux) Group(fn func(r Router)) Router {
	sub := mx.With()
	if fn != nil {
		fn(sub)
	}
	return sub
}

func (mx *Mux) Route(pattern string, fn func(r Router)) Router {
	if fn == nil {
		panic(fmt.Sprintf("chi: attempting to Route() a nil subrouter on %q", pattern))
	}
	subRouter := NewRouter()
	fn(subRouter)
	mx.Mount(pattern, subRouter)
	return subRouter
}

func (mx *Mux) Mount(pattern string, handler http.Handler) {
	if handler == nil {
		panic(fmt.Sprintf("chi: attempting to Mount() a nil handler on %q", pattern))
	}

	if pattern == "" || pattern[0] != '/' {
		panic(fmt.Sprintf("chi: routing pattern must begin with '/' in %q", pattern))
	}

	mountPattern := strings.TrimSuffix(pattern, "/")
	if mountPattern == "" {
		mountPattern = "/*"
	} else {
		mountPattern = mountPattern + "/*"
	}

	mx.handle("*", mountPattern, handler)
	if pattern != "/" && !strings.HasSuffix(pattern, "/*") {
		exactPattern := strings.TrimSuffix(pattern, "/")
		mx.handle("*", exactPattern, handler)
	}
}

func (mx *Mux) Handle(pattern string, handler http.Handler) {
	mx.handle("*", pattern, handler)
}

func (mx *Mux) HandleFunc(pattern string, handler http.HandlerFunc) {
	mx.handle("*", pattern, handler)
}

func (mx *Mux) Method(method, pattern string, handler http.Handler) {
	mx.handle(method, pattern, handler)
}

func (mx *Mux) MethodFunc(method, pattern string, handler http.HandlerFunc) {
	mx.handle(method, pattern, handler)
}

func (mx *Mux) Connect(pattern string, h http.HandlerFunc) { mx.MethodFunc(http.MethodConnect, pattern, h) }
func (mx *Mux) Delete(pattern string, h http.HandlerFunc)  { mx.MethodFunc(http.MethodDelete, pattern, h) }
func (mx *Mux) Get(pattern string, h http.HandlerFunc)     { mx.MethodFunc(http.MethodGet, pattern, h) }
func (mx *Mux) Head(pattern string, h http.HandlerFunc)    { mx.MethodFunc(http.MethodHead, pattern, h) }
func (mx *Mux) Options(pattern string, h http.HandlerFunc) { mx.MethodFunc(http.MethodOptions, pattern, h) }
func (mx *Mux) Patch(pattern string, h http.HandlerFunc)   { mx.MethodFunc(http.MethodPatch, pattern, h) }
func (mx *Mux) Post(pattern string, h http.HandlerFunc)    { mx.MethodFunc(http.MethodPost, pattern, h) }
func (mx *Mux) Put(pattern string, h http.HandlerFunc)     { mx.MethodFunc(http.MethodPut, pattern, h) }
func (mx *Mux) Trace(pattern string, h http.HandlerFunc)   { mx.MethodFunc(http.MethodTrace, pattern, h) }

func (mx *Mux) NotFound(h http.HandlerFunc) {
	mx.notFoundHandler = h
}

func (mx *Mux) MethodNotAllowed(h http.HandlerFunc) {
	mx.methodNotAllowedHandler = h
}

func (mx *Mux) handle(method, pattern string, handler http.Handler) {
	if pattern == "" || pattern[0] != '/' {
		panic(fmt.Sprintf("chi: routing pattern must begin with '/' in %q", pattern))
	}
	if handler == nil {
		panic(fmt.Sprintf("chi: attempting to route to a nil handler on %q", pattern))
	}

	h := mx.chain(handler)
	mx.tree.InsertRoute(method, pattern, h)
}

func (mx *Mux) chain(handler http.Handler) http.Handler {
	for i := len(mx.middlewares) - 1; i >= 0; i-- {
		handler = mx.middlewares[i](handler)
	}
	return handler
}

func (mx *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rctx, _ := r.Context().Value(RouteCtxKey).(*Context)
	isRoot := false
	if rctx == nil {
		rctx = mx.pool.Get().(*Context)
		rctx.Reset()
		rctx.parentCtx = r.Context()
		r = r.WithContext(context.WithValue(r.Context(), RouteCtxKey, rctx))
		isRoot = true
	}

	routePath := rctx.RoutePath
	if routePath == "" {
		if r.URL.RawPath != "" {
			routePath = r.URL.RawPath
		} else {
			routePath = r.URL.Path
		}
		if routePath == "" {
			routePath = "/"
		}
		rctx.RoutePath = routePath
	}
	rctx.RouteMethod = r.Method

	target, matchedPattern, subPath, isMethodNotAllowed := mx.tree.FindRoute(rctx, r.Method, routePath)

	if target != nil {
		if matchedPattern != "" {
			rctx.RoutePatterns = append(rctx.RoutePatterns, matchedPattern)
		}
		if subPath != "" {
			rctx.RoutePath = subPath
		}

		target.ServeHTTP(w, r)
		return
	}

	if isMethodNotAllowed {
		if mx.methodNotAllowedHandler != nil {
			mx.chain(mx.methodNotAllowedHandler).ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
		return
	}

	if mx.notFoundHandler != nil {
		mx.chain(mx.notFoundHandler).ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}

	_ = isRoot
}

func (mx *Mux) Match(rctx *Context, method, path string) bool {
	target, _, _, _ := mx.tree.FindRoute(rctx, method, path)
	return target != nil
}

func (mx *Mux) Middlewares() Middlewares {
	return mx.middlewares
}

func (mx *Mux) Routes() []Route {
	return mx.tree.Routes()
}
