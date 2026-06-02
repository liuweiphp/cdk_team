package middleware

import (
	"exchange_cdk/pkg/jwt"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen int64
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     rate.Limit
	burst    int
}

func newRateLimiter(r rate.Limit, burst int) *rateLimiter {
	return &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    burst,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, ok := rl.visitors[key]
	if !ok {
		rl.visitors[key] = &visitor{limiter: rate.NewLimiter(rl.rate, rl.burst)}
		return true
	}
	return v.limiter.Allow()
}

// ExchangeRateLimit 限流中间件: 同一 user 每分钟 ≤ 10 次, 同一 IP 每分钟 ≤ 20 次
func ExchangeRateLimit() gin.HandlerFunc {
	userLimiter := newRateLimiter(10.0/60.0, 10)
	ipLimiter := newRateLimiter(20.0/60.0, 20)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		userKey := "ip:" + ip
		if claims, ok := c.Get(ClaimsKey); ok {
			if cv, ok := claims.(*jwt.Claims); ok {
				userKey = "user:" + fmt.Sprintf("%d", cv.UserID)
			}
		}

		if !ipLimiter.allow("ip:" + ip) {
			c.JSON(429, gin.H{"code": 42901, "message": "请求过于频繁,请稍后再试", "data": nil})
			c.Abort()
			return
		}
		if !userLimiter.allow(userKey) {
			c.JSON(429, gin.H{"code": 42901, "message": "请求过于频繁,请稍后再试", "data": nil})
			c.Abort()
			return
		}
		c.Next()
	}
}
