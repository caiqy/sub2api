package service

import (
	"context"
	"encoding/json"
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

	t.Run("AllowsPreviouslyReservedHeaderAndBodyKeys", func(t *testing.T) {
		normalized, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "forward", "key": "Authorization"},
					map[string]any{"target": "body", "mode": "map", "key": "model", "source_key": "source_model"},
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, []PassthroughFieldRule{
			{Target: "header", Mode: "forward", Key: "Authorization"},
			{Target: "body", Mode: "map", Key: "model", SourceKey: "source_model"},
		}, normalized["passthrough_field_rules"])
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

	t.Run("RejectsInvalidRulesStructure", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules":    map[string]any{"target": "header"},
			},
		})

		require.EqualError(t, err, "invalid passthrough field rules")
	})

	t.Run("RejectsNonStringTargetField", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": true, "mode": "forward", "key": "X-Test"},
				},
			},
		})

		require.EqualError(t, err, "passthrough target must be a string")
	})

	t.Run("RejectsNonStringModeField", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": 1, "key": "X-Test"},
				},
			},
		})

		require.EqualError(t, err, "passthrough mode must be a string: X-Test")
	})

	t.Run("RejectsNonStringKeyField", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "forward", "key": 1},
				},
			},
		})

		require.EqualError(t, err, "passthrough key must be a string")
	})

	t.Run("RejectsInvalidRuleEntryStructure", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					"bad-entry",
				},
			},
		})

		require.EqualError(t, err, "invalid passthrough field rule")
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

	t.Run("RejectsNonStringSourceKey", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "map", "key": "target_value", "source_key": true},
				},
			},
		})

		require.EqualError(t, err, "passthrough map source_key must be a string: target_value")
	})

	t.Run("RejectsHeaderMapNonStringSourceKey", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "map", "key": "X-Target-ID", "source_key": 123},
				},
			},
		})

		require.EqualError(t, err, "passthrough map source_key must be a string: X-Target-ID")
	})

	t.Run("RejectsInjectNonStringValue", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "inject", "key": "metadata.user_id", "value": 123},
				},
			},
		})

		require.EqualError(t, err, "passthrough inject value must be a string: metadata.user_id")
	})

	t.Run("RejectsBodyPathPrefixConflicts", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "inject", "key": "metadata", "value": "root"},
					map[string]any{"target": "body", "mode": "forward", "key": "metadata.user_id"},
				},
			},
		})

		require.EqualError(t, err, "conflicting passthrough body path prefixes: metadata, metadata.user_id")
	})

	t.Run("RejectsMapWithoutSourceKey", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "map", "key": "metadata.user_id"},
				},
			},
		})

		require.EqualError(t, err, "passthrough map source_key is required: metadata.user_id")
	})

	t.Run("RejectsHeaderMapWithoutSourceKey", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "map", "key": "X-Target-ID"},
				},
			},
		})

		require.EqualError(t, err, "passthrough map source_key is required: X-Target-ID")
	})

	t.Run("RejectsHeaderMapWithSameSourceAndTargetCaseInsensitive", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "map", "key": "X-Trace-ID", "source_key": " x-trace-id "},
				},
			},
		})

		require.EqualError(t, err, "passthrough map source_key and key must be different: x-trace-id")
	})

	t.Run("RejectsBodyMapWithSameSourceAndTarget", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "map", "key": "metadata.user_id", "source_key": " metadata.user_id "},
				},
			},
		})

		require.EqualError(t, err, "passthrough map source_key and key must be different: metadata.user_id")
	})

	t.Run("NormalizesByModeAndPreservesOriginalHeaderKeyCase", func(t *testing.T) {
		normalized, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "forward", "key": " X-Trace-ID ", "value": "ignored", "source_key": "ignored"},
					map[string]any{"target": "header", "mode": "inject", "key": " X-Env ", "value": "prod", "source_key": "ignored"},
					map[string]any{"target": "header", "mode": "map", "key": " X-Request-ID ", "source_key": " X-Source-ID ", "value": "ignored"},
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, []PassthroughFieldRule{
			{Target: "header", Mode: "forward", Key: "X-Trace-ID"},
			{Target: "header", Mode: "inject", Key: "X-Env", Value: "prod"},
			{Target: "header", Mode: "map", Key: "X-Request-ID", SourceKey: "X-Source-ID"},
		}, normalized["passthrough_field_rules"])
	})

	t.Run("ValidatesRulesEvenWhenDisabled", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": false,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "map", "key": "X-Trace-ID"},
				},
			},
		})

		require.EqualError(t, err, "passthrough map source_key is required: X-Trace-ID")
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

