package middleware

import (
	"crypto/rand"
	"encoding/hex"
)

// RequestID 确保请求链路有可追踪的 X-Request-ID（与参考 iam 的链路追踪习惯一致）。
func RequestID() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			rid := string(c.Request.Header.Peek("X-Request-ID"))
			if rid == "" {
				rid = randomID()
			}
			c.RequestID = rid
			c.Response.Header.Set("X-Request-ID", rid)
			next(c)
		}
	}
}

func randomID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}
