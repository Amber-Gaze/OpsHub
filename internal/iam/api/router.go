package api

import (
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/transport"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/utils"
	"github.com/fasthttp/router"
)

func RegisterRoutes(r *router.Router, svc *Service) {
	group := transport.NewRouterGroup(r, "", middleware.RequestID(), middleware.Recover())
	handler := NewHandler(svc)

	group.GET("/healthz", handler.Healthz)
	group.GET("/readyz", handler.Readyz)

	group.POST("/signup", handler.Signup)
	group.POST("/login", handler.Login)
	group.POST("/logout", handler.Logout)
	group.POST("/refresh", handler.Refresh)
	group.POST("/authorize", handler.Authorize)

	users := group.Group(utils.UserPath, middleware.JWTAuthMiddleware())
	{
		users.GET("/", handler.ListUsers, middleware.RequireAdmin())
		users.DELETE("/", handler.DeleteUsers, middleware.RequireAdmin())
		users.DELETE("/:name", handler.DeleteUser)
		users.GET("/:name", handler.GetUser, middleware.RequireSelfOrAdmin())
		users.PUT("/:name/change-passwd", handler.ChangePassword, middleware.RequireSelfOrAdmin())
		users.PUT("/:name", handler.UpdateUser, middleware.RequireSelfOrAdmin())
	}
	middleware.AttachRouterErrors(r)
}
