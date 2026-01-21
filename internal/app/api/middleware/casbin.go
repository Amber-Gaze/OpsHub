package middleware

import "github.com/gin-gonic/gin"

func Casbin() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
    }
}
