package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func TestNormalizeAccountPassthroughFields(t *testing.T) {
	t.Run("RejectsNonAPIKeyAccount", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType:             AccountTypeOAuth,
			ExplicitlySubmittedConfig: true,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
			},
		})

		require.EqualError(t, err, "passthrough field rules are only supported for apikey accounts")
	})

	t.Run("RejectsDuplicateHeadersCaseInsensitive", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "forward", "key": "X-Test"},
					map[string]any{"target": "header", "mode": "inject", "key": "x-test", "value": "prod"},
				},
			},
		})

		require.EqualError(t, err, "duplicate passthrough header key: x-test")
	})

	t.Run("RejectsInvalidBodyPath", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "forward", "key": "messages.0.role"},
				},
			},
		})

		require.EqualError(t, err, "invalid passthrough body path: messages.0.role")
	})

	t.Run("RejectsReservedHeader", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "forward", "key": "Authorization"},
				},
			},
		})

		require.EqualError(t, err, "reserved passthrough header key: authorization")
	})

	t.Run("RejectsBlankHeaderKey", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "forward", "key": "   "},
				},
			},
		})

		require.EqualError(t, err, "passthrough header key is required")
	})

	t.Run("RejectsGeminiUpstreamAuthHeader", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "inject", "key": "X-Goog-Api-Key", "value": "evil-key"},
				},
			},
		})

		require.EqualError(t, err, "reserved passthrough header key: x-goog-api-key")
	})

	t.Run("RejectsCookieHeader", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "forward", "key": "Cookie"},
				},
			},
		})

		require.EqualError(t, err, "reserved passthrough header key: cookie")
	})

	t.Run("RejectsBlankInjectValue", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "inject", "key": "metadata.user_id", "value": "   "},
				},
			},
		})

		require.EqualError(t, err, "passthrough inject value cannot be blank: metadata.user_id")
	})

	t.Run("RejectsReservedBodyPath", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "forward", "key": "model"},
				},
			},
		})

		require.EqualError(t, err, "reserved passthrough body path: model")
	})

	t.Run("RejectsDuplicateBodyPath", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "forward", "key": "metadata.user_id"},
					map[string]any{"target": "body", "mode": "forward", "key": "metadata.user_id"},
				},
			},
		})

		require.EqualError(t, err, "duplicate passthrough body path: metadata.user_id")
	})

	t.Run("RejectsConflictingBodyModes", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "forward", "key": "metadata.user_id"},
					map[string]any{"target": "body", "mode": "inject", "key": "metadata.user_id", "value": "user-1"},
				},
			},
		})

		require.EqualError(t, err, "duplicate passthrough body path: metadata.user_id")
	})

	t.Run("RemovesRulesWhenTypeChangesAwayFromAPIKey", func(t *testing.T) {
		normalized, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			ExistingType:  AccountTypeAPIKey,
			RequestedType: AccountTypeOAuth,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "forward", "key": "X-Test"},
				},
				"other": "keep",
			},
		})

		require.NoError(t, err)
		require.Equal(t, "keep", normalized["other"])
		require.NotContains(t, normalized, "passthrough_fields_enabled")
		require.NotContains(t, normalized, "passthrough_field_rules")
	})
}

func TestApplyAccountPassthroughFields_InjectsHeader(t *testing.T) {
	account := &Account{
		ID:   101,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "inject", Key: "X-Account-Tag", Value: "prod"},
			},
		},
	}
	outbound := http.Header{}

	gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, []byte(`{"input":[]}`), []byte(`{"input":[]}`), outbound)

	require.NoError(t, err)
	require.JSONEq(t, `{"input":[]}`, string(gotBody))
	require.Equal(t, "prod", outbound.Get("X-Account-Tag"))
}

