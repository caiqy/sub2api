//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type gatewayRuntimeSettingRepoStub struct {
	getValueByKey map[string]string
	getErrByKey   map[string]error
	setErr        error
	setCalls      []gatewayRuntimeSettingRepoSetCall
}

type gatewayRuntimeSettingRepoSetCall struct {
	key   string
	value string
}

func (s *gatewayRuntimeSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *gatewayRuntimeSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if err, ok := s.getErrByKey[key]; ok {
		return "", err
	}
	if value, ok := s.getValueByKey[key]; ok {
		return value, nil
	}
	return "", nil
}

func (s *gatewayRuntimeSettingRepoStub) Set(ctx context.Context, key, value string) error {
	s.setCalls = append(s.setCalls, gatewayRuntimeSettingRepoSetCall{key: key, value: value})
	return s.setErr
}

func (s *gatewayRuntimeSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *gatewayRuntimeSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *gatewayRuntimeSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *gatewayRuntimeSettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type gatewayRuntimeIdleInvalidatorStub struct {
	calls int
}

func (s *gatewayRuntimeIdleInvalidatorStub) InvalidateIdleClients() {
	s.calls++
}

func newGatewayRuntimeTestConfig(responseHeaderTimeout, streamDataIntervalTimeout int) *config.Config {
	return &config.Config{
		Gateway: config.GatewayConfig{
			ResponseHeaderTimeout:     responseHeaderTimeout,
			StreamDataIntervalTimeout: streamDataIntervalTimeout,
		},
	}
}

func TestSettingService_GetGatewayRuntimeSettings_FallsBackToConfigWhenDBValueMissing(t *testing.T) {
	repo := &gatewayRuntimeSettingRepoStub{}
	cfg := newGatewayRuntimeTestConfig(120, 60)

	svc := NewSettingService(repo, cfg)
	got, err := svc.GetGatewayRuntimeSettings(context.Background())

	require.NoError(t, err)
	require.Equal(t, &GatewayRuntimeSettings{
		ResponseHeaderTimeout:     120,
		StreamDataIntervalTimeout: 60,
	}, got)
}

func TestNewSettingService_LoadsGatewayRuntimeSettingsFromDB(t *testing.T) {
	repo := &gatewayRuntimeSettingRepoStub{
		getValueByKey: map[string]string{
			SettingKeyGatewayRuntimeSettings: `{"response_header_timeout":45,"stream_data_interval_timeout":90}`,
		},
	}
	cfg := newGatewayRuntimeTestConfig(120, 60)

	svc := NewSettingService(repo, cfg)
	got, err := svc.GetGatewayRuntimeSettings(context.Background())

	require.NotNil(t, svc)
	require.NoError(t, err)
	require.Equal(t, 45, cfg.Gateway.ResponseHeaderTimeout)
	require.Equal(t, 90, cfg.Gateway.StreamDataIntervalTimeout)
	require.Equal(t, &GatewayRuntimeSettings{
		ResponseHeaderTimeout:     45,
		StreamDataIntervalTimeout: 90,
	}, got)
}

func TestSettingService_SetGatewayRuntimeSettings_PersistsUpdatesCfgAndInvalidatesOnResponseHeaderTimeoutChange(t *testing.T) {
	repo := &gatewayRuntimeSettingRepoStub{}
	cfg := newGatewayRuntimeTestConfig(120, 60)
	svc := NewSettingService(repo, cfg)
	invalidator := &gatewayRuntimeIdleInvalidatorStub{}
	svc.SetGatewayRuntimeIdleInvalidator(invalidator)

	err := svc.SetGatewayRuntimeSettings(context.Background(), &GatewayRuntimeSettings{
		ResponseHeaderTimeout:     180,
		StreamDataIntervalTimeout: 0,
	})

	require.NoError(t, err)
	require.Len(t, repo.setCalls, 1)
	require.Equal(t, SettingKeyGatewayRuntimeSettings, repo.setCalls[0].key)

	var persisted GatewayRuntimeSettings
	require.NoError(t, json.Unmarshal([]byte(repo.setCalls[0].value), &persisted))
	require.Equal(t, GatewayRuntimeSettings{
		ResponseHeaderTimeout:     180,
		StreamDataIntervalTimeout: 0,
	}, persisted)
	require.Equal(t, 180, cfg.Gateway.ResponseHeaderTimeout)
	require.Equal(t, 0, cfg.Gateway.StreamDataIntervalTimeout)
	require.Equal(t, 1, invalidator.calls)
}

func TestSettingService_SetGatewayRuntimeSettings_RejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		settings *GatewayRuntimeSettings
	}{
		{name: "nil settings", settings: nil},
		{name: "response header timeout must be positive", settings: &GatewayRuntimeSettings{ResponseHeaderTimeout: 0, StreamDataIntervalTimeout: 60}},
		{name: "stream interval timeout too small", settings: &GatewayRuntimeSettings{ResponseHeaderTimeout: 120, StreamDataIntervalTimeout: 29}},
		{name: "stream interval timeout too large", settings: &GatewayRuntimeSettings{ResponseHeaderTimeout: 120, StreamDataIntervalTimeout: 301}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &gatewayRuntimeSettingRepoStub{}
			cfg := newGatewayRuntimeTestConfig(120, 60)
			svc := NewSettingService(repo, cfg)
			invalidator := &gatewayRuntimeIdleInvalidatorStub{}
			svc.SetGatewayRuntimeIdleInvalidator(invalidator)

			err := svc.SetGatewayRuntimeSettings(context.Background(), tt.settings)

			require.Error(t, err)
			require.Equal(t, 400, infraerrors.Code(err))
			require.Empty(t, repo.setCalls)
			require.Equal(t, 120, cfg.Gateway.ResponseHeaderTimeout)
			require.Equal(t, 60, cfg.Gateway.StreamDataIntervalTimeout)
			require.Zero(t, invalidator.calls)
		})
	}
}

