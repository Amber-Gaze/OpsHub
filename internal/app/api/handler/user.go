package handler

import (
	"net/http"

	"github.com/Amber-Gaze/OpsHub/internal/domain"
	"github.com/Amber-Gaze/OpsHub/internal/service"

	"github.com/gin-gonic/gin"
)

func ListUsers(svc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := svc.List()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, users)
	}
}

func CreateUser(svc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body domain.User
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		svc.Create(&body)
		c.JSON(http.StatusCreated, body)
	}
}
