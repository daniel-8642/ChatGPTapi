package Middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
	"github.com/spf13/viper"
	"net/http"
	"time"
)

var UsersBucket map[string]*ratelimit.Bucket = nil

func initUsersBucket(fillInterval time.Duration, cap int64) {
	if UsersBucket == nil {
		list := viper.GetStringMap("UserList")
		UsersBucket = make(map[string]*ratelimit.Bucket, len(list)+3)
		for _, value := range list {
			var kmap map[string]any
			var ok bool
			if kmap, ok = value.(map[string]any); !ok {
				panic("读取用户列表有误")
			}

			if v, ok := kmap["name"].(string); ok {
				UsersBucket[v] = ratelimit.NewBucket(fillInterval, cap)
			} else {
				panic("读取用户列表有误")
			}
		}
	}
}

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

// UserRateLimitMiddleware 提供令牌桶的限流，按照用户名区分，fillInterval 加入令牌时间间隔 cap 令牌桶容量
func UserRateLimitMiddleware(fillInterval time.Duration, cap int64, errtext string) gin.HandlerFunc {
	initUsersBucket(fillInterval, cap)
	return func(c *gin.Context) {
		if UsersBucket[c.GetString("userName")].TakeAvailable(1) < 1 {
			c.String(http.StatusForbidden, errtext)
			c.Abort()
			return
		}
		c.Next()
	}
}
