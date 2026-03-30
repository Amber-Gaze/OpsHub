package middleware

import (
	"strings"

	"github.com/valyala/fasthttp"
)

// RequireAdmin 仅管理员可访问（需先经过 JWTAuthMiddleware）。
func RequireAdmin() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			if !c.IsAdmin {
				c.Abort(fasthttp.StatusForbidden, "admin required")
				return
			}
			next(c)
		}
	}
}

// RequireSelfOrAdmin 仅目标用户本人或管理员（路径参数 name）。
func RequireSelfOrAdmin() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			name, _ := c.UserValue("name").(string)
			name = strings.TrimSpace(name)
			if name == "" {
				c.Abort(fasthttp.StatusBadRequest, "invalid name")
				return
			}
			if c.IsAdmin || strings.EqualFold(name, c.Username) {
				next(c)
				return
			}
			c.Abort(fasthttp.StatusForbidden, "forbidden")
		}
	}
}