func TestApplyAccountPassthroughFields_InjectAndForwardOverrideBaseOutboundValues(t *testing.T) {
	account := &Account{
		ID:   1042,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "inject", Key: "X-Env", Value: "prod"},
				{Target: "header", Mode: "forward", Key: "X-Trace-ID"},
			},
		},
	}
	outbound := http.Header{
		"X-Env":      []string{"staging"},
		"X-Trace-ID": []string{"base-trace"},
	}

	_, err := ApplyAccountPassthroughFields(account, http.Header{"X-Trace-ID": []string{"request-trace"}}, nil, []byte(`{"input":[]}`), outbound)

	require.NoError(t, err)
	require.Equal(t, "prod", outbound.Get("X-Env"))
	require.Equal(t, "request-trace", outbound.Get("X-Trace-ID"))
}

func TestApplyAccountPassthroughFields_InjectAndForwardOverrideBaseOutboundBodyValues(t *testing.T) {
	t.Run("InjectOverridesExistingTargetBodyValue", func(t *testing.T) {
		account := &Account{
			ID:   10421,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "injected-user"},
				},
			},
		}

		gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, nil, []byte(`{"metadata":{"user_id":"base-user"}}`), http.Header{})

		require.NoError(t, err)
		require.Equal(t, "injected-user", gjson.GetBytes(gotBody, "metadata.user_id").String())
	})

	t.Run("ForwardOverridesExistingTargetBodyValue", func(t *testing.T) {
		account := &Account{
			ID:   10422,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "body", Mode: "forward", Key: "metadata.user_id"},
				},
			},
		}

		gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, []byte(`{"metadata":{"user_id":"source-user"}}`), []byte(`{"metadata":{"user_id":"base-user"}}`), http.Header{})

		require.NoError(t, err)
		require.Equal(t, "source-user", gjson.GetBytes(gotBody, "metadata.user_id").String())
	})
}

func TestApplyAccountPassthroughFields_HeaderMapSkipsWhenTargetExistsInBaseOutbound(t *testing.T) {
	account := &Account{
		ID:   1043,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "map", Key: "X-Target-ID", SourceKey: "X-Source-ID"},
			},
		},
	}
	outbound := http.Header{}
	outbound.Set("X-Target-ID", "base-target")

	_, err := ApplyAccountPassthroughFields(account, http.Header{"X-Source-ID": []string{"source-id"}}, nil, []byte(`{"input":[]}`), outbound)

	require.NoError(t, err)
	require.Equal(t, "base-target", outbound.Get("X-Target-ID"))
}

func TestNormalizeAccountPassthroughFields_RejectsInjectAndMapSharingSameTargetKey(t *testing.T) {
	t.Run("HeaderTarget", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "header", "mode": "inject", "key": "X-Target-ID", "value": "injected"},
					map[string]any{"target": "header", "mode": "map", "key": "X-Target-ID", "source_key": "X-Source-ID"},
				},
			},
		})

		require.EqualError(t, err, "duplicate passthrough header key: x-target-id")
	})

	t.Run("BodyTarget", func(t *testing.T) {
		_, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
			RequestedType: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []any{
					map[string]any{"target": "body", "mode": "inject", "key": "metadata.user_id", "value": "injected"},
					map[string]any{"target": "body", "mode": "map", "key": "metadata.user_id", "source_key": "source.user_id"},
				},
			},
		})

		require.EqualError(t, err, "duplicate passthrough body path: metadata.user_id")
	})
}

func TestApplyAccountPassthroughFields_HeaderMapDoesNotChainAcrossRules(t *testing.T) {
	account := &Account{
		ID:   1044,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "map", Key: "X-First", SourceKey: "X-Source"},
				{Target: "header", Mode: "map", Key: "X-Second", SourceKey: "X-First"},
			},
		},
	}
	outbound := http.Header{}

	_, err := ApplyAccountPassthroughFields(account, http.Header{"X-Source": []string{"value-1"}}, nil, []byte(`{"input":[]}`), outbound)

	require.NoError(t, err)
	require.Equal(t, "value-1", outbound.Get("X-First"))
	require.Empty(t, outbound.Get("X-Second"))
}

