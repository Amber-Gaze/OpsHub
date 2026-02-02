package api

import (
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

	handler := NewHandler(svc, HandlerOptions{AuthLoginPath: "/login"})

	group.Use(middleware.RateLimit(cfg.RateLimitRPS))
	group.GET("/healthz", handler.Health)
	group.POST("/login", handler.Login)
	// group.POST("/logout", handler.Logout)
	// group.POST("/refresh", handler.Refresh)

	// group.Use(middleware.JWTAuthMiddleware())
	// users := group.Group(utils.UserPath)
	// {
	// 	users.GET("/", handler.ListUsers)
	// 	users.GET("/:name", handler.GetUser)
	// 	users.PUT(":name/change-passwd", handler.ChangePassword)
	// 	users.PUT("/:name", handler.UpdateUser)
	// 	users.DELETE("/:name", handler.DeleteUser)
	// 	users.DELETE("/", handler.DeleteUsers)
	// }

	// configs := group.Group(utils.ConfigPath)
	// {
	// 	configs.GET("/", handler.ListConfigs)
	// 	configs.GET("/:key", handler.GetConfig)
	// 	configs.POST("/", handler.CreateConfig)
	// 	configs.PUT("/:key", handler.UpdateConfig)
	// 	configs.DELETE("/:key", handler.DeleteConfig)
	// }
}
