package api

import (
	"github.com/Amber-Gaze/OpsHub/internal/app/api/handler"
	"github.com/Amber-Gaze/OpsHub/internal/infrastructure"
	"github.com/Amber-Gaze/OpsHub/internal/repository/mysql"
	"github.com/Amber-Gaze/OpsHub/internal/service"

	"github.com/Amber-Gaze/OpsHub/internal/app/api/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	r.Use(middleware.RateLimit())

	enforcer := infrastructure.InitCasbin("your_dsn_here")

	userRepo := mysql.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo)

	projectRepo := mysql.NewProjectRepository(db)
	projectSvc := service.NewProjectService(projectRepo)

	r.POST("/users", handler.CreateUser(userSvc))
	r.GET("/users", handler.ListUsers(userSvc))

	r.POST("/projects", handler.CreateProject(projectSvc))
	r.GET("/projects", handler.ListProjects(projectSvc))

	// …继续补齐其它
}
