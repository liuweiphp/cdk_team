package middleware

import (
	"time"

	"exchange_cdk/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestLogger 请求日志中间件,注入 trace_id 并记录 method/path/status/latency/user_id
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := uuid.New().String()
		c.Set("trace_id", traceID)
		c.Header("X-Trace-ID", traceID)

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		userID := getUserID(c)

		logger.Info("request",
			zap.String("trace_id", traceID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.Uint("user_id", userID),
			zap.String("ip", c.ClientIP()),
		)
	}
}

func getUserID(c *gin.Context) uint {
	if claims, ok := c.Get(ClaimsKey); ok {
		if cv, ok := claims.(*jwt.Claims); ok {
			return cv.UserID
		}
	}
	return 0
}
