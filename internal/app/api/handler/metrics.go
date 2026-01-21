package handler

import "github.com/gin-gonic/gin"

func Metrics(c *gin.Context) {
    c.JSON(200, gin.H{"metrics": "ok"})
}
