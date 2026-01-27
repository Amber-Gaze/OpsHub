package api

import (
	"strings"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/fasthttp/router"
)

type RoutesConfig struct {
	AuthBaseURL   string
	AuthorizePath string
	LoginPath     string
	RateLimitRPS  int
}

func RegisterRoutes(r *router.Router, svc *Service, cfg RoutesConfig) {
	authorizeURL := buildAuthorizeURL(cfg.AuthBaseURL, cfg.AuthorizePath)
	loginPath := normalizePath(cfg.LoginPath, "/login")

	handler := NewHandler(svc, HandlerOptions{AuthLoginPath: loginPath})

	base := []middleware.Middleware{middleware.Recover()}
	protected := append([]middleware.Middleware{}, base...)

	if cfg.RateLimitRPS > 0 {
		protected = append(protected, middleware.RateLimit(cfg.RateLimitRPS))
	}
	if authorizeURL != "" {
		protected = append(protected, middleware.Auth(middleware.AuthConfig{AuthURL: authorizeURL}))
	}

	r.GET("/healthz", middleware.Chain(handler.Health, base...))
	r.POST("/auth/login", middleware.Chain(handler.Login, base...))
	r.GET("/configs", middleware.Chain(handler.ListConfigs, protected...))
	r.GET("/configs/:key", middleware.Chain(handler.GetConfig, protected...))
	r.POST("/configs", middleware.Chain(handler.CreateConfig, protected...))
	r.PUT("/configs/:key", middleware.Chain(handler.UpdateConfig, protected...))
	r.DELETE("/configs/:key", middleware.Chain(handler.DeleteConfig, protected...))
}

func buildAuthorizeURL(baseURL, path string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return ""
	}
	authorizePath := normalizePath(path, "/authorize")
	return base + authorizePath
}

func normalizePath(p, fallback string) string {
	if strings.TrimSpace(p) == "" {
		return fallback
	}
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}
