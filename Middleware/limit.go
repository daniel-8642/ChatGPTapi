package Middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
	"net/http"
	"time"
)

// RateLimitMiddleware 提供令牌桶的限流，fillInterval 加入令牌时间间隔 cap 令牌桶容量
func RateLimitMiddleware(fillInterval time.Duration, cap int64) gin.HandlerFunc {
	bucket := ratelimit.NewBucket(fillInterval, cap)
	return func(c *gin.Context) {
		if bucket.TakeAvailable(1) < 1 {
			c.String(http.StatusForbidden, "rate limit...")
			c.Abort()
			return
		}
		c.Next()
	}
}
