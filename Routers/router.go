package Routers

import (
	"GPTapi/Middleware"
	"GPTapi/SendSyslog"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
	"golang.org/x/net/proxy"
	"io"
	"net/http"
	"net/url"
	"time"
)

var client *openai.Client
var sendLog SendSyslog.Syslogger

func SetUpRouter(api *gin.Engine) {
	sendLog = SendSyslog.GetLogger()
	api.Use(Middleware.Cors)
	api.POST("/chat-process", Middleware.Auth("UserList"), Middleware.RateLimitMiddleware(time.Second*10, 10), Middleware.UserRateLimitMiddleware(time.Second*20, 1, "rate limit..."), chatProcess)
	api.POST("/config", Middleware.Auth("UserList"), Middleware.RateLimitMiddleware(time.Second, 30), config)
	api.POST("/session", Middleware.RateLimitMiddleware(time.Second, 30), sessiondata)
	api.POST("/verify", Middleware.RateLimitMiddleware(time.Second*5, 5), verify)
	var cliconfig openai.ClientConfig
	if viper.GetBool("Use_Azure") {
		cliconfig = openai.DefaultAzureConfig(viper.GetString("Azure_OpenAI.API_Key"), viper.GetString("Azure_OpenAI.Endpoint"))
		cliconfig.AzureModelMapperFunc = func(model string) string {
			return map[string]string{
				"gpt-3.5-turbo": viper.GetString("Azure_OpenAI.ModelName"),
			}[model]
		}
	} else {
		cliconfig = openai.DefaultConfig(viper.GetString("OpenAI.API_Key"))
		cliconfig.BaseURL = viper.GetString("OpenAI.Base_URL") + "/v1"
	}
	// 判断代理
	if viper.GetString("HttpsProxy") != "" {
		u := url.URL{}
		urlproxy, _ := u.Parse(viper.GetString("HttpsProxy"))
		cliconfig.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(urlproxy),
			},
		}
	} else if viper.GetString("Socks_Proxy.Host") != "" &&
		viper.GetInt("Socks_Proxy.Port") > 0 &&
		viper.GetInt("Socks_Proxy.Port") <= 65535 {
		proxyURL, err := url.Parse("socks5://" + viper.GetString("Socks_Proxy.Host") + ":" + viper.GetString("Socks_Proxy.Port"))
		if err != nil {
			fmt.Println("Parse socks5 address error")
			panic(err)
		}
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			fmt.Println("Diale socks5 failed")
			panic(err)
		}
		transport := &http.Transport{
			Dial: dialer.Dial,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		cliconfig.HTTPClient = &http.Client{
			Transport: transport,
		}
	}
	client = openai.NewClientWithConfig(cliconfig)
}

func chatProcess(c *gin.Context) {
	reqBody := struct {
		Prompt        string `json:"prompt" binding:"required"`
		SystemMessage string `json:"systemMessage" `
	}{}
	err := c.ShouldBindJSON(&reqBody)
	if err != nil {
		fmt.Printf("read text error: %v\n", err)
		return
	}
	sendLog(reqBody.Prompt + "[" + c.GetString("userName") + "]")
	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		MaxTokens: 1500,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: reqBody.SystemMessage,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: reqBody.Prompt,
			},
		},
		Stream: true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("Requests.Timeout")*time.Millisecond)
	defer cancel()
	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		fmt.Printf("ChatCompletionStream error: %v\n", err)
		return
	}
	chanStream := make(chan []byte, 100)
	initTemplate := false
	template := struct {
		Role            string                              `json:"role"`
		Id              string                              `json:"id"`
		ParentMessageId string                              `json:"parentMessageId"`
		Text            string                              `json:"text"`
		Detail          openai.ChatCompletionStreamResponse `json:"detail"`
	}{
		Role: "assistant",
		Text: "",
	}
	go func() {
		defer stream.Close()
		defer close(chanStream)
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				fmt.Printf("\nStream error: %v\n", err)
				return
			}
			template.Detail = response
			template.Text += response.Choices[0].Delta.Content
			if !initTemplate {
				template.Role = "assistant"
				template.Id = template.Detail.ID
				template.ParentMessageId = "1234567890123-1234-1234-123456789012"
				initTemplate = true
			}
			marshal, _ := json.Marshal(template)
			chanStream <- append(marshal, '\n')
		}
	}()
	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-chanStream; ok {
			_, err := w.Write(msg)
			if err != nil {
				return false
			}
			return true
		}
		return false
	})
}

func sessiondata(c *gin.Context) {
	data := map[string]any{
		"status":  "Success",
		"message": "",
		"data": map[string]any{
			"auth":  viper.IsSet("UserList"),
			"model": "ChatGPTAPI",
		},
	}
	c.JSON(200, &data)
}

func config(c *gin.Context) {
	data := map[string]any{
		"status":  "Success",
		"message": nil,
		"data": map[string]any{
			"apiModel":     "ChatGPTAPI",
			"balance":      "0.000000000000001",
			"httpsProxy":   "-",
			"reverseProxy": "",
			"socksProxy":   "-",
			"timeoutMs":    viper.GetInt("Requests.Timeout"),
		},
	}
	c.JSON(200, &data)
}

func verify(c *gin.Context) {
	reqBody := struct {
		Token string `json:"token" binding:"required"`
	}{}
	err := c.ShouldBindJSON(&reqBody)
	if err != nil {
		fmt.Printf("read text error: %v\n", err)
		data := map[string]any{
			"status":  "Fail",
			"message": "密钥无效 | Secret key is invalid",
			"data":    nil,
		}
		c.JSON(200, &data)
		return
	}

	if err, userInfo := Middleware.VerifyUser("UserList", reqBody.Token); err == nil {
		data := map[string]any{
			"status":  "Success",
			"message": "登录成功 | Login succeeded",
			"data": map[string]any{
				"name": userInfo.Name,
			},
		}
		c.JSON(200, data)
	} else {
		data := map[string]any{
			"status":  "Fail",
			"message": "密钥无效 | Secret key is invalid",
			"data":    nil,
		}
		c.JSON(200, &data)
		return
	}
}

//var ErrorCodeMessage = map[int]string{
//	401: "[OpenAI] 提供错误的API密钥 | Incorrect API key provided",
//	403: "[OpenAI] 服务器拒绝访问，请稍后再试 | Server refused to access, please try again later",
//	502: "[OpenAI] 错误的网关 |  Bad Gateway",
//	503: "[OpenAI] 服务器繁忙，请稍后再试 | Server is busy, please try again later",
//	504: "[OpenAI] 网关超时 | Gateway Time-out",
//	500: "[OpenAI] 服务器繁忙，请稍后再试 | Internal Server Error",
//}
