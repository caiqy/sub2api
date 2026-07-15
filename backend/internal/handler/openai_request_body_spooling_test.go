package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIGatewayHandler_ChatAndEmbeddingsReplayMappedSpoolAcrossFailover(t *testing.T) {
	for _, tt := range []struct {
		name         string
		route        string
		body         func(string) []byte
		upstreamPath string
		gzip         bool
	}{
		{"chat", "/v1/chat/completions", func(padding string) []byte {
			return []byte(`{"model":"alias-model","messages":[{"role":"user","content":"` + padding + `"}],"stream":false}`)
		}, "/v1/responses", false},
		{"chat-compressed", "/v1/chat/completions", func(padding string) []byte {
			return []byte(`{"model":"alias-model","messages":[{"role":"user","content":"` + padding + `"}],"stream":false}`)
		}, "/v1/responses", true},
		{"embeddings", "/v1/embeddings", func(padding string) []byte { return []byte(`{"model":"alias-model","input":"` + padding + `"}`) }, "/v1/embeddings", false},
		{"embeddings-compressed", "/v1/embeddings", func(padding string) []byte { return []byte(`{"model":"alias-model","input":"` + padding + `"}`) }, "/v1/embeddings", true},
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
			upstream := &openAIReplaySpoolUpstream{started: make(chan struct{}), release: make(chan struct{}), snapshots: make(chan openAIReplaySnapshots, 1)}
			group := &service.Group{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
			accounts := []*service.Account{
				{ID: 11, Name: "first", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-first", "model_mapping": map[string]any{"alias-model": "mapped-model"}}},
				{ID: 12, Name: "second", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "sk-second", "model_mapping": map[string]any{"alias-model": "mapped-model"}}},
			}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: accounts}, upstream)
			env.handler.maxAccountSwitches = 1
			var requestContext *gin.Context
			router := env.router(tt.route, func(c *gin.Context) {
				requestContext = c
				if tt.route == "/v1/chat/completions" {
					env.handler.ChatCompletions(c)
				} else {
					env.handler.Embeddings(c)
				}
			})
			upstream.snapshot = func() openAIReplaySnapshots {
				detail := middleware.GetUsageDetailSnapshot(requestContext)
				ops, _ := requestContext.Get(service.OpsUpstreamRequestBodyKey)
				opsBody, _ := ops.(string)
				return openAIReplaySnapshots{usageRequest: detail.RequestBody, usageUpstream: detail.UpstreamRequestBody, opsUpstream: opsBody}
			}
			rec := httptest.NewRecorder()
			requestBody := body
			if tt.gzip {
				var compressed bytes.Buffer
				zipper := gzip.NewWriter(&compressed)
				require.NoError(t, func() error {
					_, err := zipper.Write(body)
					return err
				}())
				require.NoError(t, zipper.Close())
				requestBody = compressed.Bytes()
			}
			req := httptest.NewRequest(http.MethodPost, tt.route, bytes.NewReader(requestBody))
			req.Header.Set("Content-Type", "application/json")
			if tt.gzip {
				req.Header.Set("Content-Encoding", "gzip")
			}
			done := make(chan struct{})
			go func() { router.ServeHTTP(rec, req); close(done) }()
			waitOpenAIReplaySignal(t, upstream.started, "second upstream attempt")
			snapshot := waitOpenAIReplaySnapshot(t, upstream.snapshots)
			assertBoundedOpenAIReplaySnapshot(t, "usage request", snapshot.usageRequest, body)
			assertBoundedOpenAIReplaySnapshot(t, "usage upstream", snapshot.usageUpstream, upstream.latestBody())
			assertBoundedOpenAIReplaySnapshot(t, "ops upstream", snapshot.opsUpstream, upstream.latestBody())
			require.NotEmpty(t, readTestDir(t, rawDir), "raw spool must survive the blocked upstream attempt")
			require.NotEmpty(t, readTestDir(t, upstreamDir), "mapped effective spool must survive the blocked upstream attempt")
			close(upstream.release)
			waitOpenAIReplaySignal(t, done, "handler completion")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			upstream.assert(t, body, tt.upstreamPath)
			require.Empty(t, readTestDir(t, rawDir))
			require.Empty(t, readTestDir(t, upstreamDir))
		})
	}
}

