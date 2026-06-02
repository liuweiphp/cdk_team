package handler

import (
	"exchange_cdk/middleware"
	"exchange_cdk/pkg/jwt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Healthz 存活探针
func Healthz(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

// Readyz 就绪探针,检查数据库连接
func Readyz(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(503, gin.H{"status": "not ready"})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "not ready"})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	}
}

// queryInt 从 query 参数获取 int
func queryInt(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n := 0
	for _, r := range v {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// queryFloat 从 query 参数获取 float64
func queryFloat(c *gin.Context, key string, def float64) float64 {
	v := c.Query(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

// getUserID 从 context 中获取当前登录用户ID
func getUserID(c *gin.Context) uint {
	if claims, ok := c.Get(middleware.ClaimsKey); ok {
		if cv, ok := claims.(*jwt.Claims); ok {
			return cv.UserID
		}
	}
	return 0
}
