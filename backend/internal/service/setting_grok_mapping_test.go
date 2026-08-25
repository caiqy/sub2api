package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

type grokMappingSettingRepoStub struct {
	values map[string]string
}

func (s *grokMappingSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *grokMappingSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *grokMappingSettingRepoStub) Set(context.Context, string, string) error {
	return nil
}

func (s *grokMappingSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *grokMappingSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}

func (s *grokMappingSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}

func (s *grokMappingSettingRepoStub) Delete(context.Context, string) error {
	return nil
}

func TestInitializeDefaultSettings_GrokCrossClientMappingDefaultsOff(t *testing.T) {
	repo := &grokMappingSettingRepoStub{}
	service := &SettingService{settingRepo: repo, cfg: &config.Config{}}

	require.NoError(t, service.InitializeDefaultSettings(context.Background()))
	require.Equal(t, "false", repo.values[SettingKeyGrokCrossClientModelMapEnabled])
}

func TestSettingServiceParseGrokCrossClientMapping(t *testing.T) {
	original := xai.RuntimeModelMappingOptions()
	t.Cleanup(func() { xai.SetRuntimeModelMappingOptions(original) })

	tests := []struct {
		name     string
		value    *string
		enabled  bool
		defaultM string
	}{
		{name: "missing", enabled: false, defaultM: "grok-4.6"},
		{name: "empty", value: grokMappingStringPtr(""), enabled: false, defaultM: "grok-4.6"},
		{name: "false", value: grokMappingStringPtr("false"), enabled: false, defaultM: "grok-4.6"},
		{name: "uppercase true", value: grokMappingStringPtr("TRUE"), enabled: false, defaultM: "grok-4.6"},
		{name: "spaced true", value: grokMappingStringPtr(" true "), enabled: false, defaultM: "grok-4.6"},
		{name: "true", value: grokMappingStringPtr("true"), enabled: true, defaultM: "grok-4.6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := map[string]string{}
			if tt.value != nil {
				settings[SettingKeyGrokCrossClientModelMapEnabled] = *tt.value
			}

			got := (&SettingService{cfg: &config.Config{}}).parseSettings(settings)
			require.Equal(t, tt.enabled, got.GrokCrossClientModelMapEnabled)

			mapping := xai.DefaultModelMapping()
			for _, wildcard := range []string{"gpt-*", "codex-*", "o1*", "o3*", "o4*", "claude-*"} {
				value, exists := mapping[wildcard]
				require.Equal(t, tt.enabled, exists, wildcard)
				if tt.enabled {
					require.Equal(t, tt.defaultM, value, wildcard)
				}
			}
		})
	}
}

func grokMappingStringPtr(value string) *string {
	return &value
}
