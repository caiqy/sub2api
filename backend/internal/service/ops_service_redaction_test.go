package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsSensitiveKey_TokenBudgetKeysNotRedacted(t *testing.T) {
	t.Parallel()

	for _, key := range []string{
		"max_tokens",
		"max_output_tokens",
		"max_input_tokens",
		"max_completion_tokens",
		"max_tokens_to_sample",
		"budget_tokens",
		"prompt_tokens",
		"completion_tokens",
		"input_tokens",
		"output_tokens",
		"total_tokens",
		"token_count",
	} {
		if isSensitiveKey(key) {
			t.Fatalf("expected key %q to NOT be treated as sensitive", key)
		}
	}

	for _, key := range []string{
		"authorization",
		"Authorization",
		"access_token",
		"refresh_token",
		"id_token",
		"session_token",
		"token",
		"client_secret",
		"private_key",
		"signature",
	} {
		if !isSensitiveKey(key) {
			t.Fatalf("expected key %q to be treated as sensitive", key)
		}
	}
}

func TestSanitizeAndTrimJSONPayload_PreservesTokenBudgetFields(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"model":"claude-3","max_tokens":123,"thinking":{"type":"enabled","budget_tokens":456},"access_token":"abc","messages":[{"role":"user","content":"hi"}]}`)
	out, _, _ := sanitizeAndTrimJSONPayload(raw, 10*1024)
	if out == "" {
		t.Fatalf("expected non-empty sanitized output")
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("unmarshal sanitized output: %v", err)
	}

	if got, ok := decoded["max_tokens"].(float64); !ok || got != 123 {
		t.Fatalf("expected max_tokens=123, got %#v", decoded["max_tokens"])
	}

	thinking, ok := decoded["thinking"].(map[string]any)
	if !ok || thinking == nil {
		t.Fatalf("expected thinking object to be preserved, got %#v", decoded["thinking"])
	}
	if got, ok := thinking["budget_tokens"].(float64); !ok || got != 456 {
		t.Fatalf("expected thinking.budget_tokens=456, got %#v", thinking["budget_tokens"])
	}

	if got := decoded["access_token"]; got != "[REDACTED]" {
		t.Fatalf("expected access_token to be redacted, got %#v", got)
	}
}

func TestShrinkToEssentials_IncludesThinking(t *testing.T) {
	t.Parallel()

	root := map[string]any{
		"model":      "claude-3",
		"max_tokens": 100,
		"thinking": map[string]any{
			"type":          "enabled",
			"budget_tokens": 200,
		},
		"messages": []any{
			map[string]any{"role": "user", "content": "first"},
			map[string]any{"role": "user", "content": "last"},
		},
	}

	out := shrinkToEssentials(root)
	if _, ok := out["thinking"]; !ok {
		t.Fatalf("expected thinking to be included in essentials: %#v", out)
	}
}

func TestSanitizeOpsUpstreamErrorsBoundsAndSanitizesEveryEventBody(t *testing.T) {
	const secret = "c2VjcmV0LW9wcy1pbWFnZQ=="
	largeText := strings.Repeat("x", opsMaxStoredErrorBodyBytes+1024)
	events := make([]*OpsUpstreamErrorEvent, 32)
	for i := range events {
		events[i] = &OpsUpstreamErrorEvent{
			UpstreamStatusCode:   500,
			UpstreamRequestBody:  `{"input":"data:image/png;base64,` + secret + `","padding":"` + largeText + `"}`,
			UpstreamResponseBody: `{"error":"data:image/png;base64,` + secret + `","padding":"` + largeText + `"}`,
		}
	}
	entry := &OpsInsertErrorLogInput{UpstreamErrors: events}

	require.NoError(t, sanitizeOpsUpstreamErrors(entry))
	require.NotNil(t, entry.UpstreamErrorsJSON)
	stored, err := ParseOpsUpstreamErrors(*entry.UpstreamErrorsJSON)
	require.NoError(t, err)
	require.Len(t, stored, 32)
	for _, event := range stored {
		require.LessOrEqual(t, len(event.UpstreamRequestBody), opsMaxStoredErrorBodyBytes)
		require.LessOrEqual(t, len(event.UpstreamResponseBody), opsMaxStoredErrorBodyBytes)
		require.NotContains(t, event.UpstreamRequestBody, secret)
		require.NotContains(t, event.UpstreamResponseBody, secret)
	}
}

func TestSanitizeOpsUpstreamErrorsPreservesPlainResponseBody(t *testing.T) {
	const payload = "c2VjcmV0LWltYWdl"
	tests := []struct {
		name     string
		response string
		want     string
		omitted  bool
	}{
		{name: "plain overloaded", response: "upstream overloaded, retry later"},
		{name: "BOM plain overloaded", response: "\ufeffupstream overloaded, retry later", want: "upstream overloaded, retry later"},
		{name: "html", response: `<html><body>upstream unavailable</body></html>`},
		{name: "payload-less data URL mention", response: "invalid prefix data:image/png;base64,"},
		{name: "small b64_json", response: `{"error":{"b64_json":"` + payload + `"}}`, omitted: true},
		{name: "error data URL", response: `{"error":"data:image/png;base64,` + payload + `"}`, omitted: true},
		{name: "large inline data", response: `{"inlineData":{"data":"` + payload + `"},"padding":"` + strings.Repeat("x", opsMaxStoredErrorBodyBytes) + `"}`, omitted: true},
		{name: "plain data URL", response: "failed with data:image/png;base64," + payload, omitted: true},
		{name: "truncated b64_json string", response: `{"error":{"b64_json":"` + payload, omitted: true},
		{name: "truncated inlineData string", response: `{"inlineData":{"data":"` + payload, omitted: true},
		{name: "truncated source data string", response: `{"source":{"type":"base64","data":"` + payload, omitted: true},
		{name: "ordinary malformed JSON", response: `{"error":"overloaded"`, omitted: true},
		{name: "BOM b64_json", response: "\ufeff" + `{"error":{"b64_json":"` + payload + `"}}`, omitted: true},
		{name: "BOM inlineData", response: "\ufeff" + `{"inlineData":{"data":"` + payload + `"}}`, omitted: true},
		{name: "BOM source", response: "\ufeff" + `{"source":{"type":"base64","data":"` + payload + `"}}`, omitted: true},
		{name: "BOM truncated b64_json", response: "\ufeff" + `{"error":{"b64_json":"` + payload, omitted: true},
		{name: "BOM truncated inlineData", response: "\ufeff" + `{"inlineData":{"data":"` + payload, omitted: true},
		{name: "BOM truncated source", response: "\ufeff" + `{"source":{"type":"base64","data":"` + payload, omitted: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &OpsInsertErrorLogInput{UpstreamErrors: []*OpsUpstreamErrorEvent{{
				UpstreamStatusCode:   502,
				UpstreamRequestBody:  RequestBodyPreviewSnapshot(`{"input":"hello"}`, 17),
				UpstreamResponseBody: tt.response,
			}}}

			require.NoError(t, sanitizeOpsUpstreamErrors(entry))
			stored, err := ParseOpsUpstreamErrors(*entry.UpstreamErrorsJSON)
			require.NoError(t, err)
			require.LessOrEqual(t, len(stored[0].UpstreamResponseBody), opsMaxStoredErrorBodyBytes)
			if tt.omitted {
				require.Equal(t, requestBodyPreviewOmittedMarker, stored[0].UpstreamResponseBody)
				require.NotContains(t, stored[0].UpstreamResponseBody, payload)
			} else {
				want := tt.response
				if tt.want != "" {
					want = tt.want
				}
				require.Equal(t, want, stored[0].UpstreamResponseBody)
			}
		})
	}
}

