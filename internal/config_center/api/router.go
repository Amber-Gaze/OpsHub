package api

import (
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/transport"
	"github.com/fasthttp/router"
)

// RegisterRoutes 注册配置中心路由，包含 /configs 与 /internal/configs（供 Gateway 转发）。
func RegisterRoutes(r *router.Router, svc *Service, middlewares ...middleware.Middleware) {
	group := transport.NewRouterGroup(r, "", middleware.RequestID(), middleware.Recover())
	handler := NewHandler(svc)

	group.GET("/healthz", handler.Healthz)
	group.GET("/readyz", handler.Readyz)

	registerConfigGroup := func(prefix string) {
		config := group.Group(prefix)

		// 控制台分层浏览：/configs/tree[/business[/module[/name]]]
		config.GET("/tree", handler.Tree)
		config.GET("/tree/{business}", handler.Business)
		config.GET("/tree/{business}/{module}", handler.Module)
		config.GET("/tree/{business}/{module}/{name}", handler.Item)
		config.PUT("/tree/{business}/{module}/{name}", handler.UpdateItem)
		config.DELETE("/tree/{business}/{module}/{name}", handler.DeleteItem)

		// 扁平 CRUD（兼容单段 key）
		config.GET("", handler.List)
		config.GET("/{key}", handler.Get)
		config.POST("", handler.Create)
		config.PUT("/{key}", handler.Update)
		config.DELETE("/{key}", handler.Delete)
	}

	registerConfigGroup("/configs")
	registerConfigGroup("/internal/configs")
	middleware.AttachRouterErrors(r)
}
