//go:build unit

package task3tests

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type gatewayControlSettingRepoStub struct {
	values  map[string]string
	updates map[string]string
	all     map[string]string
}

func (s *gatewayControlSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *gatewayControlSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *gatewayControlSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *gatewayControlSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *gatewayControlSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	s.updates = make(map[string]string, len(settings))
	for k, v := range settings {
		s.updates[k] = v
	}
	return nil
}

func (s *gatewayControlSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	if s.all == nil {
		return map[string]string{}, nil
	}
	result := make(map[string]string, len(s.all))
	for k, v := range s.all {
		result[k] = v
	}
	return result, nil
}

func (s *gatewayControlSettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func newGatewayControlTestConfig() *config.Config {
	return &config.Config{
		Gateway: config.GatewayConfig{
			Sticky: config.GatewayStickyConfig{
				OpenAI:    config.GatewayStickyPlatformConfig{Enabled: true},
				Gemini:    config.GatewayStickyPlatformConfig{Enabled: false},
				Anthropic: config.GatewayStickyPlatformConfig{Enabled: true},
			},
		},
	}
}

func TestSettingService_UpdateSettings_PersistsAndHotUpdatesGatewayControls(t *testing.T) {
	repo := &gatewayControlSettingRepoStub{}
	cfg := newGatewayControlTestConfig()
	svc := service.NewSettingService(repo, cfg)

	err := svc.UpdateSettings(context.Background(), &service.SystemSettings{
		GatewayStickyOpenAIEnabled:    false,
		GatewayStickyGeminiEnabled:    true,
		GatewayStickyAnthropicEnabled: false,
	})

	require.NoError(t, err)
	require.Equal(t, "false", repo.updates[service.SettingKeyGatewayStickyOpenAIEnabled])
	require.Equal(t, "true", repo.updates[service.SettingKeyGatewayStickyGeminiEnabled])
	require.Equal(t, "false", repo.updates[service.SettingKeyGatewayStickyAnthropicEnabled])

	require.False(t, cfg.Gateway.Sticky.OpenAI.Enabled)
	require.True(t, cfg.Gateway.Sticky.Gemini.Enabled)
	require.False(t, cfg.Gateway.Sticky.Anthropic.Enabled)
}

func TestSettingService_GetAllSettings_BackfillsGatewayControlsFromConfigAndDB(t *testing.T) {
	repo := &gatewayControlSettingRepoStub{all: map[string]string{
		service.SettingKeyGatewayStickyOpenAIEnabled: "false",
	}}
	cfg := newGatewayControlTestConfig()
	svc := service.NewSettingService(repo, cfg)

	settings, err := svc.GetAllSettings(context.Background())

	require.NoError(t, err)
	require.False(t, settings.GatewayStickyOpenAIEnabled)
	require.False(t, settings.GatewayStickyGeminiEnabled)
	require.True(t, settings.GatewayStickyAnthropicEnabled)
}

func TestProvideSettingService_LoadsGatewayControlsFromDBIntoConfig(t *testing.T) {
	repo := &gatewayControlSettingRepoStub{values: map[string]string{
		service.SettingKeyGatewayStickyOpenAIEnabled:    "false",
		service.SettingKeyGatewayStickyGeminiEnabled:    "true",
		service.SettingKeyGatewayStickyAnthropicEnabled: "false",
	}}
	cfg := newGatewayControlTestConfig()

	service.ProvideSettingService(nil, repo, nil, nil, cfg)

	require.False(t, cfg.Gateway.Sticky.OpenAI.Enabled)
	require.True(t, cfg.Gateway.Sticky.Gemini.Enabled)
	require.False(t, cfg.Gateway.Sticky.Anthropic.Enabled)
}