func TestSanitizeRequestBodyForStoragePreservesPreviewWrapper(t *testing.T) {
	t.Run("large preview remains a valid bounded wrapper", func(t *testing.T) {
		raw := RequestBodyPreviewSnapshot(strings.Repeat("x", 30*1024), 50*1024)

		stored, truncated := sanitizeRequestBodyForStorage(raw, opsMaxStoredErrorBodyBytes)

		require.True(t, truncated)
		require.LessOrEqual(t, len(stored), opsMaxStoredErrorBodyBytes)
		snapshot, ok := parseRequestBodyPreviewSnapshot(stored)
		require.True(t, ok)
		require.Equal(t, int64(50*1024), snapshot.Size)
		require.True(t, snapshot.Truncated)
		require.NotEmpty(t, snapshot.Preview)
	})

	t.Run("inner preview is sanitized without losing wrapper", func(t *testing.T) {
		rawBytes, err := json.Marshal(requestBodyPreviewSnapshot{
			Kind:    requestBodyPreviewSnapshotKind,
			Preview: `{"image_url":"data:image/png;base64,c2VjcmV0"}`,
			Size:    123,
		})
		require.NoError(t, err)

		stored, truncated := sanitizeRequestBodyForStorage(string(rawBytes), opsMaxStoredErrorBodyBytes)

		require.True(t, truncated)
		snapshot, ok := parseRequestBodyPreviewSnapshot(stored)
		require.True(t, ok)
		require.Equal(t, int64(123), snapshot.Size)
		require.True(t, snapshot.Truncated)
		require.Contains(t, snapshot.Preview, "omitted")
		require.NotContains(t, stored, "c2VjcmV0")
	})
}

