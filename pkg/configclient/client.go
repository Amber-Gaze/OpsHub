// Package configclient 提供「专门的下游服务」从配置中心拉取配置的客户端。
//
// 典型用法（示例见 examples/config-consumer）：
//
//	cli := configclient.New("http://127.0.0.1:8004", "http://127.0.0.1:8007")
//	if err := cli.Login(ctx, "svc-user", "pass"); err != nil { ... }
//	items, err := cli.Pull(ctx) // 拉取全部可读配置快照
//	kv := configclient.Snapshot(items) // key -> value
//
// 客户端封装了「登录 IAM → authorize 取 scope → 携带 X-Auth-* 头调用 /configs/pull」的完整链路，
// 下游服务只需关心拉到的配置内容。
package configclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/jwt"
	"github.com/valyala/fasthttp"
)

// Item 配置项，与配置中心 /configs/pull 的 wire 契约一致。
type Item struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// PullResponse 拉取接口响应。
// Revision 为拉取时的全局配置版本号；增量拉取（since）时 Removed 为区间内被删除的 key。
type PullResponse struct {
	Revision    int64     `json:"revision"`
	Items       []Item    `json:"items"`
	Removed     []string  `json:"removed,omitempty"`
	GeneratedAt time.Time `json:"generated_at"`
}

// PullResult 拉取结果（含增量信息），供下游做增量更新与版本比较。
type PullResult struct {
	Revision int64    // 本次拉取时的全局配置版本号
	Items    []Item   // 配置项（增量模式为变更项）
	Removed  []string // 增量模式下区间内被删除的 key
}

// HasChanged 判断自 since 之后配置是否发生变化（增量拉取前后对比用）。
func (r *PullResult) HasChanged(since int64) bool {
	return r.Revision > since
}

type loginResponse struct {
	Token string `json:"token"`
}

type authorizeResponse struct {
	Decision  string `json:"decision"`
	Signature string `json:"signature"`
	Subject   string `json:"subject"`
}

// Client 配置拉取客户端。
type Client struct {
	authURL   string // IAM 服务根地址，如 http://127.0.0.1:8004
	configURL string // 配置中心根地址，如 http://127.0.0.1:8007
	http      *fasthttp.Client
	token     string
	decision  *authorizeResponse
}

// New 创建客户端。也可将 configURL 指向网关（如 http://127.0.0.1:8001）走统一入口。
func New(authBaseURL, configBaseURL string) *Client {
	return &Client{
		authURL:   strings.TrimRight(authBaseURL, "/"),
		configURL: strings.TrimRight(configBaseURL, "/"),
		http:      &fasthttp.Client{},
	}
}

// Login 在 IAM 登录并保存令牌。
func (c *Client) Login(ctx context.Context, user, password string) error {
	body, _ := json.Marshal(map[string]string{"user": user, "password": password})
	resp, err := c.do(ctx, c.authURL, fasthttp.MethodPost, "/login", body, nil)
	if err != nil {
		return err
	}
	if resp.code != fasthttp.StatusOK {
		return fmt.Errorf("login: status %d %s", resp.code, resp.body)
	}
	var lr loginResponse
	if err := json.Unmarshal(resp.body, &lr); err != nil {
		return err
	}
	c.token = strings.TrimSpace(lr.Token)
	c.decision = nil // 令牌变化后需重新授权
	return nil
}

// LoginWithAccessKey 使用服务凭证（AccessKeyID + AccessKeySecret）认证，替代账号密码登录。
// 用 Secret 自签短期 JWT（header kid=AccessKeyID），iam 按 kid 验签并识别服务账号身份；
// 无需在服务配置里存明文密码，凭证可独立创建/轮换/吊销。
func (c *Client) LoginWithAccessKey(ctx context.Context, accessKeyID, accessKeySecret string) error {
	accessKeyID = strings.TrimSpace(accessKeyID)
	accessKeySecret = strings.TrimSpace(accessKeySecret)
	if accessKeyID == "" || accessKeySecret == "" {
		return fmt.Errorf("access key id and secret required")
	}
	token, err := jwt.GenAccessToken(accessKeyID, "", []byte(accessKeySecret), 10*time.Minute)
	if err != nil {
		return err
	}
	c.token = token
	c.decision = nil // 令牌变化后需重新授权
	return nil
}

// authorize 调用 IAM /authorize 获取 scope 决策（含签名），供后续拉取携带。
func (c *Client) authorize(ctx context.Context, resource, action string) error {
	if c.token == "" {
		return fmt.Errorf("not logged in")
	}
	body, _ := json.Marshal(map[string]string{
		"token":    c.token,
		"resource": resource,
		"action":   action,
	})
	resp, err := c.do(ctx, c.authURL, fasthttp.MethodPost, "/authorize", body, nil)
	if err != nil {
		return err
	}
	if resp.code != fasthttp.StatusOK {
		return fmt.Errorf("authorize: status %d %s", resp.code, resp.body)
	}
	var ar authorizeResponse
	if err := json.Unmarshal(resp.body, &ar); err != nil {
		return err
	}
	c.decision = &ar
	return nil
}

