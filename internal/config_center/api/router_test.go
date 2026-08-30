package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Amber-Gaze/OpsHub/internal/config_center/domain"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/authutil"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/casbinx"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// dispatch 直接通过路由器分发请求，返回处理后上下文。
func dispatch(method, path string, headers map[string]string, body string) *fasthttp.RequestCtx {
	var ctx fasthttp.RequestCtx
	var req fasthttp.Request
	req.Header.SetMethod(method)
	req.SetRequestURI(path)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != "" {
		req.Header.SetContentType("application/json")
		req.SetBodyString(body)
	}
	ctx.Init(&req, nil, nil)
	return &ctx
}

// authHeaders 构造携带指定授权的请求头（scope 载体 + 有效签名）。
func authHeaders(grants ...casbinx.Grant) map[string]string {
	payload, _ := casbinx.BuildScopePayload("tester", grants)
	return map[string]string{
		"X-Auth-Decision":  payload,
		"X-Auth-Signature": authutil.Sign(payload, authutil.DefaultDecisionSecret),
		"X-Auth-Subject":   "tester",
	}
}

// adminHeaders 管理员全量授权。
func adminHeaders() map[string]string {
	return authHeaders(casbinx.AdminGrant...)
}

func testRouter(t *testing.T) *router.Router {
	t.Helper()
	svc := NewService()
	for _, k := range []string{"pay/gateway/timeout_ms", "pay/gateway/retry", "cdn/vod/bitrate"} {
		if _, err := svc.Create(k, "v", "tester"); err != nil {
			t.Fatalf("create %s: %v", k, err)
		}
	}
	r := router.New()
	RegisterRoutes(r, svc)
	return r
}

