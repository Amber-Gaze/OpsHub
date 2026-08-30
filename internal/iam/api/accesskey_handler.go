package api

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"github.com/valyala/fasthttp"
)

// accessKeyResponse 凭证响应；Secret 仅在创建时返回一次。
type accessKeyResponse struct {
	Username        string `json:"username"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret,omitempty"`
	Expires         int64  `json:"expires"`
	Status          int    `json:"status"`
	Description     string `json:"description"`
	CreatedAt       int64  `json:"created_at"`
}

func accessKeyToResponse(ak *store.AccessKey, withSecret bool) accessKeyResponse {
	r := accessKeyResponse{
		Username:    ak.Username,
		AccessKeyID: ak.AccessKeyID,
		Expires:     ak.Expires,
		Status:      ak.Status,
		Description: ak.Description,
		CreatedAt:   ak.CreatedAt,
	}
	if withSecret {
		r.AccessKeySecret = ak.AccessKeySecret
	}
	return r
}

type createAccessKeyRequest struct {
	Username    string `json:"username,omitempty"` // 管理员可指定；默认当前用户
	Description string `json:"description,omitempty"`
	Expires     int64  `json:"expires,omitempty"` // Unix 秒；0=永不过期
}

// CreateAccessKey 创建服务凭证（本人，或管理员给任意服务账号创建）。
func (h *Handler) CreateAccessKey(c *middleware.Context) {
	var req createAccessKeyRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = c.Username
	}
	username = strings.TrimSpace(username)
	if username == "" {
		c.Abort(fasthttp.StatusBadRequest, "username required")
		return
	}
	if username != c.Username && !c.IsAdmin {
		c.Abort(fasthttp.StatusForbidden, "forbidden")
		return
	}
	ak, err := h.svc.CreateAccessKey(context.Background(), username, strings.TrimSpace(req.Description), req.Expires)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(fasthttp.StatusCreated, accessKeyToResponse(ak, true))
}

// ListAccessKeys 凭证列表（本人；管理员可 ?username= 查任意服务账号）。
func (h *Handler) ListAccessKeys(c *middleware.Context) {
	username := strings.TrimSpace(string(c.QueryArgs().Peek("username")))
	if username == "" {
		username = c.Username
	}
	if username != c.Username && !c.IsAdmin {
		c.Abort(fasthttp.StatusForbidden, "forbidden")
		return
	}
	list, err := h.svc.ListAccessKeys(context.Background(), username)
	if err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]accessKeyResponse, 0, len(list))
	for _, ak := range list {
		resp = append(resp, accessKeyToResponse(ak, false))
	}
	c.JSON(fasthttp.StatusOK, resp)
}

// DeleteAccessKey 吊销服务凭证（本人，或管理员用 ?username= 指定归属账号）。
func (h *Handler) DeleteAccessKey(c *middleware.Context) {
	keyID, _ := c.UserValue("keyID").(string)
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid access key id")
		return
	}
	username := strings.TrimSpace(string(c.QueryArgs().Peek("username")))
	if username == "" {
		username = c.Username
	}
	if username != c.Username && !c.IsAdmin {
		c.Abort(fasthttp.StatusForbidden, "forbidden")
		return
	}
	if err := h.svc.DeleteAccessKey(context.Background(), username, keyID); err != nil {
		c.Abort(fasthttp.StatusInternalServerError, err.Error())
		return
	}
	c.SetStatusCode(fasthttp.StatusNoContent)
	c.SetBody(nil)
}
