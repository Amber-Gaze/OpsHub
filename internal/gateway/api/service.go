package api

import (
	"fmt"
	"strings"

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
