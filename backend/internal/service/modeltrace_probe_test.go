package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type modelTraceTransport struct {
	HTTPUpstream
	request   *http.Request
	body      map[string]any
	proxy     string
	accountID int64
	status    int
	calls     int
}

func (u *modelTraceTransport) DoWithTLS(r *http.Request, proxy string, id int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	u.calls++
	u.request, u.proxy, u.accountID = r, proxy, id
	if err := json.NewDecoder(r.Body).Decode(&u.body); err != nil {
		return nil, err
	}
	output := `data: {"type":"response.output_text.delta","delta":"1,2,3"}` + "\n\n" + `data: {"type":"response.completed","response":{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"1,2,3"}]}]}}` + "\n\n"
	if strings.HasSuffix(r.URL.Path, "/chat/completions") {
		output = "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"1,2,3\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
	}
	status := u.status
	if status == 0 {
		status = 200
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(output))}, nil
}

type modelTraceCredentialRepo struct {
	AccountRepository
	accounts map[int64]*Account
}

func (r *modelTraceCredentialRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	if account := r.accounts[id]; account != nil {
		return account, nil
	}
	return nil, ErrAccountNotFound
}

func TestModelTraceProbeProtocolsAndSol(t *testing.T) {
	require.Contains(t, modeltrace.Models(), "gpt-5.6-sol")
	for _, kind := range []string{AccountTypeOAuth, AccountTypeSetupToken, AccountTypeAPIKey, "chat"} {
		t.Run(kind, func(t *testing.T) {
			u := &modelTraceTransport{}
			account := &Account{ID: 19, Platform: PlatformOpenAI, Type: kind, Credentials: map[string]any{"access_token": "secret", "api_key": "secret", "base_url": "https://example.com/v1"}, Extra: map[string]any{}}
			if kind == "chat" {
				account.Type = AccountTypeAPIKey
				account.Extra[openai_compat.ExtraKeyResponsesSupported] = false
			}
			repo := &modelTraceCredentialRepo{accounts: map[int64]*Account{19: account}}
			s := &AccountTestService{accountRepo: repo, httpUpstream: u, cfg: &config.Config{}}
			resolved, target, err := s.ResolveModelTraceTarget(context.Background(), 19, "gpt-5.6-sol")
			require.NoError(t, err)
			require.Equal(t, "gpt-5.6-sol", target)
			output, err := s.ProbeModelTrace(context.Background(), resolved, target, modeltrace.Challenge{Prompt: "unique challenge", ExpectedCount: 300})
			require.NoError(t, err)
			require.Equal(t, "1,2,3", output)
			require.Equal(t, 1, u.calls)
			require.Equal(t, "gpt-5.6-sol", u.body["model"])
			require.Equal(t, "Bearer secret", u.request.Header.Get("Authorization"))
			require.Equal(t, int64(19), u.accountID)
			raw, _ := json.Marshal(u.body)
			require.Contains(t, string(raw), "unique challenge")
			if kind == "chat" {
				require.Equal(t, "/v1/chat/completions", u.request.URL.Path)
			} else {
				require.True(t, strings.HasSuffix(u.request.URL.Path, "/responses"))
			}
		})
	}
}

func TestModelTraceProbeShadowCredentialsAndNoRetry(t *testing.T) {
	parentID, proxyID := int64(31), int64(5)
	parent := &Account{ID: parentID, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "parent-secret"}}
	shadow := &Account{ID: 32, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ParentAccountID: &parentID, ProxyID: &proxyID, Proxy: &Proxy{Protocol: "http", Host: "proxy.example", Port: 8080}, Credentials: map[string]any{"model_mapping": map[string]any{"gpt-6-astra": "gpt-5.6-sol"}}}
	u := &modelTraceTransport{status: 401}
	s := &AccountTestService{accountRepo: &modelTraceCredentialRepo{accounts: map[int64]*Account{31: parent, 32: shadow}}, httpUpstream: u}
	account, target, err := s.ResolveModelTraceTarget(context.Background(), 32, "gpt-6-astra")
	require.NoError(t, err)
	require.Equal(t, "gpt-6-astra", target)
	_, err = s.ProbeModelTrace(context.Background(), account, target, modeltrace.Challenge{Prompt: "probe"})
	require.Error(t, err)
	require.Equal(t, 1, u.calls)
	require.Equal(t, "Bearer parent-secret", u.request.Header.Get("Authorization"))
	require.Equal(t, shadow.Proxy.URL(), u.proxy)
	require.Equal(t, shadow.ID, u.accountID)
	// Embedded repository methods panic if the probe tries SetError or cooldown mutation.
	shadow.Credentials = map[string]any{"model_mapping": map[string]any{"gpt-6-astra": "unsupported-model"}}
	_, target, err = s.ResolveModelTraceTarget(context.Background(), 32, "gpt-6-astra")
	require.NoError(t, err)
	require.Equal(t, "gpt-6-astra", target)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.ProbeModelTrace(ctx, account, target, modeltrace.Challenge{Prompt: "probe"})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, u.calls)
}

