package middleware

import (
	"encoding/json"
	"net"

	"github.com/valyala/fasthttp"
)

type Context struct {
	*fasthttp.RequestCtx

	UserID    string
	Username  string
	IsAdmin   bool
	Roles     []string
	RequestID string
	Decision  *AuthDecision
}

func NewContext(ctx *fasthttp.RequestCtx) *Context {
	return &Context{
		RequestCtx: ctx,
		RequestID:  string(ctx.Request.Header.Peek("X-Request-ID")),
	}
}

func (c *Context) JSON(code int, v any) {
	b, _ := json.Marshal(v)
	c.SetStatusCode(code)
	c.SetContentType("application/json")
	c.SetBody(b)
}

func (c *Context) Abort(code int, msg string) {
	c.JSON(code, map[string]string{"error": msg})
}

type AuthDecision struct {
	Allow     bool   `json:"allow"`
	Subject   string `json:"subject"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Decision  string `json:"decision"`
	Signature string `json:"signature"`
}

func (c *Context) SetAuthDecision(d *AuthDecision) {
	c.Decision = d
}

func (c *Context) GetAuthDecision() *AuthDecision {
	return c.Decision
}

// RemoteIP returns the client address for logging/forwarding (e.g. X-Forwarded-For).
func (c *Context) RemoteIP() net.Addr {
	return c.RemoteAddr()
}
