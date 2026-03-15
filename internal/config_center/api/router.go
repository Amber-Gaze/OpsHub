package api

import (
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/transport"
	"github.com/fasthttp/router"
)

// RegisterRoutes 注册配置中心路由，包含 /configs 与 /internal/configs（供 Gateway 转发）。
func RegisterRoutes(r *router.Router, svc *Service, middlewares ...middleware.Middleware) {
	group := transport.NewRouterGroup(r, "", middleware.Recover())
	handler := NewHandler(svc)

	registerConfigGroup := func(prefix string) {
		config := group.Group(prefix)
		config.GET("", handler.List)
		config.GET("/:key", handler.Get)
		config.POST("", handler.Create)
		config.PUT("/:key", handler.Update)
		config.DELETE("/:key", handler.Delete)
	}

	registerConfigGroup("/configs")
	registerConfigGroup("/internal/configs")
}
