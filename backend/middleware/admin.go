package middleware

import (
	"exchange_cdk/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// AdminRequired role=admin 校验,需在 AuthRequired 之后使用
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.Get(ClaimsKey)
		if !ok {
			c.JSON(403, gin.H{"code": 40301, "message": "权限不足", "data": nil})
			c.Abort()
			return
		}
		cv, ok := claims.(*jwt.Claims)
		if !ok || cv.Role != "admin" {
			c.JSON(403, gin.H{"code": 40301, "message": "权限不足", "data": nil})
			c.Abort()
			return
		}
		c.Next()
	}
}
