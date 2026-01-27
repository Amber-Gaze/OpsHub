package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/valyala/fasthttp"
)

type AuthConfig struct {
	AuthURL string
}

func Auth(cfg AuthConfig) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			token := strings.TrimSpace(string(c.Request.Header.Peek("Authorization")))
			if token == "" {
				c.Abort(fasthttp.StatusUnauthorized, "missing token")
				return
			}

			decision, err := authorize(cfg.AuthURL, token, string(c.Path()), string(c.Method()))
			if err != nil {
				c.Abort(fasthttp.StatusBadGateway, err.Error())
				return
			}
			if decision == nil || !decision.Allow {
				c.Abort(fasthttp.StatusForbidden, "permission denied")
				return
			}

			c.UserID = decision.Subject
			c.Username = decision.Subject
			c.SetAuthDecision(decision)

			next(c)
		}
	}
}

func authorize(url, token, res, act string) (*AuthDecision, error) {
	if strings.TrimSpace(url) == "" {
		return nil, errors.New("auth url is not configured")
	}
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
		return nil, err
	}

	if resp.StatusCode() >= 500 {
		return nil, fmt.Errorf("authorize upstream error: %d", resp.StatusCode())
	}

	var result struct {
		Allow     bool   `json:"allow"`
		Subject   string `json:"subject"`
		Decision  string `json:"decision"`
		Signature string `json:"signature"`
		Action    string `json:"action"`
		Resource  string `json:"resource"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}

	return &AuthDecision{
		Allow:     result.Allow,
		Subject:   result.Subject,
		Action:    choose(result.Action, act),
		Resource:  choose(result.Resource, res),
		Decision:  result.Decision,
		Signature: result.Signature,
	}, nil
}

func choose(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
