package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type retentionBlockingTransport struct {
	service.HTTPUpstream
	started        chan struct{}
	release        chan struct{}
	body           string
	contentType    string
	firstStatus    int
	firstBody      string
	streamBody     bool
	attachReq      bool
	retainReq      bool
	blockedReq     *http.Request
	responseHeader http.Header
	calls          int
}

type retentionLiveGatewayCache struct {
	openAIChatCompletionsGatewayCacheStub
	openAIChatCompletionsConcurrencyCacheStub
}

type retentionLiveAttestation struct{}

func (retentionLiveAttestation) Check(context.Context) error { return nil }
func (retentionLiveAttestation) Generate(context.Context) (string, error) {
	return `{"v":1,"s":0,"t":"test"}`, nil
}

type retentionLiveCipher struct{}

func (retentionLiveCipher) Encrypt(string) (string, error) { return "encrypted", nil }
func (retentionLiveCipher) Decrypt(string) (string, error) { return `{"v":1,"s":0,"t":"test"}`, nil }

func enableRetentionLiveAttestation(t *testing.T, gateway *service.OpenAIGatewayService, cache *retentionLiveGatewayCache) {
	t.Helper()
	value := reflect.ValueOf(gateway).Elem()
	for fieldName, replacement := range map[string]any{
		"liveAttestation":       retentionLiveAttestation{},
		"liveAttestationCipher": retentionLiveCipher{},
		"concurrencyService":    service.NewConcurrencyService(cache),
	} {
		field := value.FieldByName(fieldName)
		require.True(t, field.IsValid(), fieldName)
		reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(replacement))
	}
}

func (*retentionLiveGatewayCache) SaveLiveCall(context.Context, *service.LiveCallRecord, time.Duration) error {
	return nil
}

func (*retentionLiveGatewayCache) GetLiveCall(context.Context, string) (*service.LiveCallRecord, error) {
	return nil, service.ErrLiveCallNotFound
}

func (*retentionLiveGatewayCache) ClaimLiveController(context.Context, string, string, string) (bool, error) {
	return false, nil
}

func (*retentionLiveGatewayCache) ReleaseLiveController(context.Context, string, string) (bool, error) {
	return false, nil
}

func (*retentionLiveGatewayCache) GetLiveController(context.Context, string) (string, error) {
	return "", service.ErrLiveCallNotFound
}

func (*retentionLiveGatewayCache) MarkLiveCallClosed(context.Context, string, time.Duration) (bool, error) {
	return false, nil
}

func (*retentionLiveGatewayCache) AcquireLiveLease(context.Context, int64, int, int64, int, int64, string, bool) (bool, error) {
	return true, nil
}

func (*retentionLiveGatewayCache) RefreshLiveLease(context.Context, int64, int64, int64, string) (bool, error) {
	return true, nil
}

func (*retentionLiveGatewayCache) ReleaseLiveLease(context.Context, int64, int64, int64, string) error {
	return nil
}

func (u *retentionBlockingTransport) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	if _, err := io.Copy(io.Discard, req.Body); err != nil {
		return nil, err
	}
	if err := req.Body.Close(); err != nil {
		return nil, err
	}
	u.calls++
	if u.calls == 1 && u.firstStatus != 0 {
		return &http.Response{
			StatusCode: u.firstStatus,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(u.firstBody)),
		}, nil
	}
	if u.streamBody {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {u.contentType}},
			Body: &retentionBlockingStreamBody{
				prefix:  strings.NewReader(u.body),
				started: u.started,
				release: u.release,
			},
		}
		if u.attachReq {
			resp.Request = req
		}
		return resp, nil
	}
	if u.retainReq {
		u.blockedReq = req
		defer func() { u.blockedReq = nil }()
	}
	close(u.started)
	<-u.release
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     u.responseHeaders(),
		Body:       io.NopCloser(strings.NewReader(u.body)),
	}, nil
}

func (u *retentionBlockingTransport) responseHeaders() http.Header {
	headers := u.responseHeader.Clone()
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Set("Content-Type", u.contentType)
	return headers
}

