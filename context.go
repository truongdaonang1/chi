package chi

import (
	"context"
	"net/http"
	"strings"
)

// RouteCtxKey is the context.Context key to store the request context.
var RouteCtxKey = &contextKey{"RouteContext"}

type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "chi context value " + k.name
}

// Context is the default routing context for a request.
type Context struct {
	Routes Routes

	// RoutePath is the request URL path being routed.
	RoutePath string

	// RouteMethod is the request method being routed.
	RouteMethod string

	// RoutePatterns is the routing pattern stack.
	RoutePatterns []string

	// URLParams are the parameters matched on the route.
	URLParams RouteParams

	// routeParams are the parameters matched during tree traversal
	routeParams RouteParams

	// method is the HTTP method of the request
	method string

	// methodPatterns are the patterns matched for the method
	methodPatterns []string

	// parentCtx is the parent context when nesting contexts
	parentCtx context.Context
}

// RouteParams is a structure to store URL params as key/value pairs.
type RouteParams struct {
	Keys   []string
	Values []string
}

// Add adds a key/value pair to the URL params.
func (s *RouteParams) Add(key, value string) {
	s.Keys = append(s.Keys, key)
	s.Values = append(s.Values, value)
}

// Get returns the value for a key in the URL params.
func (s *RouteParams) Get(key string) string {
	for i := len(s.Keys) - 1; i >= 0; i-- {
		if s.Keys[i] == key {
			return s.Values[i]
		}
	}
	return ""
}

// NewRouteContext returns a new routing Context object.
func NewRouteContext() *Context {
	return &Context{
		URLParams: RouteParams{
			Keys:   make([]string, 0),
			Values: make([]string, 0),
		},
		routeParams: RouteParams{
			Keys:   make([]string, 0),
			Values: make([]string, 0),
		},
		RoutePatterns: make([]string, 0),
	}
}

// Reset clears the context fields.
func (rctx *Context) Reset() {
	rctx.Routes = nil
	rctx.RoutePath = ""
	rctx.RouteMethod = ""
	rctx.RoutePatterns = rctx.RoutePatterns[:0]
	rctx.URLParams.Keys = rctx.URLParams.Keys[:0]
	rctx.URLParams.Values = rctx.URLParams.Values[:0]
	rctx.routeParams.Keys = rctx.routeParams.Keys[:0]
	rctx.routeParams.Values = rctx.routeParams.Values[:0]
	rctx.method = ""
	rctx.methodPatterns = rctx.methodPatterns[:0]
	rctx.parentCtx = nil
}

// URLParam returns the corresponding URL parameter value for the key.
func (rctx *Context) URLParam(key string) string {
	return rctx.URLParams.Get(key)
}

// RoutePattern builds the matched route pattern from RoutePatterns stack.
func (rctx *Context) RoutePattern() string {
	if rctx == nil || len(rctx.RoutePatterns) == 0 {
		return ""
	}
	if len(rctx.RoutePatterns) == 1 {
		return rctx.RoutePatterns[0]
	}

	var sb strings.Builder
	for i, p := range rctx.RoutePatterns {
		if p == "" {
			continue
		}
		p = strings.TrimSuffix(p, "/*")
		if i < len(rctx.RoutePatterns)-1 {
			p = strings.TrimSuffix(p, "/")
		}
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "/") && sb.Len() > 0 {
			sb.WriteString("/")
		}
		sb.WriteString(p)
	}

	if sb.Len() == 0 {
		return "/"
	}
	return sb.String()
}

// RouteContext returns chi's routing Context from a request context.
func RouteContext(ctx context.Context) *Context {
	if ctx == nil {
		return nil
	}
	rctx, _ := ctx.Value(RouteCtxKey).(*Context)
	return rctx
}
