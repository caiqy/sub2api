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
	panic("unexpected Set call")
}

func (s *gatewayControlSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
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
			OpenAIWS: config.GatewayOpenAIWSConfig{
				SchedulerMode: "weighted",
				SchedulerLayered: config.GatewayOpenAIWSSchedulerLayeredConfig{
					ErrorPenaltyThreshold: 0.3,
					ErrorPenaltyValue:     100,
					TTFTPenaltyMultiplier: 3.0,
					TTFTPenaltyValue:      50,
					ProbeCooldownSeconds:  60,
					ProbeIntervalSeconds:  30,
					ProbeMaxFailures:      3,
					ProbeTimeoutSeconds:   15,
				},
			},
		},
	}
}

func TestSettingService_UpdateSettings_PersistsAndHotUpdatesGatewayControls(t *testing.T) {
	repo := &gatewayControlSettingRepoStub{}
	cfg := newGatewayControlTestConfig()
	svc := service.NewSettingService(repo, cfg)

	err := svc.UpdateSettings(context.Background(), &service.SystemSettings{
		GatewayStickyOpenAIEnabled:                           false,
		GatewayStickyGeminiEnabled:                           true,
		GatewayStickyAnthropicEnabled:                        false,
		GatewayOpenAIWSSchedulerMode:                         "layered",
		GatewayOpenAIWSSchedulerLayeredErrorPenaltyThreshold: 0.45,
		GatewayOpenAIWSSchedulerLayeredErrorPenaltyValue:     120,
		GatewayOpenAIWSSchedulerLayeredTTFTPenaltyMultiplier: 4.5,
		GatewayOpenAIWSSchedulerLayeredTTFTPenaltyValue:      70,
		GatewayOpenAIWSSchedulerLayeredProbeCooldownSeconds:  90,
		GatewayOpenAIWSSchedulerLayeredProbeIntervalSeconds:  40,
		GatewayOpenAIWSSchedulerLayeredProbeMaxFailures:      5,
		GatewayOpenAIWSSchedulerLayeredProbeTimeoutSeconds:   20,
	})

	require.NoError(t, err)
	require.Equal(t, "false", repo.updates[service.SettingKeyGatewayStickyOpenAIEnabled])
	require.Equal(t, "true", repo.updates[service.SettingKeyGatewayStickyGeminiEnabled])
	require.Equal(t, "false", repo.updates[service.SettingKeyGatewayStickyAnthropicEnabled])
	require.Equal(t, "layered", repo.updates[service.SettingKeyGatewayOpenAIWSSchedulerMode])
	require.Equal(t, "0.45", repo.updates[service.SettingKeyGatewayOpenAIWSSchedulerLayeredErrorPenaltyThreshold])
	require.Equal(t, "120", repo.updates[service.SettingKeyGatewayOpenAIWSSchedulerLayeredErrorPenaltyValue])
	require.Equal(t, "4.5", repo.updates[service.SettingKeyGatewayOpenAIWSSchedulerLayeredTTFTPenaltyMultiplier])
	require.Equal(t, "70", repo.updates[service.SettingKeyGatewayOpenAIWSSchedulerLayeredTTFTPenaltyValue])
	require.Equal(t, "90", repo.updates[service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeCooldownSeconds])
	require.Equal(t, "40", repo.updates[service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeIntervalSeconds])
	require.Equal(t, "5", repo.updates[service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeMaxFailures])
	require.Equal(t, "20", repo.updates[service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeTimeoutSeconds])

	require.False(t, cfg.Gateway.Sticky.OpenAI.Enabled)
	require.True(t, cfg.Gateway.Sticky.Gemini.Enabled)
	require.False(t, cfg.Gateway.Sticky.Anthropic.Enabled)
	require.Equal(t, "layered", cfg.Gateway.OpenAIWS.SchedulerMode)
	require.Equal(t, 0.45, cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold)
	require.Equal(t, 120, cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue)
	require.Equal(t, 4.5, cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier)
	require.Equal(t, 70, cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue)
	require.Equal(t, 90, cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds)
	require.Equal(t, 40, cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds)
	require.Equal(t, 5, cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures)
	require.Equal(t, 20, cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds)
}

