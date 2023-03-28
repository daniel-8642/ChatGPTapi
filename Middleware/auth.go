package Middleware

import (
	"github.com/gin-gonic/gin"
	"strings"
)

func Auth(key string) gin.HandlerFunc {
	if key == "" { // 当未配置密码时，直接通过请求
		return func(c *gin.Context) {
			c.Next()
		}
	} else {
		return func(c *gin.Context) {
			authrow := c.GetHeader("Authorization")
			if len(authrow) == 0 || !strings.HasPrefix(authrow, "Bearer ") {
				c.Abort()
				data := map[string]any{
					"status": "Unauthorized", "message": "Error: 无访问权限 | No access rights", "data": nil,
				}
				c.JSON(200, data)
				return
			}
			authrow = strings.Fields(authrow)[1]
			if authrow != key {
				c.Abort()
				data := map[string]any{
					"status": "Unauthorized", "message": "Error: 无访问权限 | No access rights", "data": nil,
				}
				c.JSON(200, data)
				return
			} else {
				c.Next()
			}
		}
	}
}