type retentionBlockingStreamBody struct {
	prefix  *strings.Reader
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *retentionBlockingStreamBody) Read(p []byte) (int, error) {
	if b.prefix.Len() > 0 {
		return b.prefix.Read(p)
	}
	b.once.Do(func() { close(b.started) })
	<-b.release
	return 0, io.EOF
}

func (*retentionBlockingStreamBody) Close() error { return nil }

func (u *retentionBlockingTransport) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func retainedHeapAfterGC() uint64 {
	runtime.GC()
	runtime.GC()
	debug.FreeOSMemory()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.HeapAlloc
}

func TestCloneGatewayParsedRequestScalarsDetachesBodyBackingArray(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-5","metadata":{"user_id":"session-user"},"output_config":{"effort":"high"},"messages":[]}`)
	parsed, err := service.ParseGatewayRequest(service.NewRequestBodyRef(body), service.PlatformAnthropic)
	require.NoError(t, err)
	require.True(t, stringBackedByBytes(parsed.Model, body))
	require.True(t, stringBackedByBytes(parsed.MetadataUserID, body))
	require.True(t, stringBackedByBytes(parsed.OutputEffort, body))

	cloneGatewayParsedRequestScalars(parsed)

	require.False(t, stringBackedByBytes(parsed.Model, body))
	require.False(t, stringBackedByBytes(parsed.MetadataUserID, body))
	require.False(t, stringBackedByBytes(parsed.OutputEffort, body))
}

func stringBackedByBytes(value string, body []byte) bool {
	if value == "" || len(body) == 0 {
		return false
	}
	valueStart := uintptr(unsafe.Pointer(unsafe.StringData(value)))
	bodyStart := uintptr(unsafe.Pointer(unsafe.SliceData(body)))
	return valueStart >= bodyStart && valueStart < bodyStart+uintptr(len(body))
}

func TestAsyncImageRequestBodyMemoryRetentionWhileWorkersBlocked(t *testing.T) {
	for run := 0; run < 3; run++ {
		t.Run(fmt.Sprintf("run-%d", run), func(t *testing.T) {
			spoolDir := useAsyncImageSpoolDir(t)
			heap2MB := measureBlockedAsyncImageHeap(t, 2<<20, 4, spoolDir)
			heap89MB := measureBlockedAsyncImageHeap(t, 89<<20/10, 4, spoolDir)
			var growth uint64
			if heap89MB > heap2MB {
				growth = heap89MB - heap2MB
			}
			require.Less(t, growth, uint64(6<<20), "four workers must not retain four complete request bodies")
		})
	}
}

func measureBlockedAsyncImageHeap(t *testing.T, size int64, workers int, spoolDir string) uint64 {
	t.Helper()
	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	release := make(chan struct{})
	tasks := service.NewImageTaskServiceWithUploader(store, nil, time.Hour, time.Minute)
	started := make(chan struct{}, workers)
	var releaseOnce sync.Once
	releaseWorkers := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(func() {
		releaseWorkers()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, tasks.Shutdown(shutdownCtx))
	})
	h := &AsyncImageHandler{tasks: tasks}
	h.execute = func(_ string, c *gin.Context) {
		started <- struct{}{}
		<-release
		c.JSON(http.StatusOK, gin.H{"data": []gin.H{{"url": "https://example.test/image.png"}}})
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		groupID := int64(1501)
		c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
			ID: 1501, UserID: 1501, GroupID: &groupID,
			Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
		})
		c.Next()
	})
	router.POST("/v1/images/generations/async", h.Submit)
	for i := 0; i < workers; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", retentionAsyncImageJSONBody(size))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusAccepted, recorder.Code, recorder.Body.String())
	}
	for i := 0; i < workers; i++ {
		waitMatrixSignal(t, started, "async image worker")
	}
	heap := retainedHeapAfterGC()
	releaseWorkers()
	require.NoError(t, tasks.Shutdown(context.Background()))
	assertMatrixTempFiles(t, spoolDir, "sub2api-request-body-", false)
	return heap
}

