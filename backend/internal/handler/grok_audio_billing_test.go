//go:build unit

package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGrokVoice_TTSAudioIsOmittedFromUsageSnapshots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const prompt = "voice-prompt-secret"
	const audio = "voice-audio-secret"

	group := &service.Group{ID: 4001, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          4002,
		Platform:    service.PlatformGrok,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "upstream-key", "base_url": "https://api.x.ai/v1"},
	}
	upstream := &openAIImagesHandlerHTTPUpstream{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"audio/mpeg"}},
		Body:       io.NopCloser(strings.NewReader(audio)),
	}}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}, upstream)

	var requestContext *gin.Context
	router := env.router("/tts", func(c *gin.Context) {
		requestContext = c
		env.handler.GrokVoice(c, "tts")
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tts", strings.NewReader(`{"input":"`+prompt+`"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, audio, recorder.Body.String())
	detail := middleware.GetUsageDetailSnapshot(requestContext)
	require.NotNil(t, detail)
	for _, snapshot := range []string{
		detail.RequestBody,
		detail.UpstreamRequestBody,
		detail.UpstreamResponseBody,
		detail.ResponseBody,
	} {
		require.NotContains(t, snapshot, prompt)
		require.NotContains(t, snapshot, audio)
	}
}

func TestGrokVoice_TTSFailoverUsesSelectedAccountMappedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 4011, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	accounts := []*service.Account{
		{
			ID: 4012, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
			Credentials: map[string]any{
				"api_key": "first-key", "base_url": "https://api.x.ai/v1",
				"model_mapping": map[string]any{"grok-voice-latest": "first-voice-model"},
			},
		},
		{
			ID: 4013, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2,
			Credentials: map[string]any{
				"api_key": "second-key", "base_url": "https://api.x.ai/v1",
				"model_mapping": map[string]any{"grok-voice-latest": "second-voice-model"},
			},
		},
	}
	upstream := &grokMediaRequestRecorder{statuses: []int{http.StatusInternalServerError, http.StatusOK}}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: accounts}}, upstream)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tts", strings.NewReader(`{"model":"grok-voice-latest","input":"speak"}`))
	req.Header.Set("Content-Type", "application/json")
	env.router("/tts", func(c *gin.Context) { env.handler.GrokVoice(c, "tts") }).ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	upstream.mu.Lock()
	defer upstream.mu.Unlock()
	require.Equal(t, []int64{4012, 4013}, upstream.accountIDs)
	require.Len(t, upstream.bodies, 2)
	require.Equal(t, "first-voice-model", gjson.GetBytes(upstream.bodies[0], "model").String())
	require.Equal(t, "second-voice-model", gjson.GetBytes(upstream.bodies[1], "model").String())
}

func TestIsExpectedGrokRealtimeClose(t *testing.T) {
	for _, status := range []coderws.StatusCode{
		coderws.StatusNormalClosure,
		coderws.StatusGoingAway,
		coderws.StatusNoStatusRcvd,
		coderws.StatusAbnormalClosure,
	} {
		if !isExpectedGrokRealtimeClose(coderws.CloseError{Code: status}) {
			t.Fatalf("status %v should be treated as an expected session close", status)
		}
	}
	if isExpectedGrokRealtimeClose(coderws.CloseError{Code: coderws.StatusPolicyViolation}) {
		t.Fatal("policy violations must not be treated as billable normal closes")
	}
}
