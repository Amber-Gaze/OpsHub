package middleware

import "github.com/valyala/fasthttp"

type HandlerFunc func(*Context)

type Middleware func(HandlerFunc) HandlerFunc

func Chain(h HandlerFunc, m ...Middleware) fasthttp.RequestHandler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}

	return func(ctx *fasthttp.RequestCtx) {
		c := NewContext(ctx)
		h(c)
	}
}