func TestConfigTreeRoutes(t *testing.T) {
	r := testRouter(t)

	// 完整树
	ctx := dispatch("GET", "/configs/tree", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /configs/tree status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}

	// 业务子树
	ctx = dispatch("GET", "/configs/tree/pay", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /configs/tree/pay status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}

	// 模块列表
	ctx = dispatch("GET", "/configs/tree/pay/gateway", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /configs/tree/pay/gateway status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}

	// 具体项
	ctx = dispatch("GET", "/configs/tree/pay/gateway/timeout_ms", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /configs/tree/pay/gateway/timeout_ms status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestConfigTreeRouteMissing(t *testing.T) {
	r := testRouter(t)
	ctx := dispatch("GET", "/configs/tree/missing/module", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("GET /configs/tree/missing/module status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestConfigTreeUpdateAndDelete(t *testing.T) {
	r := testRouter(t)

	ctx := dispatch("PUT", "/configs/tree/pay/gateway/timeout_ms", adminHeaders(), `{"value":"200","operator":"tester"}`)
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("PUT item status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}

	// 更新后读取应为新值
	ctx = dispatch("GET", "/configs/tree/pay/gateway/timeout_ms", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET item after update status = %d", ctx.Response.StatusCode())
	}

	// 删除
	ctx = dispatch("DELETE", "/configs/tree/cdn/vod/bitrate", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("DELETE item status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// TestConfigFlatKeyRoute 回归：{key} 语法必须真实匹配单段 key 请求。
func TestConfigFlatKeyRoute(t *testing.T) {
	r := testRouter(t)

	// 创建单段配置
	ctx := dispatch("POST", "/configs", adminHeaders(), `{"key":"debug","value":"false","operator":"tester"}`)
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("POST /configs status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}

	// 读取单段 key（此前 :key 语法下会 404）
	ctx = dispatch("GET", "/configs/debug", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /configs/debug status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}

	// 不存在的 key 应 404
	ctx = dispatch("GET", "/configs/nonexistent", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("GET /configs/nonexistent status = %d, want 404", ctx.Response.StatusCode())
	}
}

// TestConfigScopeFiltering 只有 pay 业务读权限的用户：树应被过滤，越权访问返回 404。
func TestConfigScopeFiltering(t *testing.T) {
	r := testRouter(t)
	hdr := authHeaders(casbinx.Grant{Obj: "config/pay/**", Act: "read"})

	// 树仅包含 pay，不包含 cdn
	ctx := dispatch("GET", "/configs/tree", hdr, "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /configs/tree status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, `"business":"pay"`) {
		t.Fatalf("tree should contain pay, body=%s", body)
	}
	if strings.Contains(body, `"business":"cdn"`) {
		t.Fatalf("tree should NOT contain cdn, body=%s", body)
	}

	// 越权访问 cdn 业务 → 404
	ctx = dispatch("GET", "/configs/tree/cdn", hdr, "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("GET /configs/tree/cdn status = %d, want 404", ctx.Response.StatusCode())
	}

	// 越权读取具体项 → 404
	ctx = dispatch("GET", "/configs/tree/cdn/vod/bitrate", hdr, "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("GET /configs/tree/cdn/vod/bitrate status = %d, want 404", ctx.Response.StatusCode())
	}

	// 有权限的具体项 → 200
	ctx = dispatch("GET", "/configs/tree/pay/gateway/timeout_ms", hdr, "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /configs/tree/pay/gateway/timeout_ms status = %d, want 200", ctx.Response.StatusCode())
	}
}

// TestConfigScopeWriteDenied 有读权限但无写权限：写操作返回 403。
func TestConfigScopeWriteDenied(t *testing.T) {
	r := testRouter(t)
	hdr := authHeaders(casbinx.Grant{Obj: "config/pay/**", Act: "read"})

	ctx := dispatch("PUT", "/configs/tree/pay/gateway/timeout_ms", hdr, `{"value":"x"}`)
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("PUT status = %d, want 403", ctx.Response.StatusCode())
	}

	ctx = dispatch("DELETE", "/configs/tree/pay/gateway/retry", hdr, "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("DELETE status = %d, want 403", ctx.Response.StatusCode())
	}

	ctx = dispatch("POST", "/configs", hdr, `{"key":"pay/gateway/new","value":"1"}`)
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("POST status = %d, want 403", ctx.Response.StatusCode())
	}
}

// TestConfigScopeWriteAllowed 有写权限时写操作成功；越权写返回 403。
func TestConfigScopeWriteAllowed(t *testing.T) {
	r := testRouter(t)
	hdr := authHeaders(casbinx.Grant{Obj: "config/pay/gateway/**", Act: "write"})

	ctx := dispatch("PUT", "/configs/tree/pay/gateway/timeout_ms", hdr, `{"value":"200","operator":"tester"}`)
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("PUT pay/gateway status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}

	ctx = dispatch("PUT", "/configs/tree/cdn/vod/bitrate", hdr, `{"value":"x"}`)
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("PUT cdn status = %d, want 403", ctx.Response.StatusCode())
	}
}

// TestConfigNoGrants 无任何配置授权：集合与单项读取均返回 404。
func TestConfigNoGrants(t *testing.T) {
	r := testRouter(t)
	hdr := authHeaders()

	ctx := dispatch("GET", "/configs/tree", hdr, "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("GET /configs/tree status = %d, want 404", ctx.Response.StatusCode())
	}

	ctx = dispatch("GET", "/configs/pay", hdr, "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("GET /configs/pay status = %d, want 404", ctx.Response.StatusCode())
	}
}

// TestConfigHistoryRoute 历史对比接口：返回 current + history。
func TestConfigHistoryRoute(t *testing.T) {
	r := testRouter(t)

	ctx := dispatch("GET", "/configs/history/pay/gateway/timeout_ms", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /configs/history status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, `"current"`) || !strings.Contains(body, `"history"`) {
		t.Fatalf("history response missing fields, body=%s", body)
	}

	// 越权查看历史 → 404
	ctx = dispatch("GET", "/configs/history/cdn/vod/bitrate", authHeaders(casbinx.Grant{Obj: "config/pay/**", Act: "read"}), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("GET /configs/history (forbidden) status = %d, want 404", ctx.Response.StatusCode())
	}
}

// TestConfigPullRoute 下游拉取接口：可按 business 过滤。
func TestConfigPullRoute(t *testing.T) {
	r := testRouter(t)

	ctx := dispatch("GET", "/configs/pull", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /configs/pull status = %d, body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if !strings.Contains(string(ctx.Response.Body()), `"pay/gateway/timeout_ms"`) {
		t.Fatalf("pull should include pay item, body=%s", ctx.Response.Body())
	}

	// 按 business 过滤：只返回 pay
	ctx = dispatch("GET", "/configs/pull?business=pay", adminHeaders(), "")
	r.Handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /configs/pull?business=pay status = %d", ctx.Response.StatusCode())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, `"pay/gateway/timeout_ms"`) {
		t.Fatalf("filtered pull should include pay item, body=%s", body)
	}
	if strings.Contains(body, `"cdn/vod/bitrate"`) {
		t.Fatalf("filtered pull should NOT include cdn item, body=%s", body)
	}
}

// pullResp 解析 /configs/pull 响应。
func pullResp(t *testing.T, c *fasthttp.RequestCtx) domain.PullResponse {
	t.Helper()
	var pr domain.PullResponse
	if err := json.Unmarshal(c.Response.Body(), &pr); err != nil {
		t.Fatalf("unmarshal pull response: %v, body=%s", err, c.Response.Body())
	}
	return pr
}

// TestConfigPullFilters 不同层级过滤：module / name / path / key。
func TestConfigPullFilters(t *testing.T) {
	r := testRouter(t) // pay/gateway/timeout_ms, pay/gateway/retry, cdn/vod/bitrate

	// 全量：3 项 + revision=3
	pr := pullResp(t, pullWith(r, "/configs/pull"))
	if len(pr.Items) != 3 || pr.Revision != 3 {
		t.Fatalf("full pull items=%d revision=%d, want 3/3", len(pr.Items), pr.Revision)
	}

	// module 过滤
	pr = pullResp(t, pullWith(r, "/configs/pull?business=pay&module=gateway"))
	if len(pr.Items) != 2 {
		t.Fatalf("module pull items=%d, want 2", len(pr.Items))
	}

	// name 精确项
	pr = pullResp(t, pullWith(r, "/configs/pull?business=pay&module=gateway&name=retry"))
	if len(pr.Items) != 1 || pr.Items[0].Key != "pay/gateway/retry" {
		t.Fatalf("name pull = %+v", pr.Items)
	}

	// path 前缀
	pr = pullResp(t, pullWith(r, "/configs/pull?path=pay/gateway"))
	if len(pr.Items) != 2 {
		t.Fatalf("path pull items=%d, want 2", len(pr.Items))
	}

	// key 精确
	pr = pullResp(t, pullWith(r, "/configs/pull?key=cdn/vod/bitrate"))
	if len(pr.Items) != 1 || pr.Items[0].Key != "cdn/vod/bitrate" {
		t.Fatalf("key pull = %+v", pr.Items)
	}

	// name 缺少 business/module → 400
	c := dispatch("GET", "/configs/pull?name=retry", adminHeaders(), "")
	r.Handler(c)
	if c.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("name-without-business status = %d, want 400", c.Response.StatusCode())
	}
}

// TestConfigPullIncremental 增量拉取：since 返回变更项与 removed，revision 单调递增。
func TestConfigPullIncremental(t *testing.T) {
	r := testRouter(t) // rev=3: pay/gateway/timeout_ms, pay/gateway/retry, cdn/vod/bitrate

	// since=2 → 只有 rev3 的 create（cdn/vod/bitrate）
	pr := pullResp(t, pullWith(r, "/configs/pull?since=2"))
	if len(pr.Items) != 1 || pr.Items[0].Key != "cdn/vod/bitrate" {
		t.Fatalf("since=2 items = %+v, want [cdn/vod/bitrate]", pr.Items)
	}
	if len(pr.Removed) != 0 || pr.Revision != 3 {
		t.Fatalf("since=2 removed=%v revision=%d", pr.Removed, pr.Revision)
	}

	// 更新 + 删除
	c := dispatch("PUT", "/configs/tree/pay/gateway/timeout_ms", adminHeaders(), `{"value":"200","operator":"tester"}`)
	r.Handler(c)
	if c.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("PUT status=%d body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	c = dispatch("DELETE", "/configs/tree/pay/gateway/retry", adminHeaders(), "")
	r.Handler(c)
	if c.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("DELETE status=%d", c.Response.StatusCode())
	}

	// since=3 → 变更项=timeout_ms(updated rev4)，removed=retry(rev5)
	pr = pullResp(t, pullWith(r, "/configs/pull?since=3"))
	if pr.Revision != 5 {
		t.Fatalf("revision after update+delete = %d, want 5", pr.Revision)
	}
	if len(pr.Items) != 1 || pr.Items[0].Key != "pay/gateway/timeout_ms" || pr.Items[0].Value != "200" {
		t.Fatalf("incremental items = %+v, want [timeout_ms=200]", pr.Items)
	}
	if len(pr.Removed) != 1 || pr.Removed[0] != "pay/gateway/retry" {
		t.Fatalf("incremental removed = %v, want [pay/gateway/retry]", pr.Removed)
	}

	// 增量 + 层级过滤组合：since=3 且 business=cdn → 空
	pr = pullResp(t, pullWith(r, "/configs/pull?since=3&business=cdn"))
	if len(pr.Items) != 0 || len(pr.Removed) != 0 {
		t.Fatalf("cdn since=3 should be empty, items=%v removed=%v", pr.Items, pr.Removed)
	}
}

// pullWith 发起一次带 admin 授权的 pull 请求。
func pullWith(r *router.Router, path string) *fasthttp.RequestCtx {
	c := dispatch("GET", path, adminHeaders(), "")
	r.Handler(c)
	return c
}
