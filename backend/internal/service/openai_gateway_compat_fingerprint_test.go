package service

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type oauthCompatFingerprintEntry struct {
	name    string
	path    string
	body    []byte
	forward func(*OpenAIGatewayService, *gin.Context, *Account, []byte) error
}

func TestOAuthCompatPaths_ConvergeCodexFingerprintModes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	modes := []struct {
		name     string
		extra    map[string]any
		expected codexFingerprintMode
	}{
		{name: "missing defaults to off", expected: codexFingerprintOff},
		{name: "empty defaults to off", extra: map[string]any{codexFingerprintModeExtraKey: ""}, expected: codexFingerprintOff},
		{name: "invalid defaults to off", extra: map[string]any{codexFingerprintModeExtraKey: "invalid"}, expected: codexFingerprintOff},
		{name: "off preserves client values", extra: map[string]any{codexFingerprintModeExtraKey: "off"}, expected: codexFingerprintOff},
		{name: "device converges installation", extra: map[string]any{codexFingerprintModeExtraKey: "device"}, expected: codexFingerprintDevice},
		{name: "full converges all identifiers", extra: map[string]any{codexFingerprintModeExtraKey: "full"}, expected: codexFingerprintFull},
	}

	for _, entry := range oauthCompatFingerprintEntries() {
		for _, mode := range modes {
			t.Run(entry.name+"/"+mode.name, func(t *testing.T) {
				c := newOAuthCompatFingerprintContext(entry.path, entry.body)
				upstream := &httpUpstreamRecorder{err: errors.New("upstream unavailable")}
				svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
				account := newOAuthCompatFingerprintAccount(4101, mode.extra)

				require.Error(t, entry.forward(svc, c, account, entry.body))
				require.Len(t, upstream.requests, 1)
				require.Equal(t, mode.expected, account.GetCodexFingerprintMode())
				requireOAuthCompatFingerprintAttempt(t, entry.name, account, mode.expected, upstream.lastReq, upstream.lastBody)
			})
		}
	}
}

func TestOAuthCompatPaths_ClearFingerprintOnOffFailoverReentry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, entry := range oauthCompatFingerprintEntries() {
		t.Run(entry.name, func(t *testing.T) {
			c := newOAuthCompatFingerprintContext(entry.path, entry.body)
			upstream := &httpUpstreamRecorder{err: errors.New("upstream unavailable")}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}

			require.Error(t, entry.forward(svc, c, newOAuthCompatFingerprintAccount(4102, map[string]any{codexFingerprintModeExtraKey: "session"}), entry.body))
			offAccount := newOAuthCompatFingerprintAccount(4103, map[string]any{codexFingerprintModeExtraKey: "off"})
			require.Error(t, entry.forward(svc, c, offAccount, entry.body))
			require.Len(t, upstream.requests, 2)

			ids, exists := c.Get("codex_fingerprint_ids")
			require.True(t, exists)
			require.Nil(t, ids)
			requireOAuthCompatFingerprintAttempt(t, entry.name, offAccount, codexFingerprintOff, upstream.lastReq, upstream.lastBody)
		})
	}
}

func oauthCompatFingerprintEntries() []oauthCompatFingerprintEntry {
	return []oauthCompatFingerprintEntry{
		{
			name: "chat completions",
			path: "/v1/chat/completions",
			body: []byte(`{"model":"gpt-5.1","stream":false,"input":"hello","client_metadata":{"x-codex-installation-id":"client-installation","session_id":"client-session-underscore","thread_id":"client-thread","turn_id":"client-turn","x-codex-window-id":"client-window"}}`),
			forward: func(svc *OpenAIGatewayService, c *gin.Context, account *Account, body []byte) error {
				_, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
				return err
			},
		},
		{
			name: "messages",
			path: "/v1/messages",
			body: []byte(`{"model":"gpt-5.1","max_tokens":32,"messages":[{"role":"user","content":"hello"}],"stream":false}`),
			forward: func(svc *OpenAIGatewayService, c *gin.Context, account *Account, body []byte) error {
				_, err := svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "")
				return err
			},
		},
	}
}

func newOAuthCompatFingerprintContext(path string, body []byte) *gin.Context {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	c.Request.Header.Set("content-type", "application/json")
	c.Request.Header.Set("session-id", "client-session-hyphen")
	c.Request.Header.Set("session_id", "client-session-underscore")
	c.Request.Header.Set("x-codex-installation-id", "client-installation")
	c.Request.Header.Set("x-codex-turn-metadata", `{"installation_id":"client-installation","session_id":"client-session-underscore","thread_id":"client-thread","turn_id":"client-turn","window_id":"client-window"}`)
	return c
}

func newOAuthCompatFingerprintAccount(id int64, extra map[string]any) *Account {
	account := newTestOAuthAccount(id, extra)
	account.Concurrency = 1
	account.Credentials = map[string]any{"access_token": "oauth-token"}
	return account
}

func requireOAuthCompatFingerprintAttempt(t *testing.T, entry string, account *Account, mode codexFingerprintMode, req *http.Request, body []byte) {
	t.Helper()
	require.NotNil(t, req)

	metadata := gjson.GetBytes(body, "client_metadata")
	if mode == codexFingerprintOff {
		require.Equal(t, "client-session-hyphen", req.Header.Get("session-id"))
		require.Equal(t, "client-session-underscore", req.Header.Get("session_id"))
		if entry == "chat completions" {
			require.Equal(t, "client-installation", metadata.Get("x-codex-installation-id").String())
			require.Equal(t, "client-session-underscore", metadata.Get("session_id").String())
		} else {
			require.False(t, metadata.Exists())
		}
		return
	}

	require.True(t, metadata.Exists())
	require.NotEqual(t, "client-installation", req.Header.Get("x-codex-installation-id"))
	require.Equal(t, req.Header.Get("x-codex-installation-id"), metadata.Get("x-codex-installation-id").String())
	if mode == codexFingerprintDevice {
		return
	}

	require.Equal(t, req.Header.Get("session_id"), req.Header.Get("session-id"))
	require.NotEqual(t, "client-session-underscore", req.Header.Get("session_id"))
	require.Equal(t, req.Header.Get("session_id"), metadata.Get("session_id").String())
	require.Equal(t, req.Header.Get("thread-id"), metadata.Get("thread_id").String())
	require.Equal(t, req.Header.Get("x-codex-window-id"), metadata.Get("x-codex-window-id").String())

	turnMetadata := gjson.Parse(req.Header.Get("x-codex-turn-metadata"))
	require.True(t, turnMetadata.Exists())
	require.NotEqual(t, "client-turn", turnMetadata.Get("turn_id").String())
	require.Equal(t, turnMetadata.Get("turn_id").String(), metadata.Get("turn_id").String())

	if mode == codexFingerprintSession {
		require.Equal(t, resolveConvergedThreadID(account, "client-session-hyphen"), req.Header.Get("thread-id"))
	}
	if mode == codexFingerprintFull {
		require.Equal(t, resolveConvergedSessionID(account), req.Header.Get("thread-id"))
	}
}
