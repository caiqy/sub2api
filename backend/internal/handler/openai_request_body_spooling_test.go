package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayHandler_ChatAndEmbeddingsReplayMappedSpoolAcrossFailover(t *testing.T) {
	for _, tt := range []struct {
		name  string
		route string
		body  func(string) []byte
	}{
		{"chat", "/v1/chat/completions", func(padding string) []byte {
			return []byte(`{"model":"alias-model","messages":[{"role":"user","content":"` + padding + `"}],"stream":false}`)
		}},
		{"embeddings", "/v1/embeddings", func(padding string) []byte { return []byte(`{"model":"alias-model","input":"` + padding + `"}`) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Keep the handler path real; the transport is the only controlled boundary.
			rawDir, upstreamDir := t.TempDir(), t.TempDir()
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 64, TempDir: rawDir, FilePrefix: "sub2api-test-"}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
			t.Setenv("TMPDIR", upstreamDir)
			t.Setenv("TMP", upstreamDir)
			t.Setenv("TEMP", upstreamDir)

			body := tt.body(strings.Repeat("x", 12<<20))
			upstream := &openAIReplaySpoolUpstream{started: make(chan struct{}), release: make(chan struct{})}
			group := &service.Group{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
			accounts := []*service.Account{
				{ID: 11, Name: "first", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-first", "model_mapping": map[string]any{"alias-model": "mapped-model"}}},
				{ID: 12, Name: "second", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "sk-second", "model_mapping": map[string]any{"alias-model": "mapped-model"}}},
			}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: accounts}, upstream)
			env.handler.maxAccountSwitches = 1
			router := env.router(tt.route, func(c *gin.Context) {
				if tt.name == "chat" {
					env.handler.ChatCompletions(c)
				} else {
					env.handler.Embeddings(c)
				}
			})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tt.route, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			done := make(chan struct{})
			go func() { router.ServeHTTP(rec, req); close(done) }()
			<-upstream.started
			require.NotEmpty(t, readTestDir(t, rawDir), "raw spool must survive the blocked upstream attempt")
			require.NotEmpty(t, readTestDir(t, upstreamDir), "mapped effective spool must survive the blocked upstream attempt")
			close(upstream.release)
			<-done
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			upstream.assert(t, body)
			require.Empty(t, readTestDir(t, rawDir))
			require.Empty(t, readTestDir(t, upstreamDir))
		})
	}
}

type openAIReplaySpoolUpstream struct {
	service.HTTPUpstream
	mu      sync.Mutex
	bodies  [][]byte
	lengths []int64
	hashes  []string
	started chan struct{}
	release chan struct{}
}

func (u *openAIReplaySpoolUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	getBody, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	reopened, err := io.ReadAll(getBody)
	if err != nil {
		return nil, err
	}
	_ = getBody.Close()
	if !bytes.Equal(body, reopened) {
		return nil, io.ErrUnexpectedEOF
	}
	sum := sha256.Sum256(body)
	u.mu.Lock()
	u.bodies = append(u.bodies, body)
	u.lengths = append(u.lengths, req.ContentLength)
	u.hashes = append(u.hashes, hex.EncodeToString(sum[:]))
	attempt := len(u.bodies)
	u.mu.Unlock()
	if attempt == 1 {
		return openAIReplayResponse(http.StatusTooManyRequests), nil
	}
	close(u.started)
	<-u.release
	return openAIReplayResponse(http.StatusOK, req.URL.Path), nil
}

func (u *openAIReplaySpoolUpstream) DoWithTLS(req *http.Request, proxy string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxy, accountID, concurrency)
}

func (u *openAIReplaySpoolUpstream) assert(t *testing.T, raw []byte) {
	t.Helper()
	u.mu.Lock()
	defer u.mu.Unlock()
	require.Len(t, u.bodies, 2)
	for i, body := range u.bodies {
		require.Equal(t, int64(len(body)), u.lengths[i])
		sum := sha256.Sum256(body)
		require.Equal(t, hex.EncodeToString(sum[:]), u.hashes[i])
		require.Contains(t, string(body), `"model":"mapped-model"`)
	}
	require.NotEqual(t, u.bodies[0], raw)
	require.Equal(t, u.bodies[0], u.bodies[1])
}

func openAIReplayResponse(status int, path ...string) *http.Response {
	if status == http.StatusTooManyRequests {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"retry"}}`))}
	}
	if len(path) != 0 && path[0] != "/v1/embeddings" {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_123\",\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"ok\"}]}],\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"))}
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"ok","model":"mapped-model","choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2},"data":[{"embedding":[0.1],"index":0}],"object":"list"}`))}
}

func readTestDir(t *testing.T, dir string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	return entries
}
