package api

import (
	"errors"
	"strings"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/valyala/fasthttp"
)

// RequireAuthDecision 在 JWT 校验通过后调用 IAM /authorize，将鉴权决策写入 Context，供转发到配置中心时携带 X-Auth-* 头。
func RequireAuthDecision(svc *Service) middleware.Middleware {
	return func(next middleware.HandlerFunc) middleware.HandlerFunc {
		return func(c *middleware.Context) {
			authHeader := string(c.Request.Header.Peek("Authorization"))
			token := strings.TrimSpace(authHeader)
			if len(token) > 7 && strings.EqualFold(token[:7], "Bearer ") {
				token = strings.TrimSpace(token[7:])
			}
			if token == "" {
				c.Abort(fasthttp.StatusUnauthorized, "missing token")
				return
			}
			resource := string(c.Path())
			if resource == "" {
				resource = "/"
			}
			action := string(c.Method())
			if action == "" {
				action = "UNKNOWN"
			}
			decision, err := svc.Authorize(token, resource, action)
			if err != nil {
				var ae *AuthError
				if errors.As(err, &ae) && ae.Status >= 400 && ae.Status < 500 {
					// 401/403 等由 IAM 判定的结果原样透传
					c.Abort(ae.Status, "permission denied")
				} else {
					c.Abort(fasthttp.StatusBadGateway, "authorize failed: "+err.Error())
				}
				return
			}
			c.SetAuthDecision(decision)
			next(c)
		}
	}
}