func TestModelTraceProbePassthroughAndCapability(t *testing.T) {
	account := &Account{ID: 41, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"api_key": "secret", "model_mapping": map[string]any{"gpt-5.6-sol": "unsupported-model"},
	}, Extra: map[string]any{"openai_passthrough": true}}
	u := &modelTraceTransport{}
	s := &AccountTestService{accountRepo: &modelTraceCredentialRepo{accounts: map[int64]*Account{41: account}}, httpUpstream: u, cfg: &config.Config{}}
	resolved, target, err := s.ResolveModelTraceTarget(context.Background(), 41, "gpt-5.6-sol")
	require.NoError(t, err)
	require.Equal(t, "gpt-5.6-sol", target)
	_, err = s.ProbeModelTrace(context.Background(), resolved, target, modeltrace.Challenge{Prompt: "probe"})
	require.NoError(t, err)
	require.Equal(t, target, u.body["model"])
	require.True(t, HTTPUpstreamRedirectsDisabled(u.request.Context()))
	account.Credentials["model_mapping"] = map[string]any{"unlisted-alias": "gpt-5.6-sol"}
	_, _, err = s.ResolveModelTraceTarget(context.Background(), 41, "unlisted-alias")
	require.Error(t, err)

	account.Credentials["openai_capabilities"] = []any{"embeddings"}
	_, _, err = s.ResolveModelTraceTarget(context.Background(), 41, "gpt-5.6-sol")
	require.Error(t, err)
	_, err = s.ProbeModelTrace(context.Background(), account, target, modeltrace.Challenge{Prompt: "probe"})
	require.Error(t, err)
	require.Equal(t, 1, u.calls)
}

func TestModelTraceProbeRedirectsNeverReplay(t *testing.T) {
	account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret"}}
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			u := &modelTraceTransport{status: status}
			s := &AccountTestService{accountRepo: &modelTraceCredentialRepo{accounts: map[int64]*Account{42: account}}, httpUpstream: u, cfg: &config.Config{}}
			_, err := s.ProbeModelTrace(context.Background(), account, "gpt-5.6-sol", modeltrace.Challenge{Prompt: "probe"})
			require.Error(t, err)
			require.Equal(t, 1, u.calls)
			require.True(t, HTTPUpstreamRedirectsDisabled(u.request.Context()))
		})
	}
}