// Pull 拉取全部可读配置快照（首次调用会自动做一次 authorize）。
func (c *Client) Pull(ctx context.Context) ([]Item, error) {
	res, err := c.pullQuery(ctx, "")
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// PullByBusiness 按业务拉取配置快照（business=pay → pay/**）。
func (c *Client) PullByBusiness(ctx context.Context, business string) ([]Item, error) {
	b := strings.TrimSpace(business)
	if b == "" {
		return c.Pull(ctx)
	}
	res, err := c.pullQuery(ctx, "business="+url.QueryEscape(b))
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// PullPath 按层级路径拉取（path=pay/gateway → 该前缀下所有项；path=pay/gateway/x → 精确一项）。
func (c *Client) PullPath(ctx context.Context, path string) ([]Item, error) {
	res, err := c.pullQuery(ctx, "path="+url.QueryEscape(strings.Trim(strings.TrimSpace(path), "/")))
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// PullKey 精确拉取某个配置项（key=pay/gateway/timeout_ms）。
func (c *Client) PullKey(ctx context.Context, key string) ([]Item, error) {
	res, err := c.pullQuery(ctx, "key="+url.QueryEscape(strings.Trim(strings.TrimSpace(key), "/")))
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// PullSince 增量拉取：返回自全局版本号 since 以来的变更项 + 被删 key + 最新版本号。
// 下游只需记录上次收到的 revision，下次用 PullSince 判断 HasChanged 并增量应用。
func (c *Client) PullSince(ctx context.Context, since int64) (*PullResult, error) {
	return c.pullQuery(ctx, "since="+strconv.FormatInt(since, 10))
}

// pullQuery 通用拉取：附带可读 scope 决策头，可带查询参数。
func (c *Client) pullQuery(ctx context.Context, query string) (*PullResult, error) {
	if c.decision == nil {
		if err := c.authorize(ctx, "/configs/pull", "GET"); err != nil {
			return nil, err
		}
	}
	path := "/configs/pull"
	if query != "" {
		path += "?" + query
	}
	resp, err := c.do(ctx, c.configURL, fasthttp.MethodGet, path, nil, c.authHeaders())
	if err != nil {
		return nil, err
	}
	if resp.code != fasthttp.StatusOK {
		return nil, fmt.Errorf("pull: status %d %s", resp.code, resp.body)
	}
	var pr PullResponse
	if err := json.Unmarshal(resp.body, &pr); err != nil {
		return nil, err
	}
	return &PullResult{
		Revision: pr.Revision,
		Items:    pr.Items,
		Removed:  pr.Removed,
	}, nil
}

func (c *Client) authHeaders() map[string]string {
	if c.decision == nil {
		return nil
	}
	return map[string]string{
		"X-Auth-Decision":  c.decision.Decision,
		"X-Auth-Signature": c.decision.Signature,
		"X-Auth-Subject":   c.decision.Subject,
	}
}

// Snapshot 将配置列表转为 key→value 映射，便于业务代码直接取用。
func Snapshot(items []Item) map[string]string {
	out := make(map[string]string, len(items))
	for _, it := range items {
		out[it.Key] = it.Value
	}
	return out
}

// PullSubscribed 按订阅的模块列表拉取配置（每个模块一次 PullPath；未授权模块返回空列表）。
// modules 形如 []string{"pay/gateway", "common/ratelimit"}。
func (c *Client) PullSubscribed(ctx context.Context, modules []string) (map[string][]Item, error) {
	out := make(map[string][]Item, len(modules))
	for _, m := range modules {
		m = strings.Trim(strings.TrimSpace(m), "/")
		if m == "" {
			continue
		}
		items, err := c.PullPath(ctx, m)
		if err != nil {
			return nil, err
		}
		out[m] = items
	}
	return out, nil
}

// WriteTo 将配置项按业务/模块分组落盘为 JSON 文件：<dir>/<business>/<module>.json。
// 文件内容为该模块下 key→value 的 JSON 对象；value 是配置的 JSON 字符串，原样写入。
func WriteTo(items []Item, dir string) error {
	grouped := map[string][]Item{}
	for _, it := range items {
		g := moduleGroup(it.Key)
		grouped[g] = append(grouped[g], it)
	}
	for g, list := range grouped {
		kv := make(map[string]string, len(list))
		for _, it := range list {
			kv[it.Key] = it.Value
		}
		b, err := json.MarshalIndent(kv, "", "  ")
		if err != nil {
			return err
		}
		p := filepath.Join(dir, g+".json")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// moduleGroup 由配置 key 推导落盘分组（business/module）。
func moduleGroup(key string) string {
	segs := strings.Split(strings.Trim(key, "/"), "/")
	if len(segs) >= 2 {
		return segs[0] + "/" + segs[1]
	}
	return key
}

// Diff 对比两次拉取的快照，返回新增/更新/删除的 key 列表（按 key 排序，便于展示）。
func Diff(prev, cur map[string]Item) (added, updated, removed []string) {
	for k, v := range cur {
		old, ok := prev[k]
		if !ok {
			added = append(added, k)
		} else if old.Version != v.Version || old.Value != v.Value {
			updated = append(updated, k)
		}
	}
	for k := range prev {
		if _, ok := cur[k]; !ok {
			removed = append(removed, k)
		}
	}
	sort.Strings(added)
	sort.Strings(updated)
	sort.Strings(removed)
	return added, updated, removed
}

func index(items []Item) map[string]Item {
	out := make(map[string]Item, len(items))
	for _, it := range items {
		out[it.Key] = it
	}
	return out
}

type httpResult struct {
	code int
	body []byte
}

func (c *Client) do(ctx context.Context, baseURL, method, path string, body []byte, headers map[string]string) (*httpResult, error) {
	var req fasthttp.Request
	var resp fasthttp.Response
	req.SetRequestURI(baseURL + path)
	req.Header.SetMethod(method)
	if len(body) > 0 {
		req.SetBody(body)
		req.Header.SetContentType("application/json")
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	if err := c.http.DoTimeout(&req, &resp, 5*time.Second); err != nil {
		return nil, err
	}
	rb := append([]byte(nil), resp.Body()...)
	return &httpResult{code: resp.StatusCode(), body: rb}, nil
}
