package transport

import (
	"strings"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	mw "github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
)

// RouterGroup mimics the grouping semantics from web frameworks
// such as Gin, allowing developers to register handlers with a shared
// path prefix and middlewares.
type RouterGroup struct {
	router      *router.Router
	prefix      string
	middlewares []mw.Middleware
}

func NewRouterGroup(r *router.Router, prefix string, m ...mw.Middleware) *RouterGroup {
	return &RouterGroup{
		router:      r,
		prefix:      normalize(prefix),
		middlewares: append([]mw.Middleware(nil), m...),
	}
}

func (g *RouterGroup) Group(prefix string, m ...mw.Middleware) *RouterGroup {
	next := &RouterGroup{
		router:      g.router,
		prefix:      join(g.prefix, prefix),
		middlewares: append([]mw.Middleware{}, g.middlewares...),
	}
	next.middlewares = append(next.middlewares, m...)
	return next
}

func (g *RouterGroup) Use(m ...mw.Middleware) {
	g.middlewares = append(g.middlewares, m...)
}

func (g *RouterGroup) Handle(method, path string, handler mw.HandlerFunc, m ...mw.Middleware) {
	fullPath := join(g.prefix, path)
	chain := append([]mw.Middleware{}, g.middlewares...)
	chain = append(chain, m...)
	g.router.Handle(method, fullPath, mw.Chain(handler, chain...))
}

func (g *RouterGroup) GET(path string, handler mw.HandlerFunc, m ...mw.Middleware) {
	g.Handle(fasthttp.MethodGet, path, handler, m...)
}

func (g *RouterGroup) POST(path string, handler mw.HandlerFunc, m ...mw.Middleware) {
	g.Handle(fasthttp.MethodPost, path, handler, m...)
}

func (g *RouterGroup) PUT(path string, handler mw.HandlerFunc, m ...mw.Middleware) {
	g.Handle(fasthttp.MethodPut, path, handler, m...)
}

func (g *RouterGroup) DELETE(path string, handler mw.HandlerFunc, m ...mw.Middleware) {
	g.Handle(fasthttp.MethodDelete, path, handler, m...)
}

func (g *RouterGroup) PATCH(path string, handler mw.HandlerFunc, m ...mw.Middleware) {
	g.Handle(fasthttp.MethodPatch, path, handler, m...)
}

func normalize(path string) string {
	if path == "" || path == "/" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}

func join(prefix, path string) string {
	p := normalize(path)
	if prefix == "" {
		return p
	}
	if p == "" {
		return prefix
	}
	return prefix + p
}