func TestModelTraceOutputRequiresSuccessfulCompletion(t *testing.T) {
	cases := []struct {
		name, body, want   string
		chat, unary, valid bool
	}{
		{"responses snapshot beats delta", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"9,9,9\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}]}}\n\n", "1,2,3", false, false, true},
		{"responses snapshot", `data: {"type":"response.completed","response":{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"1,2,3"}]}]}}`, "1,2,3", false, false, true},
		{"responses done", `data: {"type":"response.done","response":{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"1,2,3"}]}]}}`, "1,2,3", false, false, true},
		{"responses multiple messages", `data: {"type":"response.done","response":{"status":"completed","output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"999"}]},{"type":"message","content":[{"type":"output_text","text":"1,2"}]},{"type":"message","content":[{"type":"output_text","text":"3,4"}]}]}}`, "1,2\n3,4", false, false, true},
		{"completed items without terminal output", "data: {\"type\":\"response.output_item.done\",\"output_index\":2,\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"3,4\"}]}}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2\"}]}}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":1,\"item\":{\"type\":\"reasoning\",\"content\":[{\"type\":\"output_text\",\"text\":\"999\"}]}}\n\n" +
			"data: {\"type\":\"response.done\",\"response\":{\"status\":\"completed\"}}\n\n", "1,2\n3,4", false, false, true},
		{"completed item replaces delta", "data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"type\":\"message\"}}\n\n" +
			"data: {\"type\":\"response.output_text.delta\",\"output_index\":0,\"delta\":\"1,2\"}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"message\",\"status\":\"completed\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n", "1,2,3", false, false, true},
		{"snapshot wins completed items", "data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"9,9,9\"}]}}\n\n" +
			"data: {\"type\":\"response.done\",\"response\":{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}]}}\n\n", "1,2,3", false, false, true},
		{"unfinished added message", "data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"type\":\"message\"}}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":1,\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n", "", false, false, false},
		{"unfinished delta message", "data: {\"type\":\"response.output_text.delta\",\"output_index\":0,\"delta\":\"9,9,9\"}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":1,\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}}\n\n" +
			"data: {\"type\":\"response.done\",\"response\":{\"status\":\"completed\"}}\n\n", "", false, false, false},
		{"unindexed delta cannot establish completion", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"9,9,9\"}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}}\n\n" +
			"data: {\"type\":\"response.done\",\"response\":{\"status\":\"completed\"}}\n\n", "", false, false, false},
		{"empty snapshot uses completed item", "data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}}\n\n" +
			"data: {\"type\":\"response.done\",\"response\":{\"status\":\"completed\",\"output\":[]}}\n\n", "1,2,3", false, false, true},
		{"null snapshot uses completed item", "data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"output\":null}}\n\n", "1,2,3", false, false, true},
		{"empty snapshot without completed item", `data: {"type":"response.done","response":{"status":"completed","output":[]}}`, "", false, false, false},
		{"empty snapshot with unfinished message", "data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"type\":\"message\"}}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":1,\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}}\n\n" +
			"data: {\"type\":\"response.done\",\"response\":{\"status\":\"completed\",\"output\":[]}}\n\n", "", false, false, false},
		{"incomplete item cannot be scored", "data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"message\",\"status\":\"incomplete\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}}\n\n" +
			"data: {\"type\":\"response.done\",\"response\":{\"status\":\"completed\"}}\n\n", "", false, false, false},
		{"unary multiple messages", `{"status":"completed","output":[{"type":"reasoning","content":[{"type":"output_text","text":"999"}]},{"type":"message","content":[{"type":"output_text","text":"1,"},{"type":"output_text","text":"2"}]},{"type":"message","content":[{"type":"output_text","text":"3,4"}]}]}`, "1,2\n3,4", false, true, true},
		{"responses delta without snapshot", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"1,2,3\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n", "", false, false, false},
		{"responses truncated", `data: {"type":"response.output_text.delta","delta":"1,2,3"}`, "", false, false, false},
		{"responses failed", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"1,2,3\"}\n\ndata: {\"type\":\"response.failed\"}\n\n", "", false, false, false},
		{"responses done failed", `data: {"type":"response.done","response":{"status":"failed","output":[{"type":"message","content":[{"type":"output_text","text":"1,2,3"}]}]}}`, "", false, false, false},
		{"responses done incomplete", `data: {"type":"response.done","response":{"status":"incomplete","output":[{"type":"message","content":[{"type":"output_text","text":"1,2,3"}]}]}}`, "", false, false, false},
		{"chat stopped", "data: {\"choices\":[{\"delta\":{\"content\":\"1,2,3\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n", "1,2,3", true, false, true},
		{"chat length", "data: {\"choices\":[{\"delta\":{\"content\":\"1,2,3\"},\"finish_reason\":\"length\"}]}\n\ndata: [DONE]\n\n", "", true, false, false},
		{"chat missing finish", "data: {\"choices\":[{\"delta\":{\"content\":\"1,2,3\"}}]}\n\ndata: [DONE]\n\n", "", true, false, false},
		{"unary chat", `{"choices":[{"message":{"content":"1,2,3"},"finish_reason":"stop"}]}`, "1,2,3", true, true, true},
		{"unary incomplete", `{"status":"incomplete","output":[{"type":"message","content":[{"type":"output_text","text":"1,2,3"}]}]}`, "", false, true, false},
		{"error data", `data: {"error":{"message":"secret"}}`, "", false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := readModelTraceOutput(strings.NewReader(tc.body), tc.chat, tc.unary)
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, tc.want, out)
			} else {
				require.Error(t, err)
				require.Empty(t, out)
				require.NotContains(t, err.Error(), "secret")
			}
		})
	}
}

func TestModelTraceOutputLimits(t *testing.T) {
	tooLong := strings.Repeat("1", (1<<20)+1)
	for _, tc := range []struct {
		name, body string
		unary      bool
	}{
		{"unary output", fmt.Sprintf(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"%s"}]}]}`, tooLong), true},
		{"stream snapshot", fmt.Sprintf("data: {\"type\":\"response.done\",\"response\":{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"%s\"}]}]}}\n\n", tooLong), false},
		{"oversized snapshot cannot fall back to completed item", fmt.Sprintf("data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"1,2,3\"}]}}\n\n"+
			"data: {\"type\":\"response.done\",\"response\":{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"%s\"}]}]}}\n\n", tooLong), false},
		{"body", strings.Repeat(" ", (8<<20)+1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readModelTraceOutput(strings.NewReader(tc.body), false, tc.unary)
			require.Error(t, err)
		})
	}
}
