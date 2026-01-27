package middleware

import (
	"runtime/debug"

	"github.com/valyala/fasthttp"
)

func Recover() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			defer func() {
				if r := recover(); r != nil {
					debug.PrintStack()
					c.Abort(fasthttp.StatusInternalServerError, "internal error")
				}
			}()
			next(c)
		}
	}
}
