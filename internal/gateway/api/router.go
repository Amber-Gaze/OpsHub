package api

import (
	"strings"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/transport"
	"github.com/fasthttp/router"
)

type RoutesConfig struct {
	AuthBaseURL   string
	AuthorizePath string
	LoginPath     string
	RateLimitRPS  int
}

func RegisterRoutes(r *router.Router, svc *Service, cfg RoutesConfig) {
	group := transport.NewRouterGroup(r, "", middleware.Recover())

	loginPath := normalizePath(cfg.LoginPath, "/login")
	handler := NewHandler(svc, HandlerOptions{AuthLoginPath: loginPath})

	group.Use(middleware.RateLimit(cfg.RateLimitRPS))
	group.Use(middleware.JWTAuthMiddleware())

	group.GET("/healthz", handler.Health)

	auth := group.Group("/auth")
	{
		auth.POST("/login", handler.Login)
		// auth.POST("/authorize", handler.Authorize)
	}

	configs := group.Group("/configs")
	{
		configs.GET("/", handler.ListConfigs)
		configs.GET("/:key", handler.GetConfig)
		configs.POST("/", handler.CreateConfig)
		configs.PUT("/:key", handler.UpdateConfig)
		configs.DELETE("/:key", handler.DeleteConfig)
	}
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
