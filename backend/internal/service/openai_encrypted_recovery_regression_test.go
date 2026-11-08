package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestEncryptedRecoveryNestedReplay(t *testing.T) {
	body := []byte(`{"n":9007199254740993,"input":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"keep"},{"type":"reasoning","encrypted_content":"bad"},{"type":"reasoning","encrypted_content":"good"}]}]}`)
	original := string(body)
	digests := collectOpenAIEncryptedContentDigestsRaw(body)
	require.Len(t, digests, 2)
	invalid := map[string]struct{}{openAIEncryptedContentDigest("bad"): {}}
	next, n, err := stripOpenAIInvalidEncryptedContentRaw(body, invalid)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, original, string(body))
	require.Contains(t, string(next), "9007199254740993")
	require.NotContains(t, string(next), `"bad"`)
	require.Contains(t, string(next), `"good"`)
	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &decoded))
	var items []json.RawMessage
	require.NoError(t, json.Unmarshal(decoded["input"], &items))
	replay, count := stripOpenAIInvalidEncryptedContentFromReplayItems(items, invalid)
	require.Equal(t, 1, count)
	require.NotContains(t, string(replay[0]), `"bad"`)
	require.Contains(t, string(replay[0]), `"good"`)
}

type encryptedRecoveryLogSink chan string

func (s encryptedRecoveryLogSink) WriteLogEvent(event *logger.LogEvent) {
	select {
	case s <- event.Message:
	default:
	}
}

func TestEncryptedRecoveryGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, shape := range []string{"safe", "unsafe", "absent"} {
		t.Run(shape, func(t *testing.T) {
			safe := shape == "safe"
			var logs bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&logs)
			t.Cleanup(func() { log.SetOutput(previous) })
			sink := make(encryptedRecoveryLogSink, 100)
			logger.SetSink(sink)
			t.Cleanup(func() { logger.SetSink(nil) })
			upstream := &httpUpstreamSequenceRecorder{responses: []*http.Response{
				{StatusCode: 400, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"code":"invalid_encrypted_content","message":"rejected"}}`))},
				{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"id":"resp_ok","usage":{"input_tokens":1,"output_tokens":1}}`))},
			}}
			cfg := &config.Config{}
			cfg.Security.URLAllowlist.Enabled = false
			cfg.Security.URLAllowlist.AllowInsecureHTTP = true
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
			account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test", "base_url": "http://localhost:12345"}}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
			body := `{"model":"gpt-5.1","stream":false,"input":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"keep"},{"type":"reasoning","encrypted_content":"bad"}]}]}`
			if !safe {
				body = `{"model":"gpt-5.1","stream":false,"input":[{"type":"function_call_output","call_id":"c1","encrypted_content":"required"}]}`
			}
			if shape == "absent" {
				body = `{"model":"gpt-5.1","stream":false,"input":"plaintext-secret"}`
			}
			body = strings.ReplaceAll(body, `"bad"`, `"ciphertext-secret"`)
			_, err := svc.Forward(context.Background(), c, account, []byte(body))
			if safe {
				require.NoError(t, err)
				require.Equal(t, 2, upstream.callCount)
				require.False(t, gjson.GetBytes(upstream.bodies[1], "input.0.content.1").Exists())
			} else {
				require.Error(t, err)
				require.Equal(t, 1, upstream.callCount)
			}
			for len(sink) > 0 {
				logs.WriteString(<-sink)
			}
			expected := map[string]string{"safe": "changed=true retry=1", "unsafe": "reason=unsafe_carrier", "absent": "reason=no_safe_carrier"}[shape]
			require.Contains(t, logs.String(), expected)
			require.NotContains(t, logs.String(), "plaintext-secret")
			require.NotContains(t, logs.String(), "ciphertext-secret")
		})
	}
}

