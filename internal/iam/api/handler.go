package api

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/valyala/fasthttp"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type loginRequest struct {
	User string `json:"user"`
}

type loginResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresAt string `json:"expires_at"`
}

func (h *Handler) Login(c *middleware.Context) {
	var req loginRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Login(req.User)
	if err != nil {
		if errors.Is(err, ErrInvalidUser) {
			c.Abort(fasthttp.StatusBadRequest, err.Error())
		} else {
			c.Abort(fasthttp.StatusInternalServerError, err.Error())
		}
		return
	}

	c.JSON(fasthttp.StatusOK, loginResponse{
		Token:     result.Token,
		TokenType: result.TokenType,
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	})
}

type authorizeRequest struct {
	Token    string `json:"token"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type authorizeResponse struct {
	Allow     bool   `json:"allow"`
	Subject   string `json:"subject"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Decision  string `json:"decision"`
	Signature string `json:"signature"`
}

func (h *Handler) Authorize(c *middleware.Context) {
	var req authorizeRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}

	decision, err := h.svc.Authorize(req.Token, req.Resource, req.Action)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidToken):
			c.Abort(fasthttp.StatusUnauthorized, err.Error())
		case errors.Is(err, ErrTokenExpired):
			c.Abort(fasthttp.StatusUnauthorized, err.Error())
		default:
			c.Abort(fasthttp.StatusForbidden, err.Error())
		}
		return
	}

	c.JSON(fasthttp.StatusOK, authorizeResponse{
		Allow:     decision.Allow,
		Subject:   decision.Subject,
		Action:    decision.Action,
		Resource:  decision.Resource,
		Decision:  decision.Decision,
		Signature: decision.Signature,
	})
}
