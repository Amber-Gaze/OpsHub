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
	"sort"
	"strings"
	"time"

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
type PullResponse struct {
	Items       []Item    `json:"items"`
	GeneratedAt time.Time `json:"generated_at"`
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
	if c.decision == nil {
		if err := c.authorize(ctx, "/configs/pull", "GET"); err != nil {
			return nil, err
		}
	}
	resp, err := c.do(ctx, c.configURL, fasthttp.MethodGet, "/configs/pull", nil, c.authHeaders())
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
	return pr.Items, nil
}

// PullByBusiness 按业务拉取配置快照。
func (c *Client) PullByBusiness(ctx context.Context, business string) ([]Item, error) {
	if strings.TrimSpace(business) == "" {
		return c.Pull(ctx)
	}
	if c.decision == nil {
		if err := c.authorize(ctx, "/configs/pull", "GET"); err != nil {
			return nil, err
		}
	}
	resp, err := c.do(ctx, c.configURL, fasthttp.MethodGet,
		"/configs/pull?business="+strings.TrimSpace(business), nil, c.authHeaders())
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
	return pr.Items, nil
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
