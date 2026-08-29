package api

import (
	"strings"

	json "github.com/json-iterator/go"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/casbinx"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/valyala/fasthttp"
)

type policyRuleRequest struct {
	Sub string `json:"sub"`
	Obj string `json:"obj"`
	Act string `json:"act"`
}

type roleBindRequest struct {
	User string `json:"user"`
	Role string `json:"role"`
}

func (h *Handler) ListPolicies(c *middleware.Context) {
	p := h.svc.ListPolicies()
	g := h.svc.ListGroupingPolicies()
	c.JSON(fasthttp.StatusOK, map[string]any{
		"policies":  p,
		"groupings": g,
	})
}

func (h *Handler) AddPolicyRule(c *middleware.Context) {
	var req policyRuleRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	req.Sub, req.Obj, req.Act = strings.TrimSpace(req.Sub), strings.TrimSpace(req.Obj), strings.TrimSpace(req.Act)
	if req.Sub == "" || req.Obj == "" || req.Act == "" {
		c.Abort(fasthttp.StatusBadRequest, "sub, obj, act required")
		return
	}
	if !h.canGrantPolicy(c, req.Obj) {
		c.Abort(fasthttp.StatusForbidden, "forbidden")
		return
	}
	ok, err := h.svc.AddPolicy(req.Sub, req.Obj, req.Act)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		c.Abort(fasthttp.StatusConflict, "policy already exists")
		return
	}
	c.JSON(fasthttp.StatusCreated, map[string]string{"message": "policy added"})
}

func (h *Handler) RemovePolicyRule(c *middleware.Context) {
	var req policyRuleRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	req.Sub, req.Obj, req.Act = strings.TrimSpace(req.Sub), strings.TrimSpace(req.Obj), strings.TrimSpace(req.Act)
	if req.Sub == "" || req.Obj == "" || req.Act == "" {
		c.Abort(fasthttp.StatusBadRequest, "sub, obj, act required")
		return
	}
	if !h.canGrantPolicy(c, req.Obj) {
		c.Abort(fasthttp.StatusForbidden, "forbidden")
		return
	}
	ok, err := h.svc.RemovePolicy(req.Sub, req.Obj, req.Act)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		c.Abort(fasthttp.StatusNotFound, "policy not found")
		return
	}
	c.JSON(fasthttp.StatusOK, map[string]string{"message": "policy removed"})
}

func (h *Handler) AddRoleBinding(c *middleware.Context) {
	var req roleBindRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	req.User, req.Role = strings.TrimSpace(req.User), strings.TrimSpace(req.Role)
	if req.User == "" || req.Role == "" {
		c.Abort(fasthttp.StatusBadRequest, "user, role required")
		return
	}
	if !c.IsAdmin {
		c.Abort(fasthttp.StatusForbidden, "admin required")
		return
	}
	ok, err := h.svc.AddRoleForUser(req.User, req.Role)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		c.Abort(fasthttp.StatusConflict, "binding already exists")
		return
	}
	c.JSON(fasthttp.StatusCreated, map[string]string{"message": "role bound"})
}

func (h *Handler) RemoveRoleBinding(c *middleware.Context) {
	var req roleBindRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	req.User, req.Role = strings.TrimSpace(req.User), strings.TrimSpace(req.Role)
	if req.User == "" || req.Role == "" {
		c.Abort(fasthttp.StatusBadRequest, "user, role required")
		return
	}
	if !c.IsAdmin {
		c.Abort(fasthttp.StatusForbidden, "admin required")
		return
	}
	ok, err := h.svc.RemoveRoleForUser(req.User, req.Role)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		c.Abort(fasthttp.StatusNotFound, "binding not found")
		return
	}
	c.JSON(fasthttp.StatusOK, map[string]string{"message": "role unbound"})
}

// configGrantRequest 面向控制台的友好授权入参：按业务/模块/项 + 动作授权。
type configGrantRequest struct {
	Sub      string `json:"sub"`
	Business string `json:"business"`
	Module   string `json:"module,omitempty"`
	Item     string `json:"item,omitempty"`
	Act      string `json:"act"` // read|write|delete|grant|*
}

func (h *Handler) ConfigGrant(c *middleware.Context) {
	var req configGrantRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	req.Sub = strings.TrimSpace(req.Sub)
	req.Business = strings.TrimSpace(req.Business)
	req.Module = strings.TrimSpace(req.Module)
	req.Item = strings.TrimSpace(req.Item)
	req.Act = strings.TrimSpace(req.Act)
	if req.Sub == "" || req.Business == "" || req.Act == "" {
		c.Abort(fasthttp.StatusBadRequest, "sub, business, act required")
		return
	}
	obj := casbinx.BuildConfigObj(req.Business, req.Module, req.Item)
	if !h.canGrantPolicy(c, obj) {
		c.Abort(fasthttp.StatusForbidden, "forbidden")
		return
	}
	ok, err := h.svc.AddPolicy(req.Sub, obj, req.Act)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		c.Abort(fasthttp.StatusConflict, "policy already exists")
		return
	}
	c.JSON(fasthttp.StatusCreated, map[string]string{
		"message": "granted",
		"sub":     req.Sub,
		"obj":     obj,
		"act":     req.Act,
	})
}

func (h *Handler) ConfigRevoke(c *middleware.Context) {
	var req configGrantRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	req.Sub = strings.TrimSpace(req.Sub)
	req.Business = strings.TrimSpace(req.Business)
	req.Module = strings.TrimSpace(req.Module)
	req.Item = strings.TrimSpace(req.Item)
	req.Act = strings.TrimSpace(req.Act)
	if req.Sub == "" || req.Business == "" || req.Act == "" {
		c.Abort(fasthttp.StatusBadRequest, "sub, business, act required")
		return
	}
	obj := casbinx.BuildConfigObj(req.Business, req.Module, req.Item)
	if !h.canGrantPolicy(c, obj) {
		c.Abort(fasthttp.StatusForbidden, "forbidden")
		return
	}
	ok, err := h.svc.RemovePolicy(req.Sub, obj, req.Act)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		c.Abort(fasthttp.StatusNotFound, "policy not found")
		return
	}
	c.JSON(fasthttp.StatusOK, map[string]string{"message": "revoked"})
}

// canGrantPolicy：终极管理员（IsAdmin）可对任意 obj 写策略；否则需对该 obj 拥有 grant。
func (h *Handler) canGrantPolicy(c *middleware.Context, obj string) bool {
	if c.IsAdmin {
		return true
	}
	ok, err := h.svc.EnforceConfig(c.Username, obj, "grant")
	return err == nil && ok
}