func TestApplyAccountPassthroughFields_BodyMapDoesNotChainAcrossRules(t *testing.T) {
	account := &Account{
		ID:   10441,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "map", Key: "target.b", SourceKey: "source.a"},
				{Target: "body", Mode: "map", Key: "target.c", SourceKey: "target.b"},
			},
		},
	}

	gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, []byte(`{"source":{"a":"value-1"}}`), []byte(`{}`), http.Header{})

	require.NoError(t, err)
	require.Equal(t, "value-1", gjson.GetBytes(gotBody, "target.b").String())
	require.False(t, gjson.GetBytes(gotBody, "target.c").Exists())
}

func TestApplyAccountPassthroughFields_ReturnsErrorWhenHeaderRulesNeedNilOutbound(t *testing.T) {
	t.Run("HeaderInjectWithNilOutbound", func(t *testing.T) {
		account := &Account{
			ID:   10442,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "header", Mode: "inject", Key: "X-Env", Value: "prod"},
				},
			},
		}

		_, err := ApplyAccountPassthroughFields(account, http.Header{}, nil, []byte(`{}`), nil)

		require.EqualError(t, err, "passthrough outbound headers are required for header rules")
	})

	t.Run("HeaderForwardWithNilOutbound", func(t *testing.T) {
		account := &Account{
			ID:   10443,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "header", Mode: "forward", Key: "X-Trace-ID"},
				},
			},
		}

		_, err := ApplyAccountPassthroughFields(account, http.Header{"X-Trace-ID": []string{"trace-1"}}, nil, []byte(`{}`), nil)

		require.EqualError(t, err, "passthrough outbound headers are required for header rules")
	})

	t.Run("HeaderMapWithNilOutbound", func(t *testing.T) {
		account := &Account{
			ID:   10444,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "header", Mode: "map", Key: "X-Target-ID", SourceKey: "X-Source-ID"},
				},
			},
		}

		_, err := ApplyAccountPassthroughFields(account, http.Header{"X-Source-ID": []string{"source-1"}}, nil, []byte(`{}`), nil)

		require.EqualError(t, err, "passthrough outbound headers are required for header rules")
	})

	t.Run("BodyOnlyRulesAllowNilOutbound", func(t *testing.T) {
		account := &Account{
			ID:   10445,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
				},
			},
		}

		gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, nil, []byte(`{}`), nil)

		require.NoError(t, err)
		require.Equal(t, "user-1", gjson.GetBytes(gotBody, "metadata.user_id").String())
	})

	t.Run("MixedHeaderAndBodyRulesFailBeforeAnyBodyWrite", func(t *testing.T) {
		account := &Account{
			ID:   10446,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
					{Target: "header", Mode: "inject", Key: "X-Env", Value: "prod"},
				},
			},
		}

		gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, nil, []byte(`{"input":[]}`), nil)

		require.Nil(t, gotBody)
		require.EqualError(t, err, "passthrough outbound headers are required for header rules")
	})
}

func TestApplyAccountPassthroughFields_BodyMapCopiesNullObjectAndArray(t *testing.T) {
	account := &Account{
		ID:   1045,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "map", Key: "copied_null", SourceKey: "nullable"},
				{Target: "body", Mode: "map", Key: "copied_obj", SourceKey: "payload"},
				{Target: "body", Mode: "map", Key: "copied_arr", SourceKey: "items"},
			},
		},
	}

	gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, []byte(`{"nullable":null,"payload":{"id":"u1"},"items":[1,{"k":"v"}]}`), nil, http.Header{})

	require.NoError(t, err)
	require.Equal(t, "null", gjson.GetBytes(gotBody, "copied_null").Raw)
	require.JSONEq(t, `{"id":"u1"}`, gjson.GetBytes(gotBody, "copied_obj").Raw)
	require.JSONEq(t, `[1,{"k":"v"}]`, gjson.GetBytes(gotBody, "copied_arr").Raw)
}

func TestApplyAccountPassthroughFields_BodyMapDeepCopiesObjectValues(t *testing.T) {
	original := map[string]any{"nested": map[string]any{"value": 1.0}}
	cloned, ok := clonePassthroughJSONValue(original).(map[string]any)
	require.True(t, ok)

	clonedNested, ok := cloned["nested"].(map[string]any)
	require.True(t, ok)
	clonedNested["extra"] = "new"

	originalNested, ok := original["nested"].(map[string]any)
	require.True(t, ok)
	_, exists := originalNested["extra"]
	require.False(t, exists)
}

