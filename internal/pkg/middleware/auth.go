package middleware

import (
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
)

type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var jwtKey = []byte("secret") // 实际从 config 读

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

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return jwtKey, nil
			})

			if err != nil || !token.Valid {
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