func TestOpenAIGatewayHandler_ChatReplayRawSpoolAcrossFailoverWhenResponsesUnsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawDir, upstreamDir := t.TempDir(), t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 64, TempDir: rawDir, FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
	t.Setenv("TMPDIR", upstreamDir)
	t.Setenv("TMP", upstreamDir)
	t.Setenv("TEMP", upstreamDir)

	body := []byte(`{"model":"alias-model","messages":[{"role":"user","content":"` + strings.Repeat("x", 12<<20) + `"}],"stream":false}`)
	upstream := &openAIReplaySpoolUpstream{started: make(chan struct{}), release: make(chan struct{}), snapshots: make(chan openAIReplaySnapshots, 1)}
	group := &service.Group{ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	accounts := []*service.Account{
		{ID: 21, Name: "first", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-first", "model_mapping": map[string]any{"alias-model": "mapped-model"}}, Extra: map[string]any{openai_compat.ExtraKeyResponsesSupported: false}},
		{ID: 22, Name: "second", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "sk-second", "model_mapping": map[string]any{"alias-model": "mapped-model"}}, Extra: map[string]any{openai_compat.ExtraKeyResponsesSupported: false}},
	}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: accounts}, upstream)
	env.handler.maxAccountSwitches = 1
	var requestContext *gin.Context
	router := env.router("/v1/chat/completions", func(c *gin.Context) {
		requestContext = c
		env.handler.ChatCompletions(c)
	})
	upstream.snapshot = func() openAIReplaySnapshots {
		detail := middleware.GetUsageDetailSnapshot(requestContext)
		ops, _ := requestContext.Get(service.OpsUpstreamRequestBodyKey)
		opsBody, _ := ops.(string)
		return openAIReplaySnapshots{usageRequest: detail.RequestBody, usageUpstream: detail.UpstreamRequestBody, opsUpstream: opsBody}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(rec, req)
		close(done)
	}()

	waitOpenAIReplaySignal(t, upstream.started, "raw Chat second upstream attempt")
	snapshot := waitOpenAIReplaySnapshot(t, upstream.snapshots)
	assertBoundedOpenAIReplaySnapshot(t, "usage request", snapshot.usageRequest, body)
	assertBoundedOpenAIReplaySnapshot(t, "usage upstream", snapshot.usageUpstream, upstream.latestBody())
	assertBoundedOpenAIReplaySnapshot(t, "ops upstream", snapshot.opsUpstream, upstream.latestBody())
	require.NotEmpty(t, readTestDir(t, rawDir), "raw spool must survive the blocked upstream attempt")
	require.NotEmpty(t, readTestDir(t, upstreamDir), "mapped effective spool must survive the blocked upstream attempt")
	close(upstream.release)
	waitOpenAIReplaySignal(t, done, "raw Chat handler completion")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	upstream.assert(t, body, "/v1/chat/completions")
	require.Empty(t, readTestDir(t, rawDir))
	require.Empty(t, readTestDir(t, upstreamDir))
}

