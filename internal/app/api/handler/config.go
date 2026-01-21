package handler

import (
	"net/http"

	"github.com/Amber-Gaze/OpsHub/internal/service"
	"github.com/gin-gonic/gin"
)

var configSvc = service.NewConfigService()

func ListConfigs(c *gin.Context) {
	data, _ := configSvc.List()
	c.JSON(http.StatusOK, data)
}
