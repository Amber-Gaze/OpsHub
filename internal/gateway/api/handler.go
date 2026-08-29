package api

import (
	"fmt"
	"path"
	"strings"

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

// GetTree 转发完整「业务 → 模块 → 具体项」层级树请求到配置中心。
func (h *Handler) GetTree(c *middleware.Context) {
	h.forwardTreePath(c, fasthttp.MethodGet, "", nil)
}

// GetBusiness 转发单个业务子树请求。
func (h *Handler) GetBusiness(c *middleware.Context) {
	h.forwardTreePath(c, fasthttp.MethodGet, h.joinParams(c, "business"), nil)
}

// GetModule 转发 business/module 下配置项列表请求。
func (h *Handler) GetModule(c *middleware.Context) {
	h.forwardTreePath(c, fasthttp.MethodGet, h.joinParams(c, "business", "module"), nil)
}

// GetItem 转发 business/module/name 具体配置项请求。
func (h *Handler) GetItem(c *middleware.Context) {
	h.forwardTreePath(c, fasthttp.MethodGet, h.joinParams(c, "business", "module", "name"), nil)
}

// UpdateItem 转发 business/module/name 配置项更新请求。
func (h *Handler) UpdateItem(c *middleware.Context) {
	h.forwardTreePath(c, fasthttp.MethodPut, h.joinParams(c, "business", "module", "name"), c.PostBody())
}

// DeleteItem 转发 business/module/name 配置项删除请求。
func (h *Handler) DeleteItem(c *middleware.Context) {
	h.forwardTreePath(c, fasthttp.MethodDelete, h.joinParams(c, "business", "module", "name"), nil)
}

// joinParams 从路径参数拼接子路径；任一参数缺失或非法（"."、".."）时中止请求并返回空串。
func (h *Handler) joinParams(c *middleware.Context, names ...string) string {
	parts := make([]string, 0, len(names))
	for _, n := range names {
		raw, _ := c.UserValue(n).(string)
		v := strings.TrimSpace(raw)
		if v == "" || v == "." || v == ".." || strings.ContainsAny(v, "/\\") {
			c.Abort(fasthttp.StatusBadRequest, "invalid path parameter: "+n)
			return ""
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, "/")
}

// forwardTreePath 将 /configs/tree[/...] 请求转发到配置中心 /internal/configs/tree[/...]。
func (h *Handler) forwardTreePath(c *middleware.Context, method, subPath string, body []byte) {
	target := "/configs/tree"
	if subPath != "" {
		target += "/" + subPath
	}
	status, respBody, contentType, err := h.forwardConfig(c, method, target, body)
	if err != nil {
		c.Abort(fasthttp.StatusBadGateway, err.Error())
		return
	}
	writeProxyResponse(c, status, contentType, respBody)
}

// ForwardUsers 将用户管理请求原样透传到 IAM（IAM 自行做 JWT/管理员校验）。
// 网关作为统一入口：后续新增的用户相关接口只需在 IAM 侧注册路由，网关自动透传。
func (h *Handler) ForwardUsers(c *middleware.Context) {
	h.forwardIAM(c, string(c.Method()), string(c.Path()), c.PostBody())
}

// ForwardPolicies 将策略/授权管理请求原样透传到 IAM。
func (h *Handler) ForwardPolicies(c *middleware.Context) {
	h.forwardIAM(c, string(c.Method()), string(c.Path()), c.PostBody())
}

// forwardIAM 透传请求到 IAM 服务（保留 Content-Type/Authorization/RequestID 等）。
func (h *Handler) forwardIAM(c *middleware.Context, method, path string, body []byte) {
	status, respBody, contentType, err := h.svc.ForwardAuth(method, path, body, h.collectHeaders(c, false))
	if err != nil {
		c.Abort(fasthttp.StatusBadGateway, err.Error())
		return
	}
	writeProxyResponse(c, status, contentType, respBody)
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
