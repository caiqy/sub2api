//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccount_IsOpenAIProbeEnabled(t *testing.T) {
	cases := []struct {
		name string
		acc  *Account
		want bool
	}{
		{name: "nil account", acc: nil, want: true},
		{name: "non-openai platform", acc: &Account{Platform: PlatformAnthropic, Extra: map[string]any{"openai_probe_enabled": false}}, want: true},
		{name: "openai with nil extra", acc: &Account{Platform: PlatformOpenAI}, want: true},
		{name: "openai with key absent", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{}}, want: true},
		{name: "openai explicit true", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_enabled": true}}, want: true},
		{name: "openai explicit false", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_enabled": false}}, want: false},
		{name: "openai non-bool type falls back to true", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_enabled": "false"}}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.acc.IsOpenAIProbeEnabled())
		})
	}
}

func TestAccount_GetOpenAIProbeModel(t *testing.T) {
	cases := []struct {
		name string
		acc  *Account
		want string
	}{
		{name: "nil account", acc: nil, want: ""},
		{name: "nil extra", acc: &Account{Platform: PlatformOpenAI}, want: ""},
		{name: "key absent", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{}}, want: ""},
		{name: "empty string", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_model": ""}}, want: ""},
		{name: "non-string type", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_model": 123}}, want: ""},
		{name: "explicit value", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_model": "gpt-image-2"}}, want: "gpt-image-2"},
		{name: "value with surrounding whitespace returns raw", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_model": "  gpt-image-2  "}}, want: "  gpt-image-2  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.acc.GetOpenAIProbeModel())
		})
	}
}
