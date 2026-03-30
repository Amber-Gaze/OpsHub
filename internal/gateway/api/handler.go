package api

import (
	"fmt"
	"path"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/valyala/fasthttp"
)

type HandlerOptions struct {
	AuthLoginPath string
}

type Handler struct {
	svc           *Service
	authLoginPath string
}

func NewHandler(svc *Service, opts HandlerOptions) *Handler {
	loginPath := opts.AuthLoginPath
	if loginPath == "" {
		loginPath = "/login"
	}
	return &Handler{svc: svc, authLoginPath: loginPath}
}

func (h *Handler) Health(c *middleware.Context) {
	c.JSON(fasthttp.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Ready(c *middleware.Context) {
	c.JSON(fasthttp.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) Login(c *middleware.Context) {
	status, body, contentType, err := h.svc.ForwardAuth(fasthttp.MethodPost, "/login", c.PostBody(), h.collectHeaders(c, false))
	if err != nil {
		c.Abort(fasthttp.StatusBadGateway, err.Error())
		return
	}
	writeProxyResponse(c, status, contentType, body)
}
func (h *Handler) Logout(c *middleware.Context) {
	status, body, contentType, err := h.svc.ForwardAuth(fasthttp.MethodPost, "/logout", c.PostBody(), h.collectHeaders(c, false))
	if err != nil {
		c.Abort(fasthttp.StatusBadGateway, err.Error())
		return
	}
	writeProxyResponse(c, status, contentType, body)
}
func (h *Handler) Refresh(c *middleware.Context) {
	status, body, contentType, err := h.svc.ForwardAuth(fasthttp.MethodPost, "/refresh", c.PostBody(), h.collectHeaders(c, false))
	if err != nil {
		c.Abort(fasthttp.StatusBadGateway, err.Error())
		return
	}
	writeProxyResponse(c, status, contentType, body)
}

func (h *Handler) ListConfigs(c *middleware.Context) {
	status, body, contentType, err := h.forwardConfig(c, fasthttp.MethodGet, "/configs", nil)
	if err != nil {
		c.Abort(fasthttp.StatusBadGateway, err.Error())
		return
	}
	writeProxyResponse(c, status, contentType, body)
}

func (h *Handler) GetConfig(c *middleware.Context) {
	key := h.extractKey(c)
	if key == "" {
		return
	}

	status, body, contentType, err := h.forwardConfig(c, fasthttp.MethodGet, "/configs/"+key, nil)
	if err != nil {
		c.Abort(fasthttp.StatusBadGateway, err.Error())
		return
	}
	writeProxyResponse(c, status, contentType, body)
}

func (h *Handler) CreateConfig(c *middleware.Context) {
	status, body, contentType, err := h.forwardConfig(c, fasthttp.MethodPost, "/configs", c.PostBody())
	if err != nil {
		c.Abort(fasthttp.StatusBadGateway, err.Error())
		return
	}
	writeProxyResponse(c, status, contentType, body)
}

func (h *Handler) UpdateConfig(c *middleware.Context) {
	key := h.extractKey(c)
	if key == "" {
		return
	}

	status, body, contentType, err := h.forwardConfig(c, fasthttp.MethodPut, "/configs/"+key, c.PostBody())
	if err != nil {
		c.Abort(fasthttp.StatusBadGateway, err.Error())
		return
	}
	writeProxyResponse(c, status, contentType, body)
}

func (h *Handler) DeleteConfig(c *middleware.Context) {
	key := h.extractKey(c)
	if key == "" {
		return
	}

	status, body, contentType, err := h.forwardConfig(c, fasthttp.MethodDelete, "/configs/"+key, nil)
	if err != nil {
		c.Abort(fasthttp.StatusBadGateway, err.Error())
		return
	}
	writeProxyResponse(c, status, contentType, body)
}

func (h *Handler) forwardConfig(c *middleware.Context, method, targetPath string, body []byte) (int, []byte, string, error) {
	headers := h.collectHeaders(c, true)
	cleanTarget := targetPath
	if cleanTarget == "" {
		cleanTarget = "/"
	}
	if cleanTarget[0] != '/' {
		cleanTarget = "/" + cleanTarget
	}
	cleanTarget = path.Clean(cleanTarget)
	internalPath := fmt.Sprintf("/internal%s", cleanTarget)
	if qs := c.URI().QueryString(); len(qs) > 0 {
		internalPath = fmt.Sprintf("%s?%s", internalPath, qs)
	}
	return h.svc.ForwardConfig(method, internalPath, body, headers)
}

func (h *Handler) extractKey(c *middleware.Context) string {
	raw := c.UserValue("key")
	key, ok := raw.(string)
	if !ok || key == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid key")
		return ""
	}
	return path.Clean("/" + key)[1:]
}

func (h *Handler) collectHeaders(c *middleware.Context, includeDecision bool) map[string]string {
	headers := map[string]string{}
	if ct := string(c.Request.Header.ContentType()); ct != "" {
		headers["Content-Type"] = ct
	}
	if auth := string(c.Request.Header.Peek("Authorization")); auth != "" {
		headers["Authorization"] = auth
	}
	if c.RequestID != "" {
		headers["X-Request-ID"] = c.RequestID
	}
	headers["X-Forwarded-For"] = c.RemoteIP().String()
	headers["X-Forwarded-Proto"] = string(c.URI().Scheme())
	if includeDecision {
		if decision := c.GetAuthDecision(); decision != nil {
			headers["X-Auth-Decision"] = decision.Decision
			headers["X-Auth-Subject"] = decision.Subject
			headers["X-Auth-Signature"] = decision.Signature
			headers["X-Auth-Action"] = decision.Action
			headers["X-Auth-Resource"] = decision.Resource
		}
	}
	return headers
}

func writeProxyResponse(c *middleware.Context, status int, contentType string, body []byte) {
	if status == 0 {
		status = fasthttp.StatusBadGateway
	}
	c.SetStatusCode(status)
	if contentType != "" {
		c.SetContentType(contentType)
	}
	if len(body) > 0 {
		c.SetBody(body)
	} else {
		c.SetBody(nil)
	}
}
