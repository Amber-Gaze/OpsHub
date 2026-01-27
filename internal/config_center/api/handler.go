package api

import (
	"encoding/json"
	"strings"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/authutil"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/valyala/fasthttp"
)

type Handler struct {
	svc            *Service
	decisionSecret []byte
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, decisionSecret: authutil.DefaultDecisionSecret}
}

type createConfigRequest struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Operator string `json:"operator"`
}

type updateConfigRequest struct {
	Value    string `json:"value"`
	Operator string `json:"operator"`
}

func (h *Handler) List(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	c.JSON(fasthttp.StatusOK, h.svc.List())
}

func (h *Handler) Get(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	key := c.UserValue("key")
	keyStr, ok := key.(string)
	if !ok || keyStr == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid key")
		return
	}

	cfg, exists := h.svc.Get(keyStr)
	if !exists {
		c.Abort(fasthttp.StatusNotFound, ErrConfigNotFound.Error())
		return
	}

	c.JSON(fasthttp.StatusOK, cfg)
}

func (h *Handler) Create(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	var req createConfigRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}

	cfg, err := h.svc.Create(req.Key, req.Value, operatorFromContext(c, req.Operator))
	if err != nil {
		switch err {
		case ErrConfigExists:
			c.Abort(fasthttp.StatusConflict, err.Error())
		default:
			c.Abort(fasthttp.StatusBadRequest, err.Error())
		}
		return
	}

	c.JSON(fasthttp.StatusCreated, cfg)
}

func (h *Handler) Update(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	key := c.UserValue("key")
	keyStr, ok := key.(string)
	if !ok || keyStr == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid key")
		return
	}

	var req updateConfigRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}

	cfg, err := h.svc.Update(keyStr, req.Value, operatorFromContext(c, req.Operator))
	if err != nil {
		switch err {
		case ErrConfigNotFound:
			c.Abort(fasthttp.StatusNotFound, err.Error())
		default:
			c.Abort(fasthttp.StatusBadRequest, err.Error())
		}
		return
	}

	c.JSON(fasthttp.StatusOK, cfg)
}

func (h *Handler) Delete(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	key := c.UserValue("key")
	keyStr, ok := key.(string)
	if !ok || keyStr == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid key")
		return
	}

	if err := h.svc.Delete(keyStr); err != nil {
		if err == ErrConfigNotFound {
			c.Abort(fasthttp.StatusNotFound, err.Error())
		} else {
			c.Abort(fasthttp.StatusBadRequest, err.Error())
		}
		return
	}

	c.SetStatusCode(fasthttp.StatusNoContent)
	c.SetBody(nil)
}

func operatorFromContext(c *middleware.Context, override string) string {
	if override != "" {
		return override
	}
	if c.Username != "" {
		return c.Username
	}
	return "system"
}

const (
	headerDecision  = "X-Auth-Decision"
	headerSignature = "X-Auth-Signature"
	headerSubject   = "X-Auth-Subject"
	headerAction    = "X-Auth-Action"
	headerResource  = "X-Auth-Resource"
)

func (h *Handler) requireAuthDecision(c *middleware.Context) bool {
	decision := strings.TrimSpace(string(c.Request.Header.Peek(headerDecision)))
	signature := strings.TrimSpace(string(c.Request.Header.Peek(headerSignature)))
	subject := strings.TrimSpace(string(c.Request.Header.Peek(headerSubject)))
	action := strings.TrimSpace(string(c.Request.Header.Peek(headerAction)))
	resource := strings.TrimSpace(string(c.Request.Header.Peek(headerResource)))

	if decision == "" || signature == "" || subject == "" {
		c.Abort(fasthttp.StatusForbidden, "missing authorization context")
		return false
	}

	if !authutil.Verify(decision, signature, h.decisionSecret) {
		c.Abort(fasthttp.StatusForbidden, "invalid authorization signature")
		return false
	}

	if action == "" {
		action = string(c.Method())
	}
	if resource == "" {
		resource = string(c.Path())
	}

	h.injectDecision(c, subject, action, resource, decision, signature)
	return true
}

func (h *Handler) injectDecision(c *middleware.Context, subject, action, resource, decision, signature string) {
	c.UserID = subject
	c.Username = subject
	c.SetAuthDecision(&middleware.AuthDecision{
		Allow:     true,
		Subject:   subject,
		Action:    action,
		Resource:  resource,
		Decision:  decision,
		Signature: signature,
	})
}
