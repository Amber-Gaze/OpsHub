package api

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/valyala/fasthttp"
)

type setServiceModulesRequest struct {
	Modules []string `json:"modules"`
}

// GetServiceModules 查看服务账号订阅的模块（管理员）。
func (h *Handler) GetServiceModules(c *middleware.Context) {
	name := strings.TrimSpace(pathParam(c, "name"))
	if name == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid name")
		return
	}
	list, err := h.svc.ListServiceModules(context.Background(), name)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	modules := make([]string, 0, len(list))
	for _, sm := range list {
		modules = append(modules, sm.Path)
	}
	c.JSON(fasthttp.StatusOK, map[string]any{"username": name, "modules": modules, "detail": list})
}

// SetServiceModules 覆盖式注册服务账号的模块订阅（仅管理员）：注册即授 read，默认只读。
func (h *Handler) SetServiceModules(c *middleware.Context) {
	name := strings.TrimSpace(pathParam(c, "name"))
	if name == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid name")
		return
	}
	var req setServiceModulesRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	added, err := h.svc.SetServiceModules(context.Background(), name, req.Modules)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	modules := make([]string, 0, len(added))
	for _, sm := range added {
		modules = append(modules, sm.Path)
	}
	c.JSON(fasthttp.StatusOK, map[string]any{"username": name, "modules": modules})
}

// RemoveServiceModule 取消单个模块订阅（仅管理员，body {"path":"pay/gateway"}）。
func (h *Handler) RemoveServiceModule(c *middleware.Context) {
	name := strings.TrimSpace(pathParam(c, "name"))
	if name == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid name")
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Path) == "" {
		c.Abort(fasthttp.StatusBadRequest, "path required")
		return
	}
	if err := h.svc.RemoveServiceModule(context.Background(), name, req.Path); err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	c.SetStatusCode(fasthttp.StatusNoContent)
	c.SetBody(nil)
}

// pathParam 读取路径参数（fasthttp/router {name} 语法）。
func pathParam(c *middleware.Context, name string) string {
	raw, _ := c.UserValue(name).(string)
	return strings.TrimSpace(raw)
}
