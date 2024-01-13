package main

import (
	"GPTapi/Routers"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	initConfig()
}

func main() {
	api := gin.Default()
	// 只信任内网http代理，影响Context.ClientIP()获取的ip，这里影响日志中来源ip的记录
	err := api.SetTrustedProxies([]string{
		"10.0.0.0/8",
		"172.17.0.0/12",
		"192.168.0.0/16",
	})
	if err != nil {
		return
	}
	Routers.SetUpRouter(api)
	_ = api.Run(":" + viper.GetString("Service.Port"))
}

func initConfig() {
	viper.SetConfigName("config")   // 配置文件名字，注意没有扩展名
	viper.SetConfigType("yaml")     // 如果配置文件的名称中没有包含扩展名，那么该字段是必需的
	viper.AddConfigPath("./Config") // 在当前工作目录寻找配置文件
	err := viper.ReadInConfig()     // 查找并读取配置文件
	if err != nil {
		panic(errors.Errorf("Fatal error config file: %v \n", err))
	}
}
