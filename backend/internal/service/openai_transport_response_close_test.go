package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type transportResponseCloseUpstream struct {
	HTTPUpstream
	response *http.Response
	err      error
}

func (u transportResponseCloseUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	return u.response, u.err
}

type transportResponseCloseBody struct {
	io.Reader
	closed bool
}

func (b *transportResponseCloseBody) Close() error {
	b.closed = true
	return nil
}

func TestOpenAIChatCompletionsOpenAIPassthroughGrokMediaTransportErrorClosesResponseBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		run  func(*OpenAIGatewayService) error
	}{
		{
			name: "chat completions conversion",
			run: func(service *OpenAIGatewayService) error {
				body := []byte(`{"model":"gpt-5.1","messages":[{"role":"user","content":"hello"}],"stream":false}`)
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
				_, err := service.ForwardAsChatCompletions(context.Background(), c, openAITestOAuthAccount(), body, "", "")
				return err
			},
		},
		{
			name: "OpenAI passthrough",
			run: func(service *OpenAIGatewayService) error {
				body := []byte(`{"model":"gpt-5.2","input":"hello","stream":false}`)
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
				account := openAITestOAuthAccount()
				account.Extra = map[string]any{"openai_passthrough": true}
				_, err := service.Forward(context.Background(), c, account, body)
				return err
			},
		},
		{
			name: "Grok media",
			run: func(service *OpenAIGatewayService) error {
				body := []byte(`{"model":"grok-imagine-image","prompt":"draw"}`)
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
				account := &Account{ID: 2, Name: "grok-media", Platform: PlatformGrok, Type: AccountTypeAPIKey, Concurrency: 1, Credentials: map[string]any{"api_key": "key", "base_url": "https://api.x.ai/v1"}}
				_, err := service.ForwardGrokMedia(context.Background(), c, account, GrokMediaEndpointImagesGenerations, "", body, "application/json")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseBody := &transportResponseCloseBody{Reader: strings.NewReader(`{"error":"partial"}`)}
			service := &OpenAIGatewayService{
				cfg: &config.Config{},
				httpUpstream: transportResponseCloseUpstream{
					response: &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}, Body: responseBody},
					err:      errors.New("transport failure"),
				},
			}

			err := tt.run(service)

			require.Error(t, err)
			require.True(t, responseBody.closed)
		})
	}
}
