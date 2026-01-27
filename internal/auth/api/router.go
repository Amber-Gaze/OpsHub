package api

import (
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/transport"
	"github.com/fasthttp/router"
)

func RegisterRoutes(r *router.Router, svc *Service) {
	group := transport.NewRouterGroup(r, "", middleware.Recover())
	h := NewHandler(svc)

	auth := group.Group("/auth")
	{
		// External auth APIs
		auth.POST("/login", h.Login)
		auth.POST("/authorize", h.Authorize)
	}
}
