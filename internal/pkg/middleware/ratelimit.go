package middleware

import (
	"github.com/Amber-Gaze/OpsHub/pkg/rate"
	"github.com/valyala/fasthttp"
)

func RateLimit(rps int) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			key := string(c.Request.Header.Peek("Authorization"))
			if !rate.Allow(key, rps) {
				c.Abort(fasthttp.StatusTooManyRequests, "rate limited")
				return
			}
			next(c)
		}
	}
}
