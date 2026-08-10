//go:build unit

package handler

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestGrokVoice_NoModelRoutesScheduleVoiceOnlyMappedAccountNeutrally(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tt := range []struct {
		endpoint string
		body     string
	}{
		{endpoint: "tts", body: `{"input":"speak"}`},
		{endpoint: "custom-voices", body: `{"name":"voice profile"}`},
	} {
		t.Run(tt.endpoint, func(t *testing.T) {
			group := &service.Group{ID: 4051, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
			account := &service.Account{
				ID: 4052, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
				Status: service.StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{
					"api_key": "voice-key", "base_url": "https://api.x.ai/v1",
					"model_mapping": map[string]any{"grok-voice-latest": "mapped-voice-model"},
				},
			}
			upstream := &grokMediaRequestRecorder{}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}, upstream)

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/"+tt.endpoint, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			env.router("/"+tt.endpoint, func(c *gin.Context) { env.handler.GrokVoice(c, tt.endpoint) }).ServeHTTP(recorder, req)

			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			upstream.mu.Lock()
			require.Equal(t, []int64{account.ID}, upstream.accountIDs)
			require.Len(t, upstream.bodies, 1)
			require.Empty(t, gjson.GetBytes(upstream.bodies[0], "model").String())
			upstream.mu.Unlock()

			if tt.endpoint == "tts" {
				select {
				case usageLog := <-env.usageRepo.created:
					require.Equal(t, "tts", usageLog.Model)
					require.Nil(t, usageLog.UpstreamModel)
				case <-time.After(time.Second):
					t.Fatal("TTS usage was not recorded")
				}
			}
		})
	}
}

func TestGrokVoice_STTMultipartFailoverUsesMappedModelAndPreservesAudio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 4021, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	accounts := []*service.Account{
		{
			ID: 4022, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
			Credentials: map[string]any{
				"api_key": "first-key", "base_url": "https://api.x.ai/v1",
				"model_mapping": map[string]any{"grok-stt": "first-stt-model"},
			},
		},
		{
			ID: 4023, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2,
			Credentials: map[string]any{
				"api_key": "second-key", "base_url": "https://api.x.ai/v1",
				"model_mapping": map[string]any{"grok-stt": "second-stt-model"},
			},
		},
	}
	upstream := &grokMediaRequestRecorder{statuses: []int{http.StatusInternalServerError, http.StatusOK}}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: accounts}}, upstream)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "grok-stt"))
	require.NoError(t, writer.WriteField("language", "en"))
	audio, err := writer.CreateFormFile("file", "speech.wav")
	require.NoError(t, err)
	_, err = audio.Write([]byte("stt-audio-bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/stt", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	env.router("/stt", func(c *gin.Context) { env.handler.GrokVoice(c, "stt") }).ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	upstream.mu.Lock()
	defer upstream.mu.Unlock()
	require.Equal(t, []int64{4022, 4023}, upstream.accountIDs)
	require.Len(t, upstream.bodies, 2)
	for i, wantModel := range []string{"first-stt-model", "second-stt-model"} {
		model, audioBytes := grokVoiceMultipartModelAndAudio(t, upstream.bodies[i], upstream.contentTypes[i])
		require.Equal(t, wantModel, model)
		require.Equal(t, []byte("stt-audio-bytes"), audioBytes)
	}
}

func TestForwardGrokVoice_NoModelUsesEndpointWithoutFalseUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 4031, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 4032, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "voice-key", "base_url": "https://api.x.ai/v1"},
	}
	upstream := &grokMediaRequestRecorder{}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}, upstream)

	for _, tt := range []struct {
		endpoint string
		body     string
	}{
		{endpoint: "tts", body: `{"text":"hello","language":"en"}`},
		{endpoint: "custom-voices", body: `{"name":"voice profile"}`},
	} {
		t.Run(tt.endpoint, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/"+tt.endpoint, strings.NewReader(tt.body))
			result, err := env.handler.gatewayService.ForwardGrokVoice(c.Request.Context(), c, account, tt.endpoint, []byte(tt.body), "application/json")
			require.NoError(t, err)
			require.Equal(t, tt.endpoint, result.Model)
			require.Empty(t, result.UpstreamModel)
		})
	}

	upstream.mu.Lock()
	defer upstream.mu.Unlock()
	for _, body := range upstream.bodies {
		require.Empty(t, gjson.GetBytes(body, "model").String())
	}
}

func TestRecordGrokVoiceUsage_RealtimePreservesMappedUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 4041, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 4042, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "voice-key", "base_url": "https://api.x.ai/v1"},
	}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}, &grokMediaRequestRecorder{})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/realtime?model=grok-voice-latest", nil)
	result := &service.OpenAIForwardResult{
		Model:         "grok-voice-latest",
		UpstreamModel: "mapped-realtime",
		AudioUsage:    &service.AudioUsage{Mode: "realtime", DurationOrUnits: 1},
	}

	env.handler.recordGrokVoiceUsage(c, env.apiKey, account, nil, "realtime", nil, result)
	usageLog := <-env.usageRepo.created
	require.Equal(t, "grok-voice-latest", usageLog.Model)
	require.NotNil(t, usageLog.UpstreamModel)
	require.Equal(t, "mapped-realtime", *usageLog.UpstreamModel)
}

func grokVoiceMultipartModelAndAudio(t *testing.T, body []byte, contentType string) (string, []byte) {
	t.Helper()
	_, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	var model, audio []byte
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		data, err := io.ReadAll(part)
		require.NoError(t, err)
		switch part.FormName() {
		case "model":
			model = data
		case "file":
			audio = data
		}
		require.NoError(t, part.Close())
	}
	return string(model), audio
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
