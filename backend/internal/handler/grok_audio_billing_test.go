//go:build unit

package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type grokRealtimeAuditPromptEngine struct {
	enqueues int
}

type grokRealtimeAuditDecisionEngine struct {
	mu       sync.Mutex
	decision *securityaudit.PromptDecision
	ctx      context.Context
}

func (*grokRealtimeAuditDecisionEngine) EffectiveMode() securityaudit.Mode {
	return securityaudit.ModeBlocking
}

func (*grokRealtimeAuditDecisionEngine) Enqueue(context.Context, securityaudit.Request) error {
	return nil
}

func (p *grokRealtimeAuditDecisionEngine) Evaluate(ctx context.Context, _ securityaudit.Request) (*securityaudit.PromptDecision, error) {
	p.mu.Lock()
	p.ctx = ctx
	p.mu.Unlock()
	return p.decision, nil
}

func (p *grokRealtimeAuditDecisionEngine) relayContext() context.Context {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ctx
}

type inMemoryGatewayCache struct {
	openAIChatCompletionsGatewayCacheStub
	mu       sync.Mutex
	sessions map[string]int64
	keys     []string
}

func newInMemoryGatewayCache() *inMemoryGatewayCache {
	return &inMemoryGatewayCache{sessions: make(map[string]int64)}
}

func (c *inMemoryGatewayCache) GetSessionAccountID(_ context.Context, groupID int64, sessionHash string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = append(c.keys, sessionHash)
	accountID, ok := c.sessions[fmt.Sprintf("%d:%s", groupID, sessionHash)]
	if !ok {
		return 0, service.ErrStickySessionNotFound
	}
	return accountID, nil
}

func (c *inMemoryGatewayCache) SetSessionAccountID(_ context.Context, groupID int64, sessionHash string, accountID int64, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = append(c.keys, sessionHash)
	c.sessions[fmt.Sprintf("%d:%s", groupID, sessionHash)] = accountID
	return nil
}

func (c *inMemoryGatewayCache) RefreshSessionTTL(_ context.Context, groupID int64, sessionHash string, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = append(c.keys, sessionHash)
	return nil
}

func (c *inMemoryGatewayCache) DeleteSessionAccountID(_ context.Context, groupID int64, sessionHash string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = append(c.keys, sessionHash)
	delete(c.sessions, fmt.Sprintf("%d:%s", groupID, sessionHash))
	return nil
}

func (c *inMemoryGatewayCache) sessionKeys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.keys...)
}

func (*grokRealtimeAuditPromptEngine) EffectiveMode() securityaudit.Mode {
	return securityaudit.ModeAsync
}

func (p *grokRealtimeAuditPromptEngine) Enqueue(context.Context, securityaudit.Request) error {
	p.enqueues++
	return nil
}

func (*grokRealtimeAuditPromptEngine) Evaluate(context.Context, securityaudit.Request) (*securityaudit.PromptDecision, error) {
	return nil, nil
}

func TestGrokVoice_ReusesStandardSessionStickyAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 4061, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	accounts := []*service.Account{
		{ID: 4062, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, GroupIDs: []int64{4061}, Credentials: map[string]any{"api_key": "first-key", "base_url": "https://api.x.ai/v1", "model_mapping": map[string]any{"grok-voice-latest": "first-voice-model"}}},
		{ID: 4063, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, GroupIDs: []int64{4061}, Credentials: map[string]any{"api_key": "second-key", "base_url": "https://api.x.ai/v1", "model_mapping": map[string]any{"grok-voice-latest": "second-voice-model"}}},
	}
	upstream := &grokMediaRequestRecorder{}
	cache := newInMemoryGatewayCache()
	env := newTerminalUsageOpenAIEnvWithUpstreamAndGatewayCache(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: accounts}}, upstream, cache)
	env.handler.cfg.Gateway.Sticky.OpenAI.Enabled = true
	router := env.router("/tts", func(c *gin.Context) { env.handler.GrokVoice(c, "tts") })

	request := func() {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/tts", strings.NewReader(`{"model":"grok-voice-latest","input":"sticky voice"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("session_id", "voice-sticky-session")
		router.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		select {
		case <-env.usageRepo.created:
		case <-time.After(time.Second):
			t.Fatal("voice usage was not recorded")
		}
	}

	request()
	accounts[0].Priority, accounts[1].Priority = 2, 1
	request()

	upstream.mu.Lock()
	require.NotEmpty(t, cache.sessionKeys())
	require.NotContains(t, cache.sessionKeys(), "")
	require.Equal(t, []int64{4062, 4062}, upstream.accountIDs)
	upstream.mu.Unlock()
}