func TestEncryptedRecoveryUnsafeShapes(t *testing.T) {
	for _, input := range []string{
		`[{"type":"message","role":"assistant","content":[{"type":"reasoning","encrypted_content":"bad"}]}]`,
		`[{"type":"reasoning","encrypted_content":"bad"},{"type":"unknown","encrypted_content":"required"}]`,
		`[{"type":"message","content":[{"type":"output_text","text":"keep"},{"type":"unknown","encrypted_content":"required"}]}]`,
		`[{"type":"reasoning","encrypted_content":"bad","extra":{"encrypted_content":"required"}}]`,
		`[{"type":"compaction","encrypted_content":"bad","content":[{"type":"reasoning","encrypted_content":"good"}]}]`,
		`[{"type":"compaction_summary","encrypted_content":"good","content":[{"type":"reasoning","encrypted_content":"bad"}]}]`,
		`[{"type":"unknown","content":[{"type":"reasoning","content":null}]}]`,
	} {
		t.Run(input, func(t *testing.T) {
			body := []byte(`{"input":` + input + `}`)
			var decoded map[string]any
			require.NoError(t, decodeOpenAIJSONUseNumber(body, &decoded))
			require.False(t, trimOpenAIEncryptedReasoningItems(decoded))
			require.Empty(t, collectOpenAIEncryptedContentDigestsRaw(body))
			invalid := map[string]struct{}{openAIEncryptedContentDigest("bad"): {}}
			next, count, err := stripOpenAIInvalidEncryptedContentRaw(body, invalid)
			require.NoError(t, err)
			require.Zero(t, count)
			require.Equal(t, string(body), string(next))
			var items []json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(input), &items))
			replay, n := stripOpenAIInvalidEncryptedContentFromReplayItems(items, invalid)
			require.Zero(t, n)
			require.Equal(t, items, replay)
		})
	}
	_, _, err := stripOpenAIInvalidEncryptedContentRaw([]byte(`{"input":`), map[string]struct{}{"bad": {}})
	require.Error(t, err)
	untouched, count, err := stripOpenAIInvalidEncryptedContentRaw([]byte(`invalid`), nil)
	require.NoError(t, err)
	require.Zero(t, count)
	require.Equal(t, `invalid`, string(untouched))
}

func TestEncryptedRecoveryUnsafeTool(t *testing.T) {
	var body map[string]any
	require.NoError(t, decodeOpenAIJSONUseNumber([]byte(`{"input":[{"type":"reasoning","encrypted_content":"bad"},{"type":"function_call_output","call_id":"c1","encrypted_content":"required"}]}`), &body))
	before, _ := json.Marshal(body)
	require.False(t, trimOpenAIEncryptedReasoningItems(body))
	after, _ := json.Marshal(body)
	require.Equal(t, string(before), string(after))
}

func TestEncryptedRecoveryEmptyMessageBoundary(t *testing.T) {
	for _, part := range []string{`{"type":"reasoning","encrypted_content":null}`, `{"type":"reasoning","encrypted_content":""}`, `{"type":"reasoning","content":null}`} {
		t.Run(part, func(t *testing.T) {
			raw := []byte(`{"model":"gpt-5.1","stream":false,"n":9007199254740993,"input":[{"type":"message","role":"assistant","content":[` + part + `]}]}`)
			var body map[string]any
			require.NoError(t, decodeOpenAIJSONUseNumber(raw, &body))
			before, _ := json.Marshal(body)
			require.False(t, trimOpenAIEncryptedReasoningItems(body))
			after, _ := json.Marshal(body)
			require.Equal(t, string(before), string(after))
			invalid := map[string]struct{}{openAIEncryptedContentDigest("bad"): {}}
			next, n, err := stripOpenAIInvalidEncryptedContentRaw(raw, invalid)
			require.NoError(t, err)
			require.Zero(t, n)
			require.Equal(t, raw, next)
			items := []json.RawMessage{json.RawMessage(`{"type":"message","role":"assistant","content":[` + part + `]}`)}
			replay, n := stripOpenAIInvalidEncryptedContentFromReplayItems(items, invalid)
			require.Zero(t, n)
			require.Equal(t, items, replay)
			upstream := &httpUpstreamSequenceRecorder{responses: []*http.Response{
				{StatusCode: 400, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"code":"invalid_encrypted_content","message":"rejected"}}`))},
				{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"id":"should_not_retry"}`))},
			}}
			cfg := &config.Config{}
			cfg.Security.URLAllowlist.AllowInsecureHTTP = true
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
			account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test", "base_url": "http://localhost:12345"}}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
			_, err = svc.Forward(context.Background(), c, account, raw)
			require.Error(t, err)
			require.Equal(t, 1, upstream.callCount)
		})
	}
}

