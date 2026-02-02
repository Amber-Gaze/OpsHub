package middleware

import (
	"errors"
	"strings"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/jwt"
	"github.com/valyala/fasthttp"
)

func JWTAuthMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			authHeader := string(c.Request.Header.Peek("Authorization"))
			if authHeader == "" {
				c.SetStatusCode(fasthttp.StatusUnauthorized)
				c.SetBodyString("missing token")
				return
			}

			tokenStr, err := parseBearer(authHeader)
			if err != nil {
				c.SetStatusCode(fasthttp.StatusUnauthorized)
				c.SetBodyString("invalid auth header")
				return
			}

			claims, err := jwt.ParseToken(tokenStr)
			if err != nil {
				c.SetStatusCode(fasthttp.StatusUnauthorized)
				c.SetBodyString("invalid token")
				return
			}

			// ✔ 身份验证完成，把最小信息放入 ctx
			c.SetUserValue("uid", claims.UserID)
			c.SetUserValue("username", claims.Username)

			next(c)
		}
	}
}

func parseBearer(h string) (string, error) {
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid bearer")
	}
	return parts[1], nil
}
