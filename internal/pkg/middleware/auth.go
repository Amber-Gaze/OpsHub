package middleware

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

type AuthConfig struct {
	AuthURL string
}

func Auth(cfg AuthConfig) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			token := string(c.Request.Header.Peek("Authorization"))
			if token == "" {
				c.Abort(fasthttp.StatusUnauthorized, "missing token")
				return
			}

			if !authorize(cfg.AuthURL, token, string(c.Path()), string(c.Method())) {
				c.Abort(fasthttp.StatusForbidden, "permission denied")
				return
			}

			next(c)
		}
	}
}

func authorize(url, token, res, act string) bool {
	reqBody := map[string]string{
		"token":    token,
		"resource": res,
		"action":   act,
	}
	b, _ := json.Marshal(reqBody)

	var req fasthttp.Request
	var resp fasthttp.Response

	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBody(b)

	client := fasthttp.Client{}
	if err := client.Do(&req, &resp); err != nil {
		return false
	}

	var result struct {
		Allow bool `json:"allow"`
	}
	_ = json.Unmarshal(resp.Body(), &result)

	return result.Allow
}
