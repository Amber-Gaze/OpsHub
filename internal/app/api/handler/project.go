package handler

import (
	"net/http"

	"github.com/Amber-Gaze/OpsHub/internal/domain"
	"github.com/Amber-Gaze/OpsHub/internal/service"

	"github.com/gin-gonic/gin"
)

func CreateProject(svc *service.ProjectService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body domain.Project
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		svc.Create(&body)
		c.JSON(http.StatusCreated, body)
	}
}

func ListProjects(svc *service.ProjectService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ps, _ := svc.List()
		c.JSON(http.StatusOK, ps)
	}
}
