package api

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/config_center/domain"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/authutil"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/casbinx"
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

func (h *Handler) Healthz(c *middleware.Context) {
	c.JSON(fasthttp.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Readyz(c *middleware.Context) {
	if err := h.svc.Ready(); err != nil {
		c.Abort(fasthttp.StatusServiceUnavailable, err.Error())
		return
	}
	c.JSON(fasthttp.StatusOK, map[string]string{"status": "ready"})
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
	grants := h.grants(c)
	if len(grants) == 0 {
		c.Abort(fasthttp.StatusNotFound, "permission denied")
		return
	}
	c.JSON(fasthttp.StatusOK, filterReadable(h.svc.List(), grants))
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
	if !h.requireRead(c, keyStr) {
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
	if !h.requireWrite(c, req.Key) {
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
	if !h.requireWrite(c, keyStr) {
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
	if !h.requireDelete(c, keyStr) {
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

// Tree 返回完整的「业务 → 模块 → 具体项」层级树，供控制台左侧导航；按读权限过滤。
func (h *Handler) Tree(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	grants := h.grants(c)
	if len(grants) == 0 {
		c.Abort(fasthttp.StatusNotFound, "permission denied")
		return
	}
	c.JSON(fasthttp.StatusOK, domain.BuildTree(filterReadable(h.svc.List(), grants)))
}

// Business 返回单个业务下的模块分组（业务子树）；对无权限的业务返回 404。
func (h *Handler) Business(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	business := stringParam(c, "business")
	if business == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid business")
		return
	}
	modules, ok := h.svc.GetBusiness(business)
	if !ok {
		c.Abort(fasthttp.StatusNotFound, ErrConfigNotFound.Error())
		return
	}
	// 按读权限过滤，过滤后为空视同无权访问该业务
	modules = filterModules(modules, h.grants(c))
	if len(modules) == 0 {
		c.Abort(fasthttp.StatusNotFound, "permission denied")
		return
	}
	c.JSON(fasthttp.StatusOK, domain.BusinessView{Business: business, Modules: modules})
}

// Module 返回 business/module 下的全部配置项列表；无读权限返回 404。
func (h *Handler) Module(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	business := stringParam(c, "business")
	module := stringParam(c, "module")
	if business == "" || module == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid business or module")
		return
	}
	items, ok := h.svc.GetModule(business, module)
	if !ok {
		c.Abort(fasthttp.StatusNotFound, ErrConfigNotFound.Error())
		return
	}
	items = filterReadable(items, h.grants(c))
	if len(items) == 0 {
		c.Abort(fasthttp.StatusNotFound, "permission denied")
		return
	}
	c.JSON(fasthttp.StatusOK, domain.ModuleView{Business: business, Module: module, Items: items})
}

// Item 返回 business/module/name 对应的具体配置项；无读权限返回 404。
func (h *Handler) Item(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	business := stringParam(c, "business")
	module := stringParam(c, "module")
	name := stringParam(c, "name")
	key := strings.Join([]string{business, module, name}, "/")
	if business == "" || module == "" || name == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid key")
		return
	}
	if !h.requireRead(c, key) {
		return
	}
	cfg, ok := h.svc.GetItem(business, module, name)
	if !ok {
		c.Abort(fasthttp.StatusNotFound, ErrConfigNotFound.Error())
		return
	}
	c.JSON(fasthttp.StatusOK, cfg)
}

// filterModules 按读权限过滤模块列表（模块内无可读项则剔除）。
func filterModules(modules []domain.ModuleNode, grants []casbinx.Grant) []domain.ModuleNode {
	if grants == nil {
		return nil
	}
	out := make([]domain.ModuleNode, 0, len(modules))
	for _, m := range modules {
		items := filterReadable(m.Items, grants)
		if len(items) == 0 {
			continue
		}
		m.Items = items
		out = append(out, m)
	}
	return out
}

// UpdateItem 更新 business/module/name 对应的具体配置项。
func (h *Handler) UpdateItem(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	business := stringParam(c, "business")
	module := stringParam(c, "module")
	name := stringParam(c, "name")
	if business == "" || module == "" || name == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid key")
		return
	}

	var req updateConfigRequest
	if err := json.Unmarshal(c.PostBody(), &req); err != nil {
		c.Abort(fasthttp.StatusBadRequest, "invalid request body")
		return
	}

	key := strings.Join([]string{business, module, name}, "/")
	if !h.requireWrite(c, key) {
		return
	}
	cfg, err := h.svc.Update(key, req.Value, operatorFromContext(c, req.Operator))
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

// DeleteItem 删除 business/module/name 对应的具体配置项；无删除权限返回 403。
func (h *Handler) DeleteItem(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	business := stringParam(c, "business")
	module := stringParam(c, "module")
	name := stringParam(c, "name")
	if business == "" || module == "" || name == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid key")
		return
	}

	key := strings.Join([]string{business, module, name}, "/")
	if !h.requireDelete(c, key) {
		return
	}
	if err := h.svc.Delete(key); err != nil {
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

// History 返回某个配置 key 的历史变更记录 + 当前值，便于对比排障。
// 无读权限返回 404；配置已删除时 current 为 null。
func (h *Handler) History(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	raw, _ := c.UserValue("path").(string)
	key := strings.Trim(strings.TrimSpace(raw), "/")
	if key == "" {
		c.Abort(fasthttp.StatusBadRequest, "invalid key")
		return
	}
	if !h.requireRead(c, key) {
		return
	}

	current, exists := h.svc.Get(key)
	var currentPtr *domain.ConfigItem
	if exists {
		cp := current
		currentPtr = &cp
	}
	c.JSON(fasthttp.StatusOK, domain.ConfigHistoryResponse{
		Key:     key,
		Current: currentPtr,
		History: h.svc.History(key),
	})
}

// Pull 供下游「专门的服务」拉取配置（机器消费友好），支持不同层级与增量拉取：
//
//	过滤（可组合，精确到任意层级）：
//	  business=pay                          → 业务 pay/**
//	  business=pay&module=gateway           → pay 的 gateway 模块
//	  business=pay&module=gateway&name=xxx  → pay/gateway/xxx 精确一项
//	  path=pay/gateway                      → key 为 pay/gateway 或以 pay/gateway/ 开头
//	  key=pay/gateway/timeout_ms            → 精确一个 key
//
//	增量判断更新：
//	  since=<rev>                           → 只返回全局版本号 > since 的变更项 + 被删 key(removed)，
//	                                           响应携带最新 revision；下游据此增量更新即可，无需全量拉取。
//
//	所有结果按当前用户 scope 过滤可读项；无读权限返回 404。
func (h *Handler) Pull(c *middleware.Context) {
	if !h.requireAuthDecision(c) {
		return
	}
	grants := h.grants(c)
	if len(grants) == 0 {
		c.Abort(fasthttp.StatusNotFound, "permission denied")
		return
	}

	q := c.QueryArgs()
	business := strings.TrimSpace(string(q.Peek("business")))
	module := strings.TrimSpace(string(q.Peek("module")))
	name := strings.TrimSpace(string(q.Peek("name")))
	path := strings.TrimSpace(string(q.Peek("path")))
	key := strings.TrimSpace(string(q.Peek("key")))
	sinceStr := strings.TrimSpace(string(q.Peek("since")))

	if name != "" && (business == "" || module == "") {
		c.Abort(fasthttp.StatusBadRequest, "name requires business and module")
		return
	}
	var since int64
	if sinceStr != "" {
		since, _ = strconv.ParseInt(sinceStr, 10, 64)
	}

	match := pullKeyMatcher(business, module, name, path, key)
	readable := func(k string) bool { return casbinx.CanRead(grants, k) }

	items := make([]domain.ConfigItem, 0)
	removed := make([]string, 0)

	if since > 0 {
		// 增量：自 since 以来的变更项 + 被删 key
		for k, ch := range h.svc.ChangesSince(since) {
			if !match(k) || !readable(k) {
				continue
			}
			if ch.Action == "delete" {
				removed = append(removed, k)
				continue
			}
			if it, ok := h.svc.Get(k); ok {
				items = append(items, it)
			}
		}
		sort.Strings(removed)
	} else {
		// 全量：当前所有可读项（按过滤条件）
		for _, it := range filterReadable(h.svc.List(), grants) {
			if match(it.Key) {
				items = append(items, it)
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })

	c.JSON(fasthttp.StatusOK, domain.PullResponse{
		Revision:    h.svc.Revision(c),
		Items:       items,
		Removed:     removed,
		GeneratedAt: time.Now().UTC(),
	})
}

// pullKeyMatcher 依据过滤参数构造 key 匹配谓词。
func pullKeyMatcher(business, module, name, path, key string) func(string) bool {
	switch {
	case key != "":
		k := strings.Trim(strings.TrimSpace(key), "/")
		return func(s string) bool { return s == k }
	case path != "":
		p := strings.Trim(strings.TrimSpace(path), "/")
		return func(s string) bool { return s == p || strings.HasPrefix(s, p+"/") }
	default:
		return func(s string) bool {
			b, m := domain.SplitKey(s)
			if business != "" && b != business {
				return false
			}
			if module != "" && m != module {
				return false
			}
			if name != "" && domain.ItemName(s) != name {
				return false
			}
			return true
		}
	}
}

func stringParam(c *middleware.Context, name string) string {
	raw, _ := c.UserValue(name).(string)
	return strings.TrimSpace(raw)
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
)

// requireAuthDecision 校验 IAM 签发的 scope 决策：
//  1. 验签（HMAC）确保决策来自 IAM；
//  2. 解析 scope 载体（scope|<subject>|<grantsJSON>|<ts>）得到该用户对配置的授权列表；
//  3. 将 subject 与 grants 写入 Context，供各处理器按权限过滤/校验。
//
// 读无权限 → 404（不泄露配置是否存在）；写/删无权限 → 403。
func (h *Handler) requireAuthDecision(c *middleware.Context) bool {
	decision := strings.TrimSpace(string(c.Request.Header.Peek(headerDecision)))
	signature := strings.TrimSpace(string(c.Request.Header.Peek(headerSignature)))
	subject := strings.TrimSpace(string(c.Request.Header.Peek(headerSubject)))

	if decision == "" || signature == "" || subject == "" {
		c.Abort(fasthttp.StatusForbidden, "missing authorization context")
		return false
	}

	if !authutil.Verify(decision, signature, h.decisionSecret) {
		c.Abort(fasthttp.StatusForbidden, "invalid authorization signature")
		return false
	}

	scopeSubject, grants, err := casbinx.ParseScopePayload(decision)
	if err != nil {
		c.Abort(fasthttp.StatusForbidden, "invalid authorization scope")
		return false
	}
	if scopeSubject != subject {
		c.Abort(fasthttp.StatusForbidden, "authorization subject mismatch")
		return false
	}

	c.UserID = subject
	c.Username = subject
	c.SetAuthDecision(&middleware.AuthDecision{
		Allow:     true,
		Subject:   subject,
		Scope:     grants,
		Decision:  decision,
		Signature: signature,
	})
	return true
}

// grants 返回当前请求携带的配置授权列表（scope）。
func (h *Handler) grants(c *middleware.Context) []casbinx.Grant {
	if c.Decision == nil {
		return nil
	}
	return c.Decision.Scope
}

// requireRead 校验读权限；无权限时返回 404（不泄露配置是否存在）。
func (h *Handler) requireRead(c *middleware.Context, key string) bool {
	if casbinx.CanRead(h.grants(c), key) {
		return true
	}
	c.Abort(fasthttp.StatusNotFound, "permission denied")
	return false
}

// requireWrite 校验写权限；无权限时返回 403。
func (h *Handler) requireWrite(c *middleware.Context, key string) bool {
	if casbinx.CanWrite(h.grants(c), key) {
		return true
	}
	c.Abort(fasthttp.StatusForbidden, "permission denied: write")
	return false
}

// requireDelete 校验删除权限；无权限时返回 403。
func (h *Handler) requireDelete(c *middleware.Context, key string) bool {
	if casbinx.CanDelete(h.grants(c), key) {
		return true
	}
	c.Abort(fasthttp.StatusForbidden, "permission denied: delete")
	return false
}

// filterReadable 按读权限过滤配置项列表。
func filterReadable(items []domain.ConfigItem, grants []casbinx.Grant) []domain.ConfigItem {
	if grants == nil {
		return nil
	}
	out := make([]domain.ConfigItem, 0, len(items))
	for _, it := range items {
		if casbinx.CanRead(grants, it.Key) {
			out = append(out, it)
		}
	}
	return out
}
