package api

import (
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/fasthttp/router"
)

func RegisterRoutes(r *router.Router, svc *Service, middlewares ...middleware.Middleware) {
	handler := NewHandler(svc)
	chain := append([]middleware.Middleware{middleware.Recover()}, middlewares...)

	r.GET("/configs", middleware.Chain(handler.List, chain...))
	r.GET("/configs/:key", middleware.Chain(handler.Get, chain...))
	r.POST("/configs", middleware.Chain(handler.Create, chain...))
	r.PUT("/configs/:key", middleware.Chain(handler.Update, chain...))
	r.DELETE("/configs/:key", middleware.Chain(handler.Delete, chain...))
}
