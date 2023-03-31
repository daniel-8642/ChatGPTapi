package Routers

import (
	"GPTapi/Middleware"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
	"io"
	"time"
)

var client *openai.Client

func SetUpRouter(api *gin.Engine) {
	api.Use(Middleware.Cors)
	api.POST("/chat-process", Middleware.Auth(viper.GetString("Requests.AuthSecretKey")), Middleware.RateLimitMiddleware(time.Second*100, 10), chatProcess)
	api.POST("/config", Middleware.Auth(viper.GetString("Requests.AuthSecretKey")), Middleware.RateLimitMiddleware(time.Second, 30), config)
	api.POST("/session", Middleware.RateLimitMiddleware(time.Second, 30), sessiondata)
	api.POST("/verify", Middleware.RateLimitMiddleware(5*time.Second, 5), verify)
	cliconfig := openai.DefaultConfig(viper.GetString("OpenAI.API_Key"))
	cliconfig.BaseURL = viper.GetString("OpenAI.Base_URL") + "/v1"
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
	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		MaxTokens: 600,
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
		"data":    map[string]any{"auth": viper.GetString("Requests.AuthSecretKey") != "", "model": "ChatGPTAPI"},
	}
	c.JSON(200, &data)
}

func config(c *gin.Context) {
	data := map[string]any{
		"status":  "Success",
		"message": nil,
		"data": map[string]any{
			"apiModel":     "ChatGPTAPI",
			"balance":      "4.938",
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
	if viper.GetString("Requests.AuthSecretKey") == "" ||
		viper.GetString("Requests.AuthSecretKey") == reqBody.Token {
		data := map[string]any{
			"status":  "Success",
			"message": "登录成功 | Login succeeded",
			"data":    nil,
		}
		c.JSON(200, data)
		return
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
