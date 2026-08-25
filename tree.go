package chi

import (
	"net/http"
	"strings"
)

type nodeType uint8

const (
	ntStatic nodeType = iota
	ntParam
	ntWildcard
)

type routeEndpoint struct {
	pattern  string
	handlers map[string]http.Handler
}

type node struct {
	typ       nodeType
	prefix    string
	paramName string
	endpoints *routeEndpoint
	children  []*node
}

func (n *node) InsertRoute(method, pattern string, handler http.Handler) {
	origPattern := pattern
	current := n

	segments := splitPattern(pattern)
	for _, seg := range segments {
		if seg == "" {
			continue
		}

		var segType nodeType
		var pName string
		var pfx string

		if seg == "*" || seg == "/*" {
			segType = ntWildcard
			pfx = "*"
		} else if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			segType = ntParam
			pName = seg[1 : len(seg)-1]
			pfx = ":" + pName
		} else {
			segType = ntStatic
			pfx = seg
		}

		var child *node
		for _, c := range current.children {
			if c.typ == segType && (segType != ntStatic || c.prefix == pfx) {
				child = c
				break
			}
		}

		if child == nil {
			child = &node{
				typ:       segType,
				prefix:    pfx,
				paramName: pName,
			}
			current.children = append(current.children, child)
		}
		current = child
	}

	if current.endpoints == nil {
		current.endpoints = &routeEndpoint{
			pattern:  origPattern,
			handlers: make(map[string]http.Handler),
		}
	}
	current.endpoints.handlers[method] = handler
}

func splitPattern(pattern string) []string {
	if pattern == "/" {
		return []string{"/"}
	}
	parts := strings.Split(pattern, "/")
	var segs []string
	for _, p := range parts {
		if p != "" {
			segs = append(segs, p)
		}
	}
	return segs
}

func (n *node) FindRoute(rctx *Context, method, path string) (handler http.Handler, matchedPattern string, subPath string, methodNotAllowed bool) {
	cleanPath := path
	if cleanPath == "" {
		cleanPath = "/"
	}

	var segs []string
	if cleanPath != "/" {
		for _, s := range strings.Split(cleanPath, "/") {
			if s != "" {
				segs = append(segs, s)
			}
		}
	}

	var params RouteParams
	h, pat, sub, notAllowed := n.matchSegments(segs, 0, method, &params)
	if h != nil {
		if rctx != nil {
			for i := 0; i < len(params.Keys); i++ {
				rctx.URLParams.Add(params.Keys[i], params.Values[i])
			}
		}
		return h, pat, sub, false
	}
	return nil, "", "", notAllowed
}

func (n *node) matchSegments(segs []string, index int, method string, params *RouteParams) (http.Handler, string, string, bool) {
	if index == len(segs) {
		if n.endpoints != nil {
			if h, ok := n.endpoints.handlers[method]; ok {
				return h, n.endpoints.pattern, "", false
			}
			if h, ok := n.endpoints.handlers["*"]; ok {
				return h, n.endpoints.pattern, "", false
			}
			if len(n.endpoints.handlers) > 0 {
				return nil, "", "", true
			}
		}

		for _, child := range n.children {
			if child.typ == ntWildcard && child.endpoints != nil {
				if h, ok := child.endpoints.handlers[method]; ok {
					return h, child.endpoints.pattern, "/", false
				}
				if h, ok := child.endpoints.handlers["*"]; ok {
					return h, child.endpoints.pattern, "/", false
				}
			}
		}
		return nil, "", "", false
	}

	seg := segs[index]

	for _, child := range n.children {
		if child.typ == ntStatic && child.prefix == seg {
			h, pat, sub, notAllowed := child.matchSegments(segs, index+1, method, params)
			if h != nil || notAllowed {
				return h, pat, sub, notAllowed
			}
		}
	}

	for _, child := range n.children {
		if child.typ == ntParam {
			params.Add(child.paramName, seg)
			h, pat, sub, notAllowed := child.matchSegments(segs, index+1, method, params)
			if h != nil || notAllowed {
				return h, pat, sub, notAllowed
			}
			if len(params.Keys) > 0 {
				params.Keys = params.Keys[:len(params.Keys)-1]
				params.Values = params.Values[:len(params.Values)-1]
			}
		}
	}

	for _, child := range n.children {
		if child.typ == ntWildcard && child.endpoints != nil {
			remaining := "/" + strings.Join(segs[index:], "/")
			if h, ok := child.endpoints.handlers[method]; ok {
				return h, child.endpoints.pattern, remaining, false
			}
			if h, ok := child.endpoints.handlers["*"]; ok {
				return h, child.endpoints.pattern, remaining, false
			}
			if len(child.endpoints.handlers) > 0 {
				return nil, "", "", true
			}
		}
	}

	return nil, "", "", false
}

func (n *node) Routes() []Route {
	var routes []Route
	n.collectRoutes(&routes)
	return routes
}

func (n *node) collectRoutes(routes *[]Route) {
	if n.endpoints != nil {
		*routes = append(*routes, Route{
			Pattern:  n.endpoints.pattern,
			Handlers: n.endpoints.handlers,
		})
	}
	for _, child := range n.children {
		child.collectRoutes(routes)
	}
}
