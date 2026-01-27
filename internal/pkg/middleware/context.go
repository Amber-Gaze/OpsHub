package middleware

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

type Context struct {
	*fasthttp.RequestCtx

	UserID    string
	Username  string
	Roles     []string
	RequestID string
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
