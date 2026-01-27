package api

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

type LoginReq struct {
	User string `json:"user"`
}

func Login(ctx *fasthttp.RequestCtx) {
	var req LoginReq
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	resp := map[string]string{
		"token": "mock-token-" + req.User,
	}

	b, _ := json.Marshal(resp)
	ctx.SetContentType("application/json")
	ctx.SetBody(b)
}

type AuthzReq struct {
	Token    string `json:"token"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func Authorize(ctx *fasthttp.RequestCtx) {
	var req AuthzReq
	_ = json.Unmarshal(ctx.PostBody(), &req)

	allow := req.Token != ""

	resp := map[string]bool{"allow": allow}
	b, _ := json.Marshal(resp)

	ctx.SetContentType("application/json")
	ctx.SetBody(b)
}