func TestEncryptedRecoveryLineageBoundaries(t *testing.T) {
	if runAllocationTestInSubprocess(t) {
		return
	}
	invalid := map[string]struct{}{openAIEncryptedContentDigest("bad"): {}}
	escaped := []byte(`{"n":9007199254740993,"input":[{"type":"reasoning","encrypted_\u0063ontent":"bad"}]}`)
	require.Equal(t, []string{openAIEncryptedContentDigest("bad")}, collectOpenAIEncryptedContentDigestsRaw(escaped))
	rewritten, n, err := stripOpenAIInvalidEncryptedContentRaw(escaped, invalid)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Contains(t, string(rewritten), "9007199254740993")
	require.Contains(t, string(escaped), `encrypted_\u0063ontent`)
	for _, raw := range []string{`{"input":[{"type":"reasoning","encrypted_content":"good"}]}`, `{"input":[{"type":"input_text","text":"keep"}]}`, `[]`, `null`} {
		body := []byte(raw)
		next, count, err := stripOpenAIInvalidEncryptedContentRaw(body, invalid)
		require.NoError(t, err)
		require.Zero(t, count)
		require.True(t, &body[0] == &next[0])
	}
	first := json.RawMessage(`{"type":"reasoning","encrypted_content":"bad"}`)
	for _, last := range []json.RawMessage{json.RawMessage(`{"type":"function_call_output","call_id":"c1","encrypted_content":"required"}`), json.RawMessage(`{"type":`)} {
		items := []json.RawMessage{first, last}
		before := string(first)
		next, count := stripOpenAIInvalidEncryptedContentFromReplayItems(items, invalid)
		require.Zero(t, count)
		require.True(t, &items[0] == &next[0])
		require.Equal(t, before, string(first))
	}
	items := []json.RawMessage{json.RawMessage(`{"type":"input_text","text":"prefix"}`), json.RawMessage(`{"type":"reasoning","encrypted_content":"good"}`)}
	next, count := stripOpenAIInvalidEncryptedContentFromReplayItems(items, invalid)
	require.Zero(t, count)
	require.True(t, &items[0] == &next[0])
	items = append(items, first, json.RawMessage(`{"type":"input_text","text":"suffix","n":9007199254740993}`))
	next, count = stripOpenAIInvalidEncryptedContentFromReplayItems(items, invalid)
	require.Equal(t, 1, count)
	require.Len(t, next, 3)
	require.Equal(t, items[0], next[0])
	require.Equal(t, items[1], next[1])
	require.Equal(t, items[3], next[2])
	require.Equal(t, `{"type":"reasoning","encrypted_content":"bad"}`, string(first))
	plain := map[string]any{"type": "input_text", "text": "keep"}
	decoded := map[string]any{"input": []any{map[string]any{"type": "message", "content": []any{plain}}}}
	require.Zero(t, testing.AllocsPerRun(10, func() {
		if stripOpenAIInvalidEncryptedContentItems(decoded, invalid) != 0 {
			panic("unexpected rewrite")
		}
	}))
	leaf := map[string]any{"type": "reasoning", "encrypted_content": "good"}
	parts := []any{plain, leaf}
	message := map[string]any{"type": "message", "content": parts}
	noMatch := func(string) bool { return false }
	leafAllocs := testing.AllocsPerRun(10, func() { sanitizeOpenAIEncryptedCarrier(leaf, noMatch) })
	messageAllocs := testing.AllocsPerRun(10, func() { sanitizeOpenAIEncryptedCarrier(message, noMatch) })
	require.Equal(t, leafAllocs, messageAllocs, "nested no-hit must not allocate an output slice beyond the digest work")
	require.True(t, &parts[0] == &message["content"].([]any)[0])
}
