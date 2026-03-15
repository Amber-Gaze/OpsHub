package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/valyala/fasthttp"
)

type Service struct {
	authBaseURL         string
	configCenterBaseURL string
	client              *fasthttp.Client
}

func NewService(authBaseURL, configCenterURL string) *Service {
	return NewServiceWithClient(authBaseURL, configCenterURL, nil)
}

func NewConfigService(configCenterURL string) *Service {
	return NewService("", configCenterURL)
}

func NewServiceWithClient(authBaseURL, configCenterURL string, client *fasthttp.Client) *Service {
	if client == nil {
		client = &fasthttp.Client{}
	}
	return &Service{
		authBaseURL:         strings.TrimRight(authBaseURL, "/"),
		configCenterBaseURL: strings.TrimRight(configCenterURL, "/"),
		client:              client,
	}
}

func (s *Service) ForwardAuth(method, path string, body []byte, headers map[string]string) (int, []byte, string, error) {
	if strings.TrimSpace(s.authBaseURL) == "" {
		return 0, nil, "", fmt.Errorf("auth service url is not configured")
	}
	return s.doRequest(s.authBaseURL, method, path, body, headers)
}

func (s *Service) ForwardConfig(method, path string, body []byte, headers map[string]string) (int, []byte, string, error) {
	if strings.TrimSpace(s.configCenterBaseURL) == "" {
		return 0, nil, "", fmt.Errorf("config center url is not configured")
	}
	return s.doRequest(s.configCenterBaseURL, method, path, body, headers)
}

type authorizeReq struct {
	Token    string `json:"token"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type authorizeResp struct {
	Allow     bool   `json:"allow"`
	Subject   string `json:"subject"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Decision  string `json:"decision"`
	Signature string `json:"signature"`
}

// Authorize 调用 IAM /authorize，返回鉴权决策，供 Gateway 转发配置请求时携带 X-Auth-* 头。
func (s *Service) Authorize(token, resource, action string) (*middleware.AuthDecision, error) {
	if strings.TrimSpace(s.authBaseURL) == "" {
		return nil, fmt.Errorf("auth service url is not configured")
	}
	body, _ := json.Marshal(authorizeReq{Token: token, Resource: resource, Action: action})
	status, respBody, _, err := s.doRequest(s.authBaseURL, fasthttp.MethodPost, "/authorize", body, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return nil, err
	}
	if status != fasthttp.StatusOK {
		return nil, fmt.Errorf("authorize returned status %d", status)
	}
	var resp authorizeResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	return &middleware.AuthDecision{
		Allow:     resp.Allow,
		Subject:   resp.Subject,
		Action:    resp.Action,
		Resource:  resp.Resource,
		Decision:  resp.Decision,
		Signature: resp.Signature,
	}, nil
}

func (s *Service) doRequest(baseURL, method, path string, body []byte, headers map[string]string) (int, []byte, string, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	uri := strings.TrimRight(baseURL, "/") + path

	var req fasthttp.Request
	var resp fasthttp.Response

	req.SetRequestURI(uri)
	req.Header.SetMethod(method)
	for k, v := range headers {
		if strings.TrimSpace(v) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	if len(body) > 0 {
		req.SetBody(body)
	}

	if err := s.client.Do(&req, &resp); err != nil {
		return 0, nil, "", err
	}

	status := resp.StatusCode()
	contentType := string(resp.Header.Peek("Content-Type"))
	respBody := append([]byte(nil), resp.Body()...)

	return status, respBody, contentType, nil
}