func TestApplyAccountPassthroughFields_BodyMapSkipsWhenTargetPathExistsInSourceOrTarget(t *testing.T) {
	t.Run("OriginalInboundBodyAlreadyHasTarget", func(t *testing.T) {
		account := &Account{
			ID:   1047,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "body", Mode: "map", Key: "metadata.user_id", SourceKey: "source.user_id"},
				},
			},
		}

		gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, []byte(`{"metadata":{"user_id":"original"},"source":{"user_id":"mapped"}}`), []byte(`{"input":[]}`), http.Header{})

		require.NoError(t, err)
		require.False(t, gjson.GetBytes(gotBody, "metadata.user_id").Exists())
	})

	t.Run("BaseOutboundBodyAlreadyHasTargetEvenWhenNull", func(t *testing.T) {
		account := &Account{
			ID:   1048,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "body", Mode: "map", Key: "metadata.user_id", SourceKey: "source.user_id"},
				},
			},
		}

		gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, []byte(`{"source":{"user_id":"mapped"}}`), []byte(`{"metadata":{"user_id":null}}`), http.Header{})

		require.NoError(t, err)
		require.Equal(t, "null", gjson.GetBytes(gotBody, "metadata.user_id").Raw)
	})
}

func TestApplyAccountPassthroughFields_BodyMapNonObjectAncestorDoesNotCountAsExistingAndLogsConflict(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	account := &Account{
		ID:   1049,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "map", Key: "metadata.user_id", SourceKey: "source.user_id"},
			},
		},
	}

	_, err := ApplyAccountPassthroughFields(account, http.Header{}, []byte(`{"metadata":"source-string","source":{"user_id":"mapped"}}`), []byte(`{"metadata":"target-string"}`), http.Header{})

	require.EqualError(t, err, "invalid_request_error: passthrough body path conflicts with non-object node: metadata")
	require.True(t, logSink.ContainsMessage("passthrough body path conflicts with non-object node"))
	require.True(t, logSink.ContainsFieldValue("account_id", "1049"))
	require.True(t, logSink.ContainsFieldValue("key", "metadata.user_id"))
}

func TestApplyAccountPassthroughFields_BodyMapUsesObjectRootForNilOrNonObjectTarget(t *testing.T) {
	t.Run("NilTargetBodyStartsFromEmptyObject", func(t *testing.T) {
		account := &Account{
			ID:   1050,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "body", Mode: "map", Key: "metadata.user_id", SourceKey: "source.user_id"},
				},
			},
		}

		gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, []byte(`{"source":{"user_id":"mapped"}}`), nil, http.Header{})

		require.NoError(t, err)
		require.Equal(t, "mapped", gjson.GetBytes(gotBody, "metadata.user_id").String())
	})

	t.Run("NonObjectTargetBodyReplacesRoot", func(t *testing.T) {
		account := &Account{
			ID:   1051,
			Type: AccountTypeAPIKey,
			Extra: map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules": []PassthroughFieldRule{
					{Target: "body", Mode: "map", Key: "metadata.user_id", SourceKey: "source.user_id"},
				},
			},
		}

		gotBody, err := ApplyAccountPassthroughFields(account, http.Header{}, []byte(`{"source":{"user_id":"mapped"}}`), []byte(`[]`), http.Header{})

		require.NoError(t, err)
		require.JSONEq(t, `{"metadata":{"user_id":"mapped"}}`, string(gotBody))
	})
}

func TestApplyAccountPassthroughFields_BodyMapFailsOnInvalidTargetJSON(t *testing.T) {
	account := &Account{
		ID:   1052,
		Type: AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "map", Key: "metadata.user_id", SourceKey: "source.user_id"},
			},
		},
	}

	_, err := ApplyAccountPassthroughFields(account, http.Header{}, []byte(`{"source":{"user_id":"mapped"}}`), []byte(`{"bad":`), http.Header{})

	require.Error(t, err)
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

	require.JSONEq(t, `{"input":[]}`, string(gotBody))
	require.EqualError(t, err, "conflicting passthrough body path prefixes: metadata.user_id, metadata")
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

func TestClonePassthroughJSONValue_DeepCopiesArrayValues(t *testing.T) {
	original := []any{map[string]any{"nested": []any{"a"}}}
	cloned, ok := clonePassthroughJSONValue(original).([]any)
	require.True(t, ok)

	clonedMap, ok := cloned[0].(map[string]any)
	require.True(t, ok)
	clonedMap["nested"] = []any{"changed"}

	encoded, err := json.Marshal(original)
	require.NoError(t, err)
	require.JSONEq(t, `[{"nested":["a"]}]`, string(encoded))
}