func TestOpenAIGatewayHandler_ChatAndEmbeddingsUpstreamErrorsPreserveStatusAndMapping(t *testing.T) {
	for _, tt := range []struct {
		name  string
		route string
		body  string
	}{
		{"chat", "/v1/chat/completions", `{"model":"alias-model","messages":[{"role":"user","content":"small"}],"stream":false}`},
		{"embeddings", "/v1/embeddings", `{"model":"alias-model","input":"small"}`},
	} {
		for _, status := range []struct {
			upstream int
			want     int
		}{{http.StatusBadRequest, http.StatusBadRequest}, {http.StatusInternalServerError, http.StatusBadGateway}} {
			t.Run(tt.name+"-"+http.StatusText(status.upstream), func(t *testing.T) {
				upstream := &openAIStatusSpoolUpstream{status: status.upstream}
				group := &service.Group{ID: 3, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
				account := &service.Account{ID: 31, Name: "status", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-status", "model_mapping": map[string]any{"alias-model": "mapped-model"}}}
				env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
				router := env.router(tt.route, func(c *gin.Context) {
					if tt.route == "/v1/chat/completions" {
						env.handler.ChatCompletions(c)
						return
					}
					env.handler.Embeddings(c)
				})
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, tt.route, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(rec, req)

				require.Equal(t, status.want, rec.Code, rec.Body.String())
				require.NotEmpty(t, gjson.Get(rec.Body.String(), "error.message").String(), rec.Body.String())
				require.Contains(t, string(upstream.body), `"model":"mapped-model"`)
			})
		}
	}
}

func TestOpenAIGatewayHandler_ChatAndEmbeddingsSpoolFailuresReturn503WithoutUsage(t *testing.T) {
	for _, tt := range []struct {
		name  string
		route string
		body  string
	}{
		{"chat", "/v1/chat/completions", `{"model":"alias-model","messages":[{"role":"user","content":"hello"}]}`},
		{"embeddings", "/v1/embeddings", `{"model":"alias-model","input":"hello"}`},
	} {
		t.Run(tt.name+"/raw-read", func(t *testing.T) {
			group := &service.Group{ID: 5, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
			account := &service.Account{ID: 51, Name: "spool", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-spool"}}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, &openAIStatusSpoolUpstream{})
			router := env.router(tt.route, func(c *gin.Context) {
				if tt.route == "/v1/chat/completions" {
					env.handler.ChatCompletions(c)
					return
				}
				env.handler.Embeddings(c)
			})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tt.route, nil)
			req.Body = io.NopCloser(requestBodyErrorReader{err: service.ErrRequestBodySpool})
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
			require.Equal(t, "api_error", gjson.Get(rec.Body.String(), "error.type").String())
			require.Nil(t, env.usageRepo.lastLog)
		})

		if tt.name != "chat" {
			continue
		}
		t.Run(tt.name+"/effective-open", func(t *testing.T) {
			rawDir, effectiveDir := t.TempDir(), t.TempDir()
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: rawDir}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
			t.Setenv("TMPDIR", effectiveDir)
			t.Setenv("TMP", effectiveDir)
			t.Setenv("TEMP", effectiveDir)

			group := &service.Group{ID: 6, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
			account := &service.Account{ID: 61, Name: "spool", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-spool"}}
			upstream := &openAISpoolDeletingUpstream{dirs: []string{rawDir, effectiveDir}}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
			router := env.router(tt.route, func(c *gin.Context) {
				if tt.route == "/v1/chat/completions" {
					env.handler.ChatCompletions(c)
					return
				}
				env.handler.Embeddings(c)
			})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tt.route, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
			require.Equal(t, "api_error", gjson.Get(rec.Body.String(), "error.type").String())
			require.Nil(t, env.usageRepo.lastLog)
			require.Len(t, upstream.accountIDs, 1)
			require.Empty(t, readTestDir(t, rawDir))
			require.Empty(t, readTestDir(t, effectiveDir))
		})
	}
}

func TestOpenAIGatewayHandler_ChatAndEmbeddingsCanceledRequestsCleanSpools(t *testing.T) {
	for _, tt := range []struct {
		name  string
		route string
		body  func(string) []byte
	}{
		{"chat", "/v1/chat/completions", func(padding string) []byte {
			return []byte(`{"model":"alias-model","messages":[{"role":"user","content":"` + padding + `"}]}`)
		}},
		{"embeddings", "/v1/embeddings", func(padding string) []byte { return []byte(`{"model":"alias-model","input":"` + padding + `"}`) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rawDir, upstreamDir := t.TempDir(), t.TempDir()
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, TempDir: rawDir, FilePrefix: "sub2api-test-"}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
			t.Setenv("TMPDIR", upstreamDir)
			t.Setenv("TMP", upstreamDir)
			t.Setenv("TEMP", upstreamDir)

			ctx, cancel := context.WithCancel(context.Background())
			group := &service.Group{ID: 4, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
			account := &service.Account{ID: 41, Name: "canceled", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-canceled", "model_mapping": map[string]any{"alias-model": "mapped-model"}}}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, cancelingTerminalHTTPUpstream{cancel: cancel})
			router := env.router(tt.route, func(c *gin.Context) {
				if tt.route == "/v1/chat/completions" {
					env.handler.ChatCompletions(c)
					return
				}
				env.handler.Embeddings(c)
			})
			req := httptest.NewRequest(http.MethodPost, tt.route, bytes.NewReader(tt.body(strings.Repeat("x", 12<<20)))).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(httptest.NewRecorder(), req)

			require.ErrorIs(t, ctx.Err(), context.Canceled)
			require.Empty(t, readTestDir(t, rawDir))
			require.Empty(t, readTestDir(t, upstreamDir))
		})
	}
}

type openAIReplaySpoolUpstream struct {
	service.HTTPUpstream
	mu        sync.Mutex
	bodies    [][]byte
	lengths   []int64
	hashes    []string
	paths     []string
	started   chan struct{}
	release   chan struct{}
	snapshot  func() openAIReplaySnapshots
	snapshots chan openAIReplaySnapshots
}

type openAIStatusSpoolUpstream struct {
	service.HTTPUpstream

	status int
	body   []byte
}

type openAISpoolDeletingUpstream struct {
	service.HTTPUpstream

	dirs       []string
	accountIDs []int64
}

func (u *openAISpoolDeletingUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.accountIDs = append(u.accountIDs, accountID)
	defer func() { _ = req.Body.Close() }()
	for _, dir := range u.dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return nil, err
			}
		}
	}
	return nil, fmt.Errorf("read request body: %w", service.ErrRequestBodySpool)
}

func (u *openAISpoolDeletingUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func (u *openAIStatusSpoolUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	defer func() { _ = req.Body.Close() }()
	var err error
	u.body, err = io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: u.status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"upstream status"}}`)),
	}, nil
}

func (u *openAIStatusSpoolUpstream) DoWithTLS(req *http.Request, proxy string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxy, accountID, concurrency)
}

const openAIReplaySpoolWait = 5 * time.Second

type openAIReplaySnapshots struct {
	usageRequest  string
	usageUpstream string
	opsUpstream   string
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
	u.paths = append(u.paths, req.URL.Path)
	attempt := len(u.bodies)
	u.mu.Unlock()
	if attempt == 1 {
		return openAIReplayResponse(http.StatusTooManyRequests), nil
	}
	if u.snapshot != nil {
		select {
		case u.snapshots <- u.snapshot():
		case <-time.After(openAIReplaySpoolWait):
			return nil, errors.New("timed out publishing replay test snapshots")
		}
	}
	close(u.started)
	select {
	case <-u.release:
	case <-time.After(openAIReplaySpoolWait):
		return nil, errors.New("timed out waiting for replay test to release upstream")
	}
	return openAIReplayResponse(http.StatusOK, req.URL.Path), nil
}

func (u *openAIReplaySpoolUpstream) DoWithTLS(req *http.Request, proxy string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxy, accountID, concurrency)
}

func (u *openAIReplaySpoolUpstream) latestBody() []byte {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]byte(nil), u.bodies[len(u.bodies)-1]...)
}

func (u *openAIReplaySpoolUpstream) assert(t *testing.T, raw []byte, wantPath string) {
	t.Helper()
	u.mu.Lock()
	defer u.mu.Unlock()
	require.Len(t, u.bodies, 2)
	for i, body := range u.bodies {
		require.Equal(t, int64(len(body)), u.lengths[i])
		sum := sha256.Sum256(body)
		require.Equal(t, hex.EncodeToString(sum[:]), u.hashes[i])
		require.Contains(t, string(body), `"model":"mapped-model"`)
		require.Equal(t, wantPath, u.paths[i])
	}
	require.NotEqual(t, u.bodies[0], raw)
	require.Equal(t, u.bodies[0], u.bodies[1])
}

func openAIReplayResponse(status int, path ...string) *http.Response {
	if status == http.StatusTooManyRequests {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"retry"}}`))}
	}
	if len(path) != 0 && path[0] == "/v1/responses" {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_123\",\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"ok\"}]}],\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"))}
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"ok","model":"mapped-model","choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2},"data":[{"embedding":[0.1],"index":0}],"object":"list"}`))}
}

func waitOpenAIReplaySignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(openAIReplaySpoolWait):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitOpenAIReplaySnapshot(t *testing.T, snapshots <-chan openAIReplaySnapshots) openAIReplaySnapshots {
	t.Helper()
	select {
	case snapshot := <-snapshots:
		return snapshot
	case <-time.After(openAIReplaySpoolWait):
		t.Fatal("timed out waiting for blocked upstream usage and ops snapshots")
		return openAIReplaySnapshots{}
	}
}

func assertBoundedOpenAIReplaySnapshot(t *testing.T, name, snapshot string, body []byte) {
	t.Helper()
	require.Truef(t, len(snapshot) < len(body), "%s snapshot retained the complete body: %d bytes", name, len(snapshot))
	require.Equal(t, "request_body_preview", gjson.Get(snapshot, "kind").String(), "%s snapshot kind", name)
	require.Equal(t, int64(len(body)), gjson.Get(snapshot, "size").Int(), "%s snapshot size", name)
	require.Truef(t, gjson.Get(snapshot, "truncated").Bool(), "%s snapshot must be truncated", name)
	require.NotEmptyf(t, gjson.Get(snapshot, "preview").String(), "%s snapshot preview", name)
}

func readTestDir(t *testing.T, dir string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	return entries
}