func TestParseRequestBodyPreviewSnapshotValidatesFieldTypes(t *testing.T) {
	for _, raw := range []string{
		`{"kind":"request_body_preview","preview":1,"truncated":false,"size":1}`,
		`{"kind":"request_body_preview","preview":"ok","truncated":"false","size":1}`,
		`{"kind":"request_body_preview","preview":"ok","truncated":false,"size":"1"}`,
	} {
		_, ok := parseRequestBodyPreviewSnapshot(raw)
		require.False(t, ok, raw)
	}
}

func TestSanitizeOpsUpstreamErrorsPreservesBoundedPreviewWrappers(t *testing.T) {
	requestWrapper := RequestBodyPreviewSnapshot(strings.Repeat("r", 30*1024), 60*1024)
	entry := &OpsInsertErrorLogInput{UpstreamErrors: []*OpsUpstreamErrorEvent{{
		UpstreamStatusCode:   500,
		UpstreamRequestBody:  requestWrapper,
		UpstreamResponseBody: "upstream failed",
	}}}

	require.NoError(t, sanitizeOpsUpstreamErrors(entry))
	stored, err := ParseOpsUpstreamErrors(*entry.UpstreamErrorsJSON)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	require.LessOrEqual(t, len(stored[0].UpstreamRequestBody), opsMaxStoredErrorBodyBytes)
	snapshot, ok := parseRequestBodyPreviewSnapshot(stored[0].UpstreamRequestBody)
	require.True(t, ok)
	require.Equal(t, requestBodyPreviewSnapshotKind, snapshot.Kind)
	require.True(t, snapshot.Truncated)
	require.NotEmpty(t, snapshot.Preview)
	require.Equal(t, int64(60*1024), mustParseRequestBodyPreviewSnapshot(t, stored[0].UpstreamRequestBody).Size)
	require.Equal(t, "upstream failed", stored[0].UpstreamResponseBody)
}

func mustParseRequestBodyPreviewSnapshot(t *testing.T, raw string) requestBodyPreviewSnapshot {
	t.Helper()
	snapshot, ok := parseRequestBodyPreviewSnapshot(raw)
	require.True(t, ok)
	return snapshot
}
