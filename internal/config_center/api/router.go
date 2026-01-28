package api

import (
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/transport"
	"github.com/fasthttp/router"
)

func RegisterRoutes(r *router.Router, svc *Service, middlewares ...middleware.Middleware) {
	group := transport.NewRouterGroup(r, "", middleware.Recover())
	handler := NewHandler(svc)

	config := group.Group("/configs")
	{
		config.GET("", handler.List)
		config.GET("/:key", handler.Get)
		config.POST("", handler.Create)
		config.PUT("/:key", handler.Update)
		config.DELETE("/:key", handler.Delete)
	}
}
