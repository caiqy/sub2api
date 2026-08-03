package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type retentionBlockingTransport struct {
	service.HTTPUpstream
	started     chan struct{}
	release     chan struct{}
	body        string
	contentType string
	firstStatus int
	firstBody   string
	streamBody  bool
	calls       int
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
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {u.contentType}},
			Body: &retentionBlockingStreamBody{
				prefix:  strings.NewReader(u.body),
				started: u.started,
				release: u.release,
			},
		}, nil
	}
	close(u.started)
	<-u.release
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {u.contentType}},
		Body:       io.NopCloser(strings.NewReader(u.body)),
	}, nil
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

	for _, branch := range []string{"responses", "passthrough", "grok-responses", "responses-chat-fallback", "chat-raw", "chat-converted", "messages-anthropic", "messages-anthropic-stream", "messages-anthropic-passthrough-stream", "messages-gemini", "messages-gemini-mixed"} {
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
	case "messages-gemini", "messages-gemini-mixed":
		path = "/v1/messages"
		platform = service.PlatformGemini
		upstream.body = `{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`
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
	} else if path == "/v1/messages" {
		group := &service.Group{ID: 1401, Platform: platform, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 1401, Name: "retention", Platform: platform, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-retention"}, Extra: extra}
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
	req := httptest.NewRequest(http.MethodPost, path, retentionJSONBody(branch, path, size))
	req.Header.Set("Content-Type", "application/json")
	go func() {
		router.ServeHTTP(recorder, req)
		close(done)
	}()
	waitMatrixSignal(t, upstream.started, "retention upstream")

	detail := middleware.GetUsageDetailSnapshot(requestContext)
	require.NotNil(t, detail)
	var snapshot matrixRequestBodyPreview
	require.NoError(t, json.Unmarshal([]byte(detail.UpstreamRequestBody), &snapshot))
	require.Greater(t, snapshot.Size, size-(1<<10))
	require.True(t, snapshot.Truncated)
	heap := retainedHeapAfterGC()
	previewSize := len(snapshot.Preview)
	snapshotSize := len(detail.UpstreamRequestBody)

	release()
	waitMatrixSignal(t, done, "retention handler completion")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assertMatrixTempFiles(t, spoolDir, "sub2api-request-body-", false)
	return heap, previewSize, snapshotSize
}

func retentionJSONBody(branch, path string, size int64) io.Reader {
	prefix, suffix := `{"model":"gpt-5","stream":false,"input":"`, `"}`
	if branch == "grok-responses" {
		prefix = `{"model":"grok-4.5","stream":false,"input":"`
	}
	if path == "/v1/chat/completions" {
		prefix, suffix = `{"model":"gpt-5","stream":false,"messages":[{"role":"user","content":"`, `"}]}`
	} else if path == "/v1/messages" {
		prefix, suffix = `{"model":"gemini-2.5-flash","max_tokens":16,"stream":false,"messages":[{"role":"user","content":"`, `"}]}`
	}
	if branch == "messages-anthropic" || branch == "messages-anthropic-stream" || branch == "messages-anthropic-passthrough-stream" {
		stream := "false"
		if branch == "messages-anthropic-stream" || branch == "messages-anthropic-passthrough-stream" {
			stream = "true"
		}
		prefix = `{"model":"claude-sonnet-4-5","max_tokens":16,"stream":` + stream + `,"messages":[{"role":"user","content":"`
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

type retentionPaddingReader struct{}

func (retentionPaddingReader) Read(p []byte) (int, error) {
	const text = "ordinary request preview text "
	for i := range p {
		p[i] = text[i%len(text)]
	}
	return len(p), nil
}