func TestRequestBodyMemoryRetentionWhileUpstreamBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spoolDir := t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{
		SpoolThresholdBytes: service.DefaultRequestBodySpoolThresholdBytes,
		PreviewLimitBytes:   service.DefaultRequestBodyPreviewLimitBytes,
		TempDir:             spoolDir,
	}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	for _, branch := range []string{"responses", "passthrough", "grok-responses", "responses-chat-fallback", "chat-raw", "chat-converted", "count-tokens", "openai-count-tokens", "alpha-search", "live", "live-multipart", "openai-messages", "openai-messages-chat-fallback", "gateway-chat-anthropic", "gateway-chat-gemini", "messages-anthropic", "messages-anthropic-stream", "messages-anthropic-passthrough-stream", "messages-antigravity-oauth", "messages-antigravity-upstream", "messages-bedrock", "messages-bedrock-stream", "messages-gemini", "messages-gemini-mixed", "gemini-antigravity-native"} {
		t.Run(branch, func(t *testing.T) {
			var heapAt2MB, heapAt89MB uint64
			var previewAt2MB, previewAt89MB, snapshotAt2MB, snapshotAt89MB int
			t.Run("2MB", func(t *testing.T) {
				heapAt2MB, previewAt2MB, snapshotAt2MB = measureBlockedRequestBodyHeap(t, branch, 2<<20, spoolDir)
			})
			t.Run("8.9MB", func(t *testing.T) {
				heapAt89MB, previewAt89MB, snapshotAt89MB = measureBlockedRequestBodyHeap(t, branch, 89<<20/10, spoolDir)
			})
			growth := uint64(0)
			if heapAt89MB >= heapAt2MB {
				growth = heapAt89MB - heapAt2MB
			}
			t.Logf("branch=%s heap_at_2mb=%d heap_at_8_9mb=%d retained_growth=%d preview_at_2mb=%d preview_at_8_9mb=%d snapshot_at_2mb=%d snapshot_at_8_9mb=%d", branch, heapAt2MB, heapAt89MB, growth, previewAt2MB, previewAt89MB, snapshotAt2MB, snapshotAt89MB)

			require.Less(t, growth, uint64(3<<20), "retained heap must not scale with full request body size")
			require.LessOrEqual(t, previewAt2MB, int(service.DefaultRequestBodyPreviewLimitBytes))
			require.LessOrEqual(t, previewAt89MB, int(service.DefaultRequestBodyPreviewLimitBytes))
			require.LessOrEqual(t, snapshotAt2MB, int(service.DefaultRequestBodyPreviewLimitBytes))
			require.LessOrEqual(t, snapshotAt89MB, int(service.DefaultRequestBodyPreviewLimitBytes))
		})
	}

	t.Run("ordinary preview boundary", func(t *testing.T) {
		limit := int(service.DefaultRequestBodyPreviewLimitBytes)
		const prefix = `{"message":"`
		const suffix = `"}`
		text := strings.Repeat("plain text ", limit/len("plain text ")+1)
		preview := prefix + text[:limit-len(prefix)-len(suffix)] + suffix
		require.Len(t, preview, limit)

		raw := service.RequestBodyPreviewSnapshot(preview, int64(len(preview)+1))
		var snapshot matrixRequestBodyPreview
		require.NoError(t, json.Unmarshal([]byte(raw), &snapshot))
		require.NotContains(t, snapshot.Preview, "inline binary payload omitted")
		require.Greater(t, len(snapshot.Preview), limit-(1<<10))
		require.LessOrEqual(t, len(snapshot.Preview), limit)
		require.LessOrEqual(t, len(raw), limit)
		t.Logf("ordinary_preview_bytes=%d serialized_snapshot_bytes=%d", len(snapshot.Preview), len(raw))
	})
}

