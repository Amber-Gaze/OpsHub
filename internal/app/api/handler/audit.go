package handler

import "github.com/gin-gonic/gin"

func ListAudits(c *gin.Context) {
    c.JSON(200, gin.H{"audits": []string{}})
}