func TestSettingService_GetAllSettings_BackfillsGatewayControlsFromConfigAndDB(t *testing.T) {
	repo := &gatewayControlSettingRepoStub{all: map[string]string{
		service.SettingKeyGatewayStickyOpenAIEnabled:                         "false",
		service.SettingKeyGatewayOpenAIWSSchedulerMode:                       "layered",
		service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeTimeoutSeconds: "22",
	}}
	cfg := newGatewayControlTestConfig()
	svc := service.NewSettingService(repo, cfg)

	settings, err := svc.GetAllSettings(context.Background())

	require.NoError(t, err)
	require.False(t, settings.GatewayStickyOpenAIEnabled)
	require.False(t, settings.GatewayStickyGeminiEnabled)
	require.True(t, settings.GatewayStickyAnthropicEnabled)
	require.Equal(t, "layered", settings.GatewayOpenAIWSSchedulerMode)
	require.Equal(t, 0.3, settings.GatewayOpenAIWSSchedulerLayeredErrorPenaltyThreshold)
	require.Equal(t, 100, settings.GatewayOpenAIWSSchedulerLayeredErrorPenaltyValue)
	require.Equal(t, 3.0, settings.GatewayOpenAIWSSchedulerLayeredTTFTPenaltyMultiplier)
	require.Equal(t, 50, settings.GatewayOpenAIWSSchedulerLayeredTTFTPenaltyValue)
	require.Equal(t, 60, settings.GatewayOpenAIWSSchedulerLayeredProbeCooldownSeconds)
	require.Equal(t, 30, settings.GatewayOpenAIWSSchedulerLayeredProbeIntervalSeconds)
	require.Equal(t, 3, settings.GatewayOpenAIWSSchedulerLayeredProbeMaxFailures)
	require.Equal(t, 22, settings.GatewayOpenAIWSSchedulerLayeredProbeTimeoutSeconds)
}

func TestProvideSettingService_LoadsGatewayControlsFromDBIntoConfig(t *testing.T) {
	repo := &gatewayControlSettingRepoStub{values: map[string]string{
		service.SettingKeyGatewayStickyOpenAIEnabled:                           "false",
		service.SettingKeyGatewayStickyGeminiEnabled:                           "true",
		service.SettingKeyGatewayStickyAnthropicEnabled:                        "false",
		service.SettingKeyGatewayOpenAIWSSchedulerMode:                         "layered",
		service.SettingKeyGatewayOpenAIWSSchedulerLayeredErrorPenaltyThreshold: "0.4",
		service.SettingKeyGatewayOpenAIWSSchedulerLayeredErrorPenaltyValue:     "150",
		service.SettingKeyGatewayOpenAIWSSchedulerLayeredTTFTPenaltyMultiplier: "6",
		service.SettingKeyGatewayOpenAIWSSchedulerLayeredTTFTPenaltyValue:      "90",
		service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeCooldownSeconds:  "120",
		service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeIntervalSeconds:  "45",
		service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeMaxFailures:      "6",
		service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeTimeoutSeconds:   "25",
	}}
	cfg := newGatewayControlTestConfig()

	service.ProvideSettingService(repo, nil, nil, cfg, nil)

	require.False(t, cfg.Gateway.Sticky.OpenAI.Enabled)
	require.True(t, cfg.Gateway.Sticky.Gemini.Enabled)
	require.False(t, cfg.Gateway.Sticky.Anthropic.Enabled)
	require.Equal(t, "layered", cfg.Gateway.OpenAIWS.SchedulerMode)
	require.Equal(t, 0.4, cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold)
	require.Equal(t, 150, cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue)
	require.Equal(t, 6.0, cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier)
	require.Equal(t, 90, cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue)
	require.Equal(t, 120, cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds)
	require.Equal(t, 45, cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds)
	require.Equal(t, 6, cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures)
	require.Equal(t, 25, cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds)
}