func measureBlockedRequestBodyHeap(t *testing.T, branch string, size int64, spoolDir string) (uint64, int, int) {
	t.Helper()
	upstream := &retentionBlockingTransport{
		started:     make(chan struct{}),
		release:     make(chan struct{}),
		contentType: "application/json",
		body:        `{"id":"resp_retention","output":[],"usage":{"input_tokens":1,"output_tokens":1}}`,
	}
	done := make(chan struct{})
	release := cleanupMatrixBlockedHandler(t, upstream.release, done)
	var requestContext *gin.Context
	var router http.Handler
	path := "/v1/responses"
	platform := service.PlatformOpenAI
	accountType := service.AccountTypeAPIKey
	extra := map[string]any{}
	switch branch {
	case "passthrough":
		extra["openai_passthrough"] = true
	case "grok-responses":
		platform = service.PlatformGrok
	case "responses-chat-fallback":
		extra["openai_responses_mode"] = "force_chat_completions"
		upstream.body = `{"id":"chatcmpl_retention","choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`
	case "chat-raw":
		path = "/v1/chat/completions"
		extra["openai_responses_supported"] = false
		upstream.body = `{"id":"chatcmpl_retention","choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`
	case "chat-converted":
		path = "/v1/chat/completions"
		extra["openai_responses_supported"] = true
		upstream.contentType = "text/event-stream"
		upstream.body = "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_retention\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"
	case "count-tokens":
		path = "/v1/messages/count_tokens"
		platform = service.PlatformAnthropic
		upstream.body = `{"input_tokens":1}`
	case "openai-count-tokens":
		path = "/v1/messages/count_tokens"
		upstream.body = `{"input_tokens":1}`
	case "alpha-search":
		path = "/v1/alpha/search"
		upstream.body = `{"type":"computer_initialize_state","id":"search_retention"}`
	case "live", "live-multipart":
		path = "/v1/realtime/calls"
		upstream.contentType = "application/sdp"
		upstream.body = "v=0\r\n"
		upstream.responseHeader = http.Header{"Location": {"/backend-api/codex/call_retention"}}
	case "openai-messages":
		path = "/v1/messages"
		extra["openai_responses_supported"] = true
		upstream.contentType = "text/event-stream"
		upstream.body = "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_retention\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"
	case "openai-messages-chat-fallback":
		path = "/v1/messages"
		extra["openai_responses_supported"] = false
		upstream.retainReq = true
		upstream.body = `{"id":"chatcmpl_retention","choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`
	case "gateway-chat-anthropic":
		path = "/v1/chat/completions"
		platform = service.PlatformAnthropic
		upstream.contentType = "text/event-stream"
		upstream.body = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_retention\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-sonnet-4-5\",\"stop_reason\":\"\",\"usage\":{\"input_tokens\":1}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"ok\"}}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":1}}\n\n"
	case "gateway-chat-gemini":
		path = "/v1/chat/completions"
		platform = service.PlatformGemini
		upstream.body = `{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`
	case "messages-anthropic":
		path = "/v1/messages"
		platform = service.PlatformAnthropic
		upstream.body = `{"id":"msg_retention","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"model":"claude-sonnet-4-5","stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`
	case "messages-anthropic-stream", "messages-anthropic-passthrough-stream":
		path = "/v1/messages"
		platform = service.PlatformAnthropic
		upstream.contentType = "text/event-stream"
		upstream.streamBody = true
		upstream.body = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_retention\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-sonnet-4-5\",\"content\":[],\"stop_reason\":null,\"usage\":{\"input_tokens\":1,\"output_tokens\":0}}}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":1}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
		if branch == "messages-anthropic-passthrough-stream" {
			extra["anthropic_passthrough"] = true
		}
	case "messages-antigravity-oauth":
		path = "/v1/messages"
		platform = service.PlatformAntigravity
		accountType = service.AccountTypeOAuth
		upstream.contentType = "text/event-stream"
		upstream.body = "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":1,\"candidatesTokenCount\":1}}}\n\n"
	case "messages-antigravity-upstream":
		path = "/v1/messages"
		platform = service.PlatformAntigravity
		accountType = service.AccountTypeUpstream
		upstream.body = `{"id":"msg_retention","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"model":"claude-sonnet-4-5","stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`
	case "messages-bedrock", "messages-bedrock-stream":
		path = "/v1/messages"
		platform = service.PlatformAnthropic
		accountType = service.AccountTypeBedrock
		upstream.body = `{"id":"msg_retention","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"model":"claude-sonnet-4-5","stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`
		if branch == "messages-bedrock-stream" {
			upstream.contentType = "application/vnd.amazon.eventstream"
			upstream.streamBody = true
			upstream.attachReq = true
			upstream.body = ""
		}
	case "messages-gemini", "messages-gemini-mixed":
		path = "/v1/messages"
		platform = service.PlatformGemini
		upstream.body = `{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`
	case "gemini-antigravity-native":
		path = "/antigravity/v1beta/models/gemini-2.5-flash:generateContent"
		platform = service.PlatformAntigravity
		accountType = service.AccountTypeOAuth
		upstream.contentType = "text/event-stream"
		upstream.body = "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":1,\"candidatesTokenCount\":1}}}\n\n"
	}
	if branch == "messages-gemini-mixed" {
		initialID, intermediateID, finalID := int64(1401), int64(1402), int64(1403)
		initialGroup := &service.Group{ID: initialID, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true, FallbackGroupIDOnInvalidRequest: &intermediateID}
		intermediateGroup := &service.Group{ID: intermediateID, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
		finalGroup := &service.Group{ID: finalID, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
		initialAccount := &service.Account{ID: 1401, Name: "retention-antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, AccountGroups: []service.AccountGroup{{GroupID: initialID}}, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
		finalAccount := &service.Account{ID: 1403, Name: "retention-gemini", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, AccountGroups: []service.AccountGroup{{GroupID: finalID}}, Credentials: map[string]any{"api_key": "sk-retention"}}
		upstream.firstStatus = http.StatusBadRequest
		upstream.firstBody = `{"error":{"message":"Prompt is too long"}}`
		env := newTerminalGatewayMessagesEnvWithGatewayCacheAndGroups(t, initialGroup, map[int64]*service.Group{initialID: initialGroup, intermediateID: intermediateGroup, finalID: finalGroup}, upstream, openAIChatCompletionsConcurrencyCacheStub{}, openAIChatCompletionsGatewayCacheStub{}, initialAccount, finalAccount)
		router = env.routerFor(path, func(c *gin.Context) {
			requestContext = c
			env.handler.Messages(c)
		})
	} else if branch == "gemini-antigravity-native" {
		group := &service.Group{ID: 1401, Platform: platform, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{
			ID: 1401, Name: "retention-antigravity-native", Platform: service.PlatformAntigravity, Type: accountType,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1,
			Credentials: map[string]any{"access_token": "token", "project_id": "project", "model_mapping": map[string]any{"gemini-2.5-flash": "gemini-2.5-flash"}},
		}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
		router = env.routerFor("/antigravity/v1beta/models/*modelAction", func(c *gin.Context) {
			requestContext = c
			c.Set(string(middleware.ContextKeyForcePlatform), service.PlatformAntigravity)
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, service.PlatformAntigravity))
			env.handler.GeminiV1BetaModels(c)
		})
	} else if branch == "count-tokens" {
		group := &service.Group{ID: 1401, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 1401, Name: "retention-count-tokens", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-retention"}}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
		router = env.routerFor(path, func(c *gin.Context) {
			requestContext = c
			env.handler.CountTokens(c)
		})
	} else if branch == "openai-count-tokens" {
		group := &service.Group{ID: 1401, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
		account := &service.Account{ID: 1401, Name: "retention-openai-count-tokens", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"access_token": "token"}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
		router = env.router(path, func(c *gin.Context) {
			requestContext = c
			env.handler.CountTokens(c)
		})
	} else if branch == "alpha-search" {
		group := &service.Group{ID: 1401, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 1401, Name: "retention-alpha-search", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-retention"}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
		router = env.router(path, func(c *gin.Context) {
			requestContext = c
			env.handler.AlphaSearch(c)
		})
	} else if branch == "live" || branch == "live-multipart" {
		group := &service.Group{ID: 1401, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowLive: true}
		account := &service.Account{ID: 1401, Name: "retention-live", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"access_token": "token", "chatgpt_account_id": "acct-retention"}}
		cache := &retentionLiveGatewayCache{}
		env := newTerminalUsageOpenAIEnvWithUpstreamAndGatewayCache(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream, cache)
		enableRetentionLiveAttestation(t, env.handler.gatewayService, cache)
		router = env.router(path, func(c *gin.Context) {
			requestContext = c
			env.handler.Live(c)
		})
	} else if branch == "openai-messages" || branch == "openai-messages-chat-fallback" {
		group := &service.Group{ID: 1401, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
		account := &service.Account{ID: 1401, Name: "retention-openai-messages", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-retention"}, Extra: extra}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
		env.handler.cfg.Gateway.OpenAIWS.Enabled = false
		router = env.router(path, func(c *gin.Context) {
			requestContext = c
			env.handler.Messages(c)
		})
	} else if branch == "gateway-chat-anthropic" || branch == "gateway-chat-gemini" {
		group := &service.Group{ID: 1401, Platform: platform, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 1401, Name: "retention", Platform: platform, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-retention"}}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
		router = env.routerFor(path, func(c *gin.Context) {
			requestContext = c
			env.handler.ChatCompletions(c)
		})
	} else if path == "/v1/messages" {
		group := &service.Group{ID: 1401, Platform: platform, Status: service.StatusActive, Hydrated: true}
		credentials := map[string]any{"api_key": "sk-retention"}
		if accountType == service.AccountTypeBedrock {
			credentials["auth_mode"] = "apikey"
			credentials["aws_region"] = "us-east-1"
		}
		if accountType == service.AccountTypeOAuth {
			credentials = map[string]any{
				"access_token": "token",
				"project_id":   "project",
				"model_mapping": map[string]any{
					"claude-sonnet-4-5": "gemini-3-pro-high",
				},
			}
		}
		if accountType == service.AccountTypeUpstream {
			credentials["base_url"] = "https://example.com"
		}
		account := &service.Account{ID: 1401, Name: "retention", Platform: platform, Type: accountType, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: credentials, Extra: extra}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
		router = env.routerFor(path, func(c *gin.Context) {
			requestContext = c
			env.handler.Messages(c)
		})
	} else {
		group := &service.Group{ID: 1401, Platform: platform, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 1401, Name: "retention", Platform: platform, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-retention"}, Extra: extra}
		if branch == "grok-responses" {
			account.Credentials["base_url"] = "https://api.x.ai/v1"
		}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
		env.handler.cfg.Gateway.OpenAIWS.Enabled = false
		router = env.router(path, func(c *gin.Context) {
			requestContext = c
			if path == "/v1/chat/completions" {
				env.handler.ChatCompletions(c)
				return
			}
			env.handler.Responses(c)
		})
	}
	recorder := httptest.NewRecorder()
	requestBody := retentionJSONBody(branch, path, size)
	contentType := "application/json"
	if branch == "live-multipart" {
		const boundary = "sub2api-retention-boundary"
		requestBody = retentionLiveMultipartBody(size, boundary)
		contentType = "multipart/form-data; boundary=" + boundary
	}
	req := httptest.NewRequest(http.MethodPost, path, requestBody)
	req.Header.Set("Content-Type", contentType)
	go func() {
		router.ServeHTTP(recorder, req)
		close(done)
	}()
	waitMatrixSignal(t, upstream.started, "retention upstream")

	detail := middleware.GetUsageDetailSnapshot(requestContext)
	require.NotNil(t, detail)
	var snapshot matrixRequestBodyPreview
	if branch != "count-tokens" && branch != "openai-count-tokens" && branch != "alpha-search" && branch != "live" && branch != "live-multipart" {
		require.NoError(t, json.Unmarshal([]byte(detail.UpstreamRequestBody), &snapshot))
		require.Greater(t, snapshot.Size, size-(1<<10))
		require.True(t, snapshot.Truncated)
	}
	heap := retainedHeapAfterGC()
	previewSize := len(snapshot.Preview)
	snapshotSize := len(detail.UpstreamRequestBody)

	release()
	waitMatrixSignal(t, done, "retention handler completion")
	wantStatus := http.StatusOK
	require.Equal(t, wantStatus, recorder.Code, recorder.Body.String())
	if branch == "live" || branch == "live-multipart" {
		require.Equal(t, 1, upstream.calls, "Live retention case must reach transport")
	}
	assertMatrixTempFiles(t, spoolDir, "sub2api-request-body-", false)
	return heap, previewSize, snapshotSize
}

func retentionLiveMultipartBody(size int64, boundary string) io.Reader {
	prefix := "--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"sdp\"\r\n\r\n" +
		"v=0\r\n\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"session\"\r\n\r\n" +
		`{"model":"gpt-live","instructions":"`
	suffix := `"}` + "\r\n--" + boundary + "--\r\n"
	return io.MultiReader(
		strings.NewReader(prefix),
		io.LimitReader(retentionPaddingReader{}, size-int64(len(prefix)+len(suffix))),
		strings.NewReader(suffix),
	)
}

func retentionJSONBody(branch, path string, size int64) io.Reader {
	prefix, suffix := `{"model":"gpt-5","stream":false,"input":"`, `"}`
	if branch == "grok-responses" {
		prefix = `{"model":"grok-4.5","stream":false,"input":"`
	}
	if path == "/v1/chat/completions" {
		prefix, suffix = `{"model":"gpt-5","stream":false,"messages":[{"role":"user","content":"`, `"}]}`
		if branch == "gateway-chat-anthropic" {
			prefix = `{"model":"claude-sonnet-4-5","stream":false,"messages":[{"role":"user","content":"`
		} else if branch == "gateway-chat-gemini" {
			prefix = `{"model":"gemini-2.5-flash","stream":false,"messages":[{"role":"user","content":"`
		}
	} else if path == "/v1/messages" {
		prefix, suffix = `{"model":"gemini-2.5-flash","max_tokens":16,"stream":false,"messages":[{"role":"user","content":"`, `"}]}`
	}
	if branch == "openai-messages" || branch == "openai-messages-chat-fallback" {
		prefix = `{"model":"gpt-5","max_tokens":16,"stream":false,"messages":[{"role":"user","content":"`
	}
	if branch == "messages-anthropic" || branch == "messages-anthropic-stream" || branch == "messages-anthropic-passthrough-stream" || branch == "messages-antigravity-oauth" || branch == "messages-antigravity-upstream" || branch == "messages-bedrock" || branch == "messages-bedrock-stream" {
		stream := "false"
		if branch == "messages-anthropic-stream" || branch == "messages-anthropic-passthrough-stream" || branch == "messages-bedrock-stream" {
			stream = "true"
		}
		prefix = `{"model":"claude-sonnet-4-5","max_tokens":16,"stream":` + stream + `,"messages":[{"role":"user","content":"`
	}
	if branch == "gemini-antigravity-native" {
		prefix, suffix = `{"contents":[{"role":"user","parts":[{"text":"`, `"}]}]}`
	}
	if branch == "count-tokens" {
		prefix, suffix = `{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"`, `"}]}`
	}
	if branch == "alpha-search" {
		prefix, suffix = `{"model":"gpt-5","query":"`, `"}`
	}
	if branch == "live" {
		prefix, suffix = `{"sdp":"v=0\\r\\n","session":{"model":"gpt-live","instructions":"`, `"}}`
	}
	if branch == "messages-gemini-mixed" {
		prefix = `{"model":"claude-opus-4-6","max_tokens":16,"stream":false,"messages":[{"role":"user","content":"`
	}
	return io.MultiReader(
		strings.NewReader(prefix),
		io.LimitReader(retentionPaddingReader{}, size-int64(len(prefix)+len(suffix))),
		strings.NewReader(suffix),
	)
}

func retentionAsyncImageJSONBody(size int64) io.Reader {
	const prefix = `{"model":"gpt-image-1","prompt":"`
	const suffix = `"}`
	return io.MultiReader(
		strings.NewReader(prefix),
		io.LimitReader(retentionPaddingReader{}, size-int64(len(prefix)+len(suffix))),
		strings.NewReader(suffix),
	)
}

type retentionPaddingReader struct{}

func (retentionPaddingReader) Read(p []byte) (int, error) {
	const text = "ordinary request preview text "
	for i := range p {
		p[i] = text[i%len(text)]
	}
	return len(p), nil
}
