package api

import (
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/transport"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/utils"
	"github.com/fasthttp/router"
)

type RoutesConfig struct {
	AuthBaseURL   string
	AuthorizePath string
	LoginPath     string
	RateLimitRPS  int
}

func RegisterRoutes(r *router.Router, svc *Service, cfg RoutesConfig) {
	group := transport.NewRouterGroup(r, "", middleware.RequestID(), middleware.Recover())

	handler := NewHandler(svc, HandlerOptions{AuthLoginPath: "/login"})

	group.Use(middleware.RateLimit(cfg.RateLimitRPS))
	group.GET("/healthz", handler.Health)
	group.GET("/readyz", handler.Ready)
	group.POST("/login", handler.Login)
	group.POST("/logout", handler.Logout)
	group.POST("/refresh", handler.Refresh)
	group.POST("/scope", handler.Scope)
	group.POST("/signup", handler.Signup) // 公开注册（IAM 侧首个用户自动成为管理员）

	group.Use(middleware.JWTAuthMiddleware())

	// 配置中心：经 RequireAuthDecision（IAM scope）鉴权后透传，配置中心按 scope 精确过滤/校验。
	configs := group.Group(utils.ConfigPath, RequireAuthDecision(svc))
	{
		// 下游服务拉取 + 历史对比透传
		configs.GET("/pull", handler.PullConfigs)
		configs.GET("/history/{path:*}", handler.GetConfigHistory)

		// 控制台分层浏览透传：/configs/tree[/business[/module[/name]]]
		configs.GET("/tree", handler.GetTree)
		configs.GET("/tree/{business}", handler.GetBusiness)
		configs.GET("/tree/{business}/{module}", handler.GetModule)
		configs.GET("/tree/{business}/{module}/{name}", handler.GetItem)
		configs.PUT("/tree/{business}/{module}/{name}", handler.UpdateItem)
		configs.DELETE("/tree/{business}/{module}/{name}", handler.DeleteItem)

		// 扁平 CRUD 透传（兼容单段 key）
		configs.GET("/", handler.ListConfigs)
		configs.GET("/{key}", handler.GetConfig)
		configs.POST("/", handler.CreateConfig)
		configs.PUT("/{key}", handler.UpdateConfig)
		configs.DELETE("/{key}", handler.DeleteConfig)
	}

	// 用户与策略管理：透传到 IAM，由 IAM 自行 JWT/管理员鉴权。
	// 网关作为统一入口，后续 IAM 新增接口只需在 IAM 侧注册路由即可复用透传。
	users := group.Group(utils.UserPath)
	{
		users.GET("/", handler.ForwardUsers)
		users.DELETE("/", handler.ForwardUsers)
		users.GET("/{name}", handler.ForwardUsers)
		users.PUT("/{name}", handler.ForwardUsers)
		users.DELETE("/{name}", handler.ForwardUsers)
		users.GET("/{name}/grants", handler.ForwardUsers)
		users.PUT("/{name}/change-passwd", handler.ForwardUsers)
	}

	policies := group.Group("/policies")
	{
		policies.GET("/", handler.ForwardPolicies)
		policies.POST("/rule", handler.ForwardPolicies)
		policies.POST("/rule/delete", handler.ForwardPolicies)
		policies.POST("/config-grant", handler.ForwardPolicies)
		policies.POST("/config-revoke", handler.ForwardPolicies)
		policies.POST("/roles", handler.ForwardPolicies)
		policies.POST("/roles/delete", handler.ForwardPolicies)
	}
	middleware.AttachRouterErrors(r)
}
