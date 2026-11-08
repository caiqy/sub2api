package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newHandlerAdvancedSchedulerRateLimitService(t *testing.T, cfg *config.Config, accounts service.AccountRepository) *service.RateLimitService {
	t.Helper()
	settings := service.NewSettingService(&contentModerationHandlerSettingRepo{values: map[string]string{}}, cfg)
	values, err := settings.GetAllSettings(context.Background())
	require.NoError(t, err)
	values.OpenAIAdvancedSchedulerEnabled = true
	// Publish through the normal write path to replace any cached disabled value.
	require.NoError(t, settings.UpdateSettings(context.Background(), values))
	t.Cleanup(func() {
		values.OpenAIAdvancedSchedulerEnabled = false
		require.NoError(t, settings.UpdateSettings(context.Background(), values))
	})
	limits := service.NewRateLimitService(accounts, nil, cfg, nil, nil)
	limits.SetSettingService(settings)
	return limits
}
