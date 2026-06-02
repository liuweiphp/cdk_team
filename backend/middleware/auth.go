package middleware

import (
	"encoding/gob"
	"exchange_cdk/pkg/jwt"
	"strings"

	"github.com/gin-gonic/gin"
)

// ClaimsKey 是 context 中存储 jwt claims 的 key
const ClaimsKey = "claims"

// AuthRequired JWT 鉴权中间件,从 Authorization: Bearer 头提取 token 并验证
func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			respondUnauthorized(c)
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := jwt.Parse(secret, tokenStr)
		if err != nil {
			respondUnauthorized(c)
			return
		}
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

func respondUnauthorized(c *gin.Context) {
	c.JSON(401, gin.H{"code": 40101, "message": "请先登录", "data": nil})
	c.Abort()
}

func init() {
	gob.Register(&jwt.Claims{})
}