func TestGrokRealtimeAuditBodyExtractsPromptTextButNotAudio(t *testing.T) {
	for _, tt := range []struct {
		name     string
		event    string
		contains []string
		excludes []string
		nilBody  bool
	}{
		{
			name:     "session instructions",
			event:    `{"type":"session.update","session":{"instructions":"safe text","audio":"base64-secret"}}`,
			contains: []string{"safe text"},
			excludes: []string{"base64-secret"},
		},
		{
			name:     "conversation text and transcript",
			event:    `{"type":"conversation.item.create","item":{"content":[{"type":"input_text","text":"conversation text"},{"type":"input_audio","audio":"base64-secret","transcript":"recognized transcript","metadata":{"audio":"nested-audio-secret"}}]}}`,
			contains: []string{"conversation text", "recognized transcript"},
			excludes: []string{"base64-secret", "nested-audio-secret"},
		},
		{
			name:     "response instructions and input",
			event:    `{"type":"response.create","response":{"instructions":"response instruction","input":[{"content":[{"type":"input_text","text":"response input"},{"type":"input_audio","audio":"base64-secret","transcript":"response transcript","metadata":{"audio":"nested-audio-secret"}}]}]}}`,
			contains: []string{"response instruction", "response input", "response transcript"},
			excludes: []string{"base64-secret", "nested-audio-secret"},
		},
		{
			name:    "audio only",
			event:   `{"type":"input_audio_buffer.append","audio":"base64-secret"}`,
			nilBody: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			auditBody := grokRealtimeAuditBody([]byte(tt.event))
			if tt.nilBody {
				require.Nil(t, auditBody)
				return
			}
			auditText := string(auditBody)
			for _, want := range tt.contains {
				require.Contains(t, auditText, want)
			}
			for _, secret := range tt.excludes {
				require.NotContains(t, auditText, secret)
			}
		})
	}
}

func TestGrokRealtimeAuditCloseClassifiesDecision(t *testing.T) {
	for _, tt := range []struct {
		name       string
		decision   *securityaudit.Decision
		wantStatus coderws.StatusCode
		wantReason string
	}{
		{
			name:       "block is policy violation",
			decision:   &securityaudit.Decision{Kind: securityaudit.DecisionBlock, ErrorCode: securityaudit.ErrorCodeBlocked},
			wantStatus: coderws.StatusPolicyViolation,
			wantReason: securityaudit.ErrorCodeBlocked,
		},
		{
			name:       "unavailable retries",
			decision:   &securityaudit.Decision{Kind: securityaudit.DecisionUnavailable, ErrorCode: securityaudit.ErrorCodeUnavailable},
			wantStatus: coderws.StatusTryAgainLater,
			wantReason: securityaudit.ErrorCodeUnavailable,
		},
		{
			name:       "invalid retries",
			decision:   &securityaudit.Decision{Kind: securityaudit.DecisionInvalid, ErrorCode: securityaudit.ErrorCodeInvalidResponse},
			wantStatus: coderws.StatusTryAgainLater,
			wantReason: securityaudit.ErrorCodeInvalidResponse,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			status, reason := grokRealtimeAuditClose(tt.decision)
			require.Equal(t, tt.wantStatus, status)
			require.Equal(t, tt.wantReason, reason)
			require.False(t, isExpectedGrokRealtimeClose(coderws.CloseError{Code: status}))
		})
	}
}

func TestGrokRealtimeAuditGuardBindsRelayContextAndPreservesDecision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/realtime", nil)
	prompt := &grokRealtimeAuditDecisionEngine{decision: &securityaudit.PromptDecision{Kind: securityaudit.DecisionBlock, ErrorCode: securityaudit.ErrorCodeBlocked}}
	h := &OpenAIGatewayHandler{securityAuditCoordinator: securityaudit.NewCoordinator(nil, prompt)}
	apiKey := &service.APIKey{ID: 4066, UserID: 4067, GroupID: new(int64), User: &service.User{ID: 4067, Status: service.StatusActive}}
	*apiKey.GroupID = 4061
	relayCtx := context.WithValue(context.Background(), "realtime-audit", "relay")

	err := h.grokRealtimeEventGuard(c, nil, apiKey, middleware.AuthSubject{UserID: apiKey.UserID}, "grok-voice-latest")(relayCtx, []byte(`{"type":"session.update","session":{"instructions":"blocked"}}`))
	var auditErr *grokRealtimeAuditTermination
	require.ErrorAs(t, err, &auditErr)
	require.NotNil(t, auditErr.decision)
	require.Equal(t, securityaudit.DecisionBlock, auditErr.decision.Kind)
	require.Equal(t, relayCtx, prompt.relayContext())
	status, reason := grokRealtimeAuditClose(auditErr.decision)
	require.Equal(t, coderws.StatusPolicyViolation, status)
	require.Equal(t, securityaudit.ErrorCodeBlocked, reason)
}

func TestGrokRealtimeAuditUsesIndependentEventStages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/realtime", nil)
	prompt := &grokRealtimeAuditPromptEngine{}
	h := &OpenAIGatewayHandler{securityAuditCoordinator: securityaudit.NewCoordinator(nil, prompt)}
	apiKey := &service.APIKey{ID: 4064, UserID: 4065, GroupID: new(int64), User: &service.User{ID: 4065, Status: service.StatusActive}}
	*apiKey.GroupID = 4061
	subject := middleware.AuthSubject{UserID: apiKey.UserID}

	for range 2 {
		decision := h.checkSecurityAuditStage(c, nil, apiKey, subject, service.ContentModerationProtocolOpenAIResponses, "grok-voice-latest", []byte(`{"input":"safe text"}`), "grok_realtime_turn")
		require.NotNil(t, decision)
		require.True(t, decision.AllowNextStage)
	}
	require.Equal(t, 2, prompt.enqueues)
}

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
