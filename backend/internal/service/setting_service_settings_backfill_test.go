//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingValueRepoStub struct {
	values map[string]string
	errs   map[string]error
}

func (s *settingValueRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingValueRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if err, ok := s.errs[key]; ok {
		return "", err
	}
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *settingValueRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingValueRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *settingValueRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingValueRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingValueRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingService_GetOverloadCooldownSettings_BackfillsLegacyEmptyPayloadWithDefaults(t *testing.T) {
	t.Parallel()

	repo := &settingValueRepoStub{values: map[string]string{
		SettingKeyOverloadCooldownSettings: `{}`,
	}}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetOverloadCooldownSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.Enabled)
	require.Equal(t, 10, settings.CooldownMinutes)
}

func TestSettingService_GetStreamTimeoutSettings_BackfillsLegacyEmptyPayloadWithDefaults(t *testing.T) {
	t.Parallel()

	repo := &settingValueRepoStub{values: map[string]string{
		SettingKeyStreamTimeoutSettings: `{}`,
	}}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetStreamTimeoutSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.Enabled)
	require.Equal(t, StreamTimeoutActionTempUnsched, settings.Action)
	require.Equal(t, 5, settings.TempUnschedMinutes)
	require.Equal(t, 3, settings.ThresholdCount)
	require.Equal(t, 10, settings.ThresholdWindowMinutes)
}
