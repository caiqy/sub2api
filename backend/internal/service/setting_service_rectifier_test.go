//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type rectifierSettingRepoStub struct {
	value string
	err   error
}

func (s *rectifierSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *rectifierSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if key != SettingKeyRectifierSettings {
		return "", ErrSettingNotFound
	}
	if s.err != nil {
		return "", s.err
	}
	return s.value, nil
}

func (s *rectifierSettingRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *rectifierSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *rectifierSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *rectifierSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *rectifierSettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestDefaultRectifierSettings_EnableAPIKeySignatureByDefault(t *testing.T) {
	t.Parallel()

	settings := DefaultRectifierSettings()
	require.True(t, settings.Enabled)
	require.True(t, settings.ThinkingSignatureEnabled)
	require.True(t, settings.APIKeySignatureEnabled)
}

func TestSettingService_GetRectifierSettings_BackfillsMissingAPIKeySignatureField(t *testing.T) {
	t.Parallel()

	repo := &rectifierSettingRepoStub{value: `{"enabled":true,"thinking_signature_enabled":true,"thinking_budget_enabled":true}`}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetRectifierSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.Enabled)
	require.True(t, settings.ThinkingSignatureEnabled)
	require.True(t, settings.ThinkingBudgetEnabled)
	require.True(t, settings.APIKeySignatureEnabled)
}
