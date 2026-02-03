package api

import (
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/transport"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/utils"
	"github.com/fasthttp/router"
)

func RegisterRoutes(r *router.Router, svc *Service) {
	group := transport.NewRouterGroup(r, "", middleware.Recover())
	handler := NewHandler(svc)

	group.POST("/signup", handler.Signup)
	group.POST("/login", handler.Login)
	group.POST("/logout", handler.Logout)
	group.POST("/refresh", handler.Refresh)
	users := group.Group(utils.UserPath)
	{
		users.GET("/", handler.ListUsers)
		users.GET("/:name", handler.GetUser)
		users.PUT(":name/change-passwd", handler.ChangePassword)
		users.PUT("/:name", handler.UpdateUser)
		users.DELETE("/:name", handler.DeleteUser)
		users.DELETE("/", handler.DeleteUsers)
	}
}
