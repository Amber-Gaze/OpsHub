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
	group.POST("/scope", handler.Scope)

	pol := group.Group("/policies", middleware.JWTAuthMiddleware())
	{
		pol.GET("/", handler.ListPolicies, middleware.RequireAdmin())
		pol.POST("/rule", handler.AddPolicyRule)
		pol.POST("/rule/delete", handler.RemovePolicyRule)
		pol.POST("/config-grant", handler.ConfigGrant)
		pol.POST("/config-revoke", handler.ConfigRevoke)
		pol.POST("/roles", handler.AddRoleBinding)
		pol.POST("/roles/delete", handler.RemoveRoleBinding)
	}

	users := group.Group(utils.UserPath, middleware.JWTAuthMiddleware())
	{
		users.GET("/", handler.ListUsers, middleware.RequireAdmin())
		users.DELETE("/", handler.DeleteUsers, middleware.RequireAdmin())
		users.DELETE("/{name}", handler.DeleteUser)
		users.GET("/{name}", handler.GetUser, middleware.RequireSelfOrAdmin())
		users.GET("/{name}/grants", handler.UserGrants, middleware.RequireAdmin())
		users.PUT("/{name}/change-passwd", handler.ChangePassword, middleware.RequireSelfOrAdmin())
		users.PUT("/{name}", handler.UpdateUser, middleware.RequireSelfOrAdmin())
	}

	// 服务凭证（AccessKey）：程序化鉴权凭证，本人或管理员管理；下游服务用它自签 JWT
	aks := group.Group("/accesskeys", middleware.JWTAuthMiddleware())
	{
		aks.POST("/", handler.CreateAccessKey)
		aks.GET("/", handler.ListAccessKeys)
		aks.DELETE("/{keyID}", handler.DeleteAccessKey)
	}

	// 服务模块订阅：注册哪些模块即可拉取对应配置（只读）；仅管理员可注册/取消
	services := group.Group("/services", middleware.JWTAuthMiddleware())
	{
		services.GET("/{name}/modules", handler.GetServiceModules, middleware.RequireAdmin())
		services.PUT("/{name}/modules", handler.SetServiceModules, middleware.RequireAdmin())
		services.DELETE("/{name}/modules", handler.RemoveServiceModule, middleware.RequireAdmin())
	}
	middleware.AttachRouterErrors(r)
}
