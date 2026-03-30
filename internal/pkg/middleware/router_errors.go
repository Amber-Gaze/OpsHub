package middleware

import (
	"encoding/json"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// AttachRouterErrors 为 fasthttp/router 设置统一的 404 / 405 JSON 响应。
func AttachRouterErrors(r *router.Router) {
	r.NotFound = jsonErrorHandler(fasthttp.StatusNotFound, "not found")
	r.MethodNotAllowed = jsonErrorHandler(fasthttp.StatusMethodNotAllowed, "method not allowed")
}

func jsonErrorHandler(code int, msg string) fasthttp.RequestHandler {
	body, _ := json.Marshal(map[string]string{"error": msg})
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(code)
		ctx.SetContentType("application/json")
		ctx.SetBody(body)
	}
}
