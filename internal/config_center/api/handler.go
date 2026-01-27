package api

import (
	"encoding/json"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/valyala/fasthttp"
)

func GetConfig(c *middleware.Context) {
	key := c.UserValue("key").(string)
	c.JSON(200, map[string]string{
		"key":   key,
		"value": "mock",
	})
}

func authorize(token, res, act string) bool {
	req := map[string]string{
		"token":    token,
		"resource": res,
		"action":   act,
	}
	b, _ := json.Marshal(req)

	var resp fasthttp.Response
	var reqHTTP fasthttp.Request
	reqHTTP.SetRequestURI("http://127.0.0.1:8081/authorize")
	reqHTTP.Header.SetMethod("POST")
	reqHTTP.Header.SetContentType("application/json")
	reqHTTP.SetBody(b)

	client := fasthttp.Client{}
	if err := client.Do(&reqHTTP, &resp); err != nil {
		return false
	}

	return resp.StatusCode() == fasthttp.StatusOK
}