func TestSettingService_SetGatewayRuntimeSettings_DoesNotInvalidateWhenResponseHeaderTimeoutUnchanged(t *testing.T) {
	repo := &gatewayRuntimeSettingRepoStub{}
	cfg := newGatewayRuntimeTestConfig(120, 60)
	svc := NewSettingService(repo, cfg)
	invalidator := &gatewayRuntimeIdleInvalidatorStub{}
	svc.SetGatewayRuntimeIdleInvalidator(invalidator)

	err := svc.SetGatewayRuntimeSettings(context.Background(), &GatewayRuntimeSettings{
		ResponseHeaderTimeout:     120,
		StreamDataIntervalTimeout: 90,
	})

	require.NoError(t, err)
	require.Equal(t, 120, cfg.Gateway.ResponseHeaderTimeout)
	require.Equal(t, 90, cfg.Gateway.StreamDataIntervalTimeout)
	require.Zero(t, invalidator.calls)
}

func TestSettingService_SetGatewayRuntimeSettings_DoesNotMutateCfgOrInvalidateWhenPersistFails(t *testing.T) {
	repo := &gatewayRuntimeSettingRepoStub{setErr: errors.New("db write failed")}
	cfg := newGatewayRuntimeTestConfig(120, 60)
	svc := NewSettingService(repo, cfg)
	invalidator := &gatewayRuntimeIdleInvalidatorStub{}
	svc.SetGatewayRuntimeIdleInvalidator(invalidator)

	err := svc.SetGatewayRuntimeSettings(context.Background(), &GatewayRuntimeSettings{
		ResponseHeaderTimeout:     180,
		StreamDataIntervalTimeout: 90,
	})

	require.Error(t, err)
	require.Len(t, repo.setCalls, 1)
	require.Equal(t, 120, cfg.Gateway.ResponseHeaderTimeout)
	require.Equal(t, 60, cfg.Gateway.StreamDataIntervalTimeout)
	require.Zero(t, invalidator.calls)
}

func TestNewSettingService_LoadsGatewayRuntimeSettings_IgnoreInvalidDBPayloads(t *testing.T) {
	tests := []struct {
		name                 string
		dbValue              string
		getErr               error
		expectResponseHeader int
		expectStreamInterval int
	}{
		{
			name:                 "db error falls back silently",
			getErr:               errors.New("db unavailable"),
			expectResponseHeader: 120,
			expectStreamInterval: 60,
		},
		{
			name:                 "invalid json does not pollute config",
			dbValue:              `{`,
			expectResponseHeader: 120,
			expectStreamInterval: 60,
		},
		{
			name:                 "invalid db values do not pollute config",
			dbValue:              `{"response_header_timeout":0,"stream_data_interval_timeout":29}`,
			expectResponseHeader: 120,
			expectStreamInterval: 60,
		},
		{
			name:                 "only valid db fields override config",
			dbValue:              `{"response_header_timeout":45,"stream_data_interval_timeout":29}`,
			expectResponseHeader: 45,
			expectStreamInterval: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &gatewayRuntimeSettingRepoStub{}
			if tt.dbValue != "" {
				repo.getValueByKey = map[string]string{
					SettingKeyGatewayRuntimeSettings: tt.dbValue,
				}
			}
			if tt.getErr != nil {
				repo.getErrByKey = map[string]error{
					SettingKeyGatewayRuntimeSettings: tt.getErr,
				}
			}
			cfg := newGatewayRuntimeTestConfig(120, 60)

			NewSettingService(repo, cfg)

			require.Equal(t, tt.expectResponseHeader, cfg.Gateway.ResponseHeaderTimeout)
			require.Equal(t, tt.expectStreamInterval, cfg.Gateway.StreamDataIntervalTimeout)
		})
	}
}
