package SendSyslog

import (
	"fmt"
	"github.com/spf13/viper"
	"log/syslog"
	"math/rand"
)

var sysLogger *syslog.Writer

type Syslogger func(str string) int

func Dial() {
	var err error
	sysLogger, err = syslog.Dial(
		viper.GetString("SysLog.protocol"),
		viper.GetString("SysLog.RemoteAddrPort"),
		syslog.LOG_INFO|syslog.LOG_USER,
		viper.GetString("SysLog.tag"))
	if err != nil {
		fmt.Println("连接远程访问记录失败")
		sysLogger = nil
	}
}

func Send(str string) (len int) {
	len = 0
	if sysLogger != nil {
		len, _ = sysLogger.Write([]byte(str))
	} else if rand.Intn(100) <= 10 {
		// 10%概率重试连接
		Dial()
	}
	return
}

func GetLogger() Syslogger {
	if sysLogger != nil {
		return Send
	} else {
		Dial()
		if sysLogger != nil {
			return Send
		} else {
			return func(str string) int { return 0 }
		}
	}
}