func TestApplyAccountPassthroughFields_DoesNothingWhenDisabled(t *testing.T) {
	account := &Account{
		ID:   102,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": false,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "inject", Key: "X-Account-Tag", Value: "prod"},
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
			},
		},
	}
	outbound := http.Header{"User-Agent": []string{"curl/8.0"}}
	targetBody := []byte(`{"input":[{"type":"text","text":"hi"}]}`)

	gotBody, err := ApplyAccountPassthroughFields(account, http.Header{"X-Trace": []string{"trace-1"}}, []byte(`{"metadata":{"user_id":"user-1"}}`), targetBody, outbound)

	require.NoError(t, err)
	require.Equal(t, string(targetBody), string(gotBody))
	require.Equal(t, "curl/8.0", outbound.Get("User-Agent"))
	require.Empty(t, outbound.Get("X-Account-Tag"))
}

func TestApplyAccountPassthroughFields_ForwardsAllowedHeaderOnly(t *testing.T) {
	account := &Account{
		ID:   103,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "forward", Key: "X-Trace-Id"},
			},
		},
	}
	inbound := http.Header{
		"X-Trace-Id": []string{"trace-1"},
		"X-Leak":     []string{"secret"},
	}
	outbound := http.Header{}

	_, err := ApplyAccountPassthroughFields(account, inbound, nil, []byte(`{"input":[]}`), outbound)

	require.NoError(t, err)
	require.Equal(t, "trace-1", outbound.Get("X-Trace-Id"))
	require.Empty(t, outbound.Get("X-Leak"))
}

func TestApplyAccountPassthroughFields_InjectsBodyStringAtDotPath(t *testing.T) {
	account := &Account{
		ID:   104,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
			},
		},
	}

	gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, nil, []byte(`{"input":[]}`), http.Header{})

	require.NoError(t, err)
	require.Equal(t, "user-1", gjson.GetBytes(gotBody, "metadata.user_id").String())
}

func TestApplyAccountPassthroughFields_AppliesInjectBeforeForwardRegardlessOfRuleOrder(t *testing.T) {
	account := &Account{
		ID:   1041,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "forward", Key: "metadata.user_id"},
				{Target: "body", Mode: "inject", Key: "metadata", Value: "injected-string"},
			},
		},
	}

	gotBody, err := ApplyAccountPassthroughFields(
		account,
		http.Header{},
		[]byte(`{"metadata":{"user_id":"forwarded-user"}}`),
		[]byte(`{"input":[]}`),
		http.Header{},
	)

	require.Nil(t, gotBody)
	require.EqualError(t, err, "invalid_request_error: passthrough body path conflicts with non-object node: metadata")
}

func TestApplyAccountPassthroughFields_ReturnsInvalidRequestOnBodyStructureConflict(t *testing.T) {
	account := &Account{
		ID:   105,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
			},
		},
	}

	_, err := ApplyAccountPassthroughFields(account, http.Header{}, nil, []byte(`{"metadata":"string"}`), http.Header{})

	require.EqualError(t, err, "invalid_request_error: passthrough body path conflicts with non-object node: metadata")
}

func TestApplyAccountPassthroughFields_LogsStructureConflict(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	account := &Account{
		ID:   106,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
			},
		},
	}

	_, err := ApplyAccountPassthroughFields(account, http.Header{}, nil, []byte(`{"metadata":"string"}`), http.Header{})

	require.EqualError(t, err, "invalid_request_error: passthrough body path conflicts with non-object node: metadata")
	require.True(t, logSink.ContainsMessage("passthrough body path conflicts with non-object node"))
	require.True(t, logSink.ContainsFieldValue("account_id", "106"))
	require.True(t, logSink.ContainsFieldValue("target", "body"))
	require.True(t, logSink.ContainsFieldValue("key", "metadata.user_id"))
	require.True(t, logSink.ContainsFieldValue("conflict_node", "metadata"))
}

func TestApplyAccountPassthroughFieldsWithContext_LogsStructureConflictWithContextFields(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	ctx := logger.IntoContext(context.Background(), logger.L().With(zap.String("request_id", "req-123")))
	account := &Account{
		ID:   107,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
			},
		},
	}

	_, err := applyAccountPassthroughFieldsWithContext(ctx, account, http.Header{}, nil, []byte(`{"metadata":"string"}`), http.Header{})

	require.EqualError(t, err, "invalid_request_error: passthrough body path conflicts with non-object node: metadata")
	require.True(t, logSink.ContainsFieldValue("request_id", "req-123"))
}
