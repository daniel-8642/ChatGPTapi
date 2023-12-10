package Middleware

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"strings"
)

func Auth(prefixkey string) gin.HandlerFunc {
	if prefixkey == "" || !viper.IsSet(prefixkey) { // 当未配置密码时，直接通过请求
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
			if err, _ := VerifyUser(prefixkey, authrow); err != nil {
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

type UserInfo struct {
	Name string
}

func VerifyUser(prefixkey string, passwd string) (err error, userInfo UserInfo) {
	if viper.IsSet(prefixkey + "." + passwd) {
		err = viper.UnmarshalKey(prefixkey+"."+passwd, &userInfo)
		return
	} else {
		return errors.New("none List"), UserInfo{}
	}
}
