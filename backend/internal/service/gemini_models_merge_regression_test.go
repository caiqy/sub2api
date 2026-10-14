//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGeminiModelsMergeActualGETPassthrough(t *testing.T) {
	for _, vertex := range []bool{false, true} {
		for _, detail := range []bool{false, true} {
			name := "AIStudio"
			if vertex {
				name = "Vertex"
			}
			if detail {
				name += "/detail"
			} else {
				name += "/list"
			}
			t.Run(name, func(t *testing.T) {
				upstream := &httpUpstreamRecorder{resp: vertexModelsJSONResponse(http.StatusOK, `{"publisherModels":[{"name":"publishers/google/models/gemini-3.8-flash"}],"models":[]}`)}
				svc := newGeminiModelsEndpointsTestService(nil, upstream)
				account := &Account{ID: 261, Platform: PlatformGemini, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "account-key"}}
				if vertex {
					account = vertexModelSyncTestAccount(nil)
				}
				account.Extra = map[string]any{
					"passthrough_fields_enabled": true,
					"passthrough_field_rules": []PassthroughFieldRule{
						{Target: "header", Mode: "inject", Key: "X-Env", Value: "prod"},
						{Target: "header", Mode: "forward", Key: "X-Trace-Id"},
					},
				}
				headers := http.Header{"X-Trace-Id": {"trace-merge"}}
				var err error
				if detail {
					_, err = svc.ForwardGetModel(context.Background(), account, "gemini-3.8-flash", headers)
				} else {
					_, err = svc.ForwardListModels(context.Background(), account, headers)
				}
				require.NoError(t, err)
				require.Equal(t, http.MethodGet, upstream.lastReq.Method)
				require.Equal(t, "prod", upstream.lastReq.Header.Get("X-Env"))
				require.Equal(t, "trace-merge", upstream.lastReq.Header.Get("X-Trace-Id"))
				if vertex {
					require.Equal(t, "Bearer ya29.vertex-token", upstream.lastReq.Header.Get("Authorization"))
				} else {
					require.Equal(t, "account-key", upstream.lastReq.Header.Get("x-goog-api-key"))
				}
			})
		}
	}
}

func TestGeminiMergeVertexHandleReplayAndOwnership(t *testing.T) {
	for _, native := range []bool{false, true} {
		name := "messages"
		body := []byte(`{"model":"gemini-3-flash-preview","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)
		if native {
			name = "native"
			body = []byte(`{"contents":[{"parts":[{"text":"hello"}],"role":"user"}]}`)
		}
		t.Run(name, func(t *testing.T) {
			handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
			require.NoError(t, err)
			defer CleanupRequestBodyHandle(handle)
			upstream := &httpUpstreamRecorder{resp: vertexModelsJSONResponse(http.StatusOK, geminiVertexAPIKeyTestResponse)}
			svc := &GeminiMessagesCompatService{httpUpstream: upstream, cfg: &config.Config{}}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
			account := geminiVertexAPIKeyTestAccount("")
			if native {
				_, err = svc.ForwardNativeHandle(context.Background(), c, account, "gemini-3-flash-preview", "generateContent", false, handle)
			} else {
				_, err = svc.ForwardHandle(context.Background(), c, account, handle)
			}
			require.NoError(t, err)
			requireVertexAPIKeyAuth(t, upstream.lastReq)
			require.Contains(t, upstream.lastReq.URL.Path, "/v1/publishers/google/models/gemini-3-flash-preview:generateContent")
			require.Len(t, upstream.replays, 1)
			require.Equal(t, upstream.lastBody, upstream.replays[0])
			require.FileExists(t, handle.spoolPath)
			replayed, err := handle.ReadAll()
			require.NoError(t, err)
			require.Equal(t, body, replayed)
			require.Zero(t, handle.spoolReaders)
		})
	}
}

func TestGeminiMergeVertexAuthFailureClosesReader(t *testing.T) {
	original := openGeminiMessagesCompatRequestBody
	defer func() { openGeminiMessagesCompatRequestBody = original }()
	reader := &passthroughCloseTrackingReadCloser{Reader: strings.NewReader(`{}`)}
	openGeminiMessagesCompatRequestBody = func(*RequestBodyHandle) (io.ReadCloser, error) { return reader, nil }
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	handle, err := NewRequestBodyHandleFromBytes([]byte(`{"model":"gemini-3-flash-preview","messages":[{"role":"user","content":"hello"}]}`), RequestBodyHandleOptions{})
	require.NoError(t, err)
	defer CleanupRequestBodyHandle(handle)
	upstream := &httpUpstreamRecorder{}
	svc := &GeminiMessagesCompatService{httpUpstream: upstream, cfg: &config.Config{}}
	account := geminiVertexAPIKeyTestAccount("")
	account.Credentials["api_key"] = ""
	_, err = svc.ForwardHandle(context.Background(), c, account, handle)
	require.Error(t, err)
	require.True(t, reader.closed)
	require.Empty(t, upstream.requests)
}

func TestGeminiModelsMergePassthroughFailureStopsRequest(t *testing.T) {
	original := geminiApplyAccountPassthroughFieldsWithContext
	defer func() { geminiApplyAccountPassthroughFieldsWithContext = original }()
	geminiApplyAccountPassthroughFieldsWithContext = func(context.Context, *Account, http.Header, []byte, []byte, http.Header) ([]byte, error) {
		return nil, context.Canceled
	}
	upstream := &httpUpstreamRecorder{}
	svc := newGeminiModelsEndpointsTestService(nil, upstream)
	account := &Account{Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "account-key"}}
	_, err := svc.ForwardListModels(context.Background(), account, http.Header{})
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, upstream.requests)
}
