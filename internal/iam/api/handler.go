package api

import (
	"context"
	"errors"
	"strings"
	"time"

	json "github.com/json-iterator/go"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/passhash"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"github.com/valyala/fasthttp"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Healthz(c *middleware.Context) {
	c.JSON(fasthttp.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Readyz(c *middleware.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := store.Client().Ping(ctx); err != nil {
		c.Abort(fasthttp.StatusServiceUnavailable, err.Error())
		return
	}
	c.JSON(fasthttp.StatusOK, map[string]string{"status": "ready"})
}

type loginRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
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

	result, err := h.svc.Login(c, req)
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

func (h *Handler) Logout(c *middleware.Context) {
	c.JSON(fasthttp.StatusOK, map[string]string{"message": "logged out"})
}

func (h *Handler) Signup(c *middleware.Context) {
	var req signupRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.Signup(c, req); err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(fasthttp.StatusCreated, map[string]string{"message": "user created"})
}

func (h *Handler) Refresh(c *middleware.Context) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Refresh(c, req.Token)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrTokenExpired) {
			c.Abort(fasthttp.StatusUnauthorized, err.Error())
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

// userResponse 对外返回用户信息，不含密码
type userResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	IsAdmin   bool   `json:"is_admin"`
	Status    int    `json:"status"`
	LoginedAt int64  `json:"logined_at"`
}

func userToResponse(u *store.User) userResponse {
	if u == nil {
		return userResponse{}
	}
	return userResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Phone:     u.Phone,
		IsAdmin:   u.IsAdmin,
		Status:    u.Status,
		LoginedAt: u.LoginedAt,
	}
}

func (h *Handler) ListUsers(c *middleware.Context) {
	ctx := context.Background()
	list, err := store.Client().Users().List(ctx)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]userResponse, 0, len(list))
	for _, u := range list {
		resp = append(resp, userToResponse(u))
	}
	c.JSON(fasthttp.StatusOK, resp)
}

func (h *Handler) GetUser(c *middleware.Context) {
	name, _ := c.UserValue("name").(string)
	name = strings.TrimSpace(name)
	if name == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid name")
		return
	}
	ctx := context.Background()
	u, err := store.Client().Users().Get(ctx, name)
	if err != nil {
		c.Abort(fasthttp.StatusNotFound, "user not found")
		return
	}
	c.JSON(fasthttp.StatusOK, userToResponse(u))
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ChangePassword(c *middleware.Context) {
	name, _ := c.UserValue("name").(string)
	name = strings.TrimSpace(name)
	if name == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid name")
		return
	}
	var req changePasswordRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	if err := validatePassword(req.NewPassword); err != nil {
		c.Abort(fasthttp.StatusBadRequest, err.Error())
		return
	}
	ctx := context.Background()
	u, err := store.Client().Users().Get(ctx, name)
	if err != nil {
		c.Abort(fasthttp.StatusNotFound, "user not found")
		return
	}
	isSelf := strings.EqualFold(name, c.Username)
	if isSelf {
		if u.ComparePassword(req.OldPassword) != nil {
			c.Abort(fasthttp.StatusBadRequest, "invalid old password")
			return
		}
	} else if !c.IsAdmin {
		c.Abort(fasthttp.StatusForbidden, "forbidden")
		return
	}
	hashed, err := passhash.Hash(req.NewPassword)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	u.Password = hashed
	if err := store.Client().Users().Update(ctx, u); err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(fasthttp.StatusOK, map[string]string{"message": "password updated"})
}

type updateUserRequest struct {
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	IsAdmin *bool  `json:"is_admin,omitempty"`
	Status  *int   `json:"status,omitempty"`
}

func (h *Handler) UpdateUser(c *middleware.Context) {
	name, _ := c.UserValue("name").(string)
	name = strings.TrimSpace(name)
	if name == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid name")
		return
	}
	var req updateUserRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	if req.IsAdmin != nil || req.Status != nil {
		if !c.IsAdmin {
			c.Abort(fasthttp.StatusForbidden, "admin required")
			return
		}
	}
	ctx := context.Background()
	u, err := store.Client().Users().Get(ctx, name)
	if err != nil {
		c.Abort(fasthttp.StatusNotFound, "user not found")
		return
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	if req.Phone != "" {
		u.Phone = req.Phone
	}
	if req.IsAdmin != nil {
		u.IsAdmin = *req.IsAdmin
	}
	if req.Status != nil {
		u.Status = *req.Status
	}
	if err := store.Client().Users().Update(ctx, u); err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(fasthttp.StatusOK, userToResponse(u))
}

func (h *Handler) DeleteUser(c *middleware.Context) {
	name, _ := c.UserValue("name").(string)
	name = strings.TrimSpace(name)
	if name == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid name")
		return
	}
	if !c.IsAdmin && !strings.EqualFold(name, c.Username) {
		c.Abort(fasthttp.StatusForbidden, "forbidden")
		return
	}
	ctx := context.Background()
	if err := store.Client().Users().Delete(ctx, name); err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	c.SetStatusCode(fasthttp.StatusNoContent)
	c.SetBody(nil)
}

type deleteUsersRequest struct {
	Usernames []string `json:"usernames"`
}

func (h *Handler) DeleteUsers(c *middleware.Context) {
	var req deleteUsersRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	ctx := context.Background()
	for _, name := range req.Usernames {
		name = strings.TrimSpace(name)
		if name != "" {
			_ = store.Client().Users().Delete(ctx, name)
		}
	}
	c.SetStatusCode(fasthttp.StatusNoContent)
	c.SetBody(nil)
}
