//go:build unit

package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDefaultSchedulerIgnoresRemovedLayeredSettings(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(strconv.FormatBool(enabled), func(t *testing.T) {
			resetOpenAIAdvancedSchedulerSettingCacheForTest()
			t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)
			repo := &openAIAdvancedSchedulerSettingRepoStub{values: map[string]string{
				openAIAdvancedSchedulerSettingKey:                                      strconv.FormatBool(enabled),
				"gateway_openai_ws_scheduler_mode":                                     "layered",
				"gateway_openai_ws_scheduler_layered_error_penalty_threshold":          "0.01",
				"gateway_openai_ws_scheduler_layered_error_penalty_value":              "999",
				"gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier":          "1.01",
				"gateway_openai_ws_scheduler_layered_ttft_penalty_value":               "999",
				"gateway_openai_ws_scheduler_layered_probe_cooldown_seconds":           "1",
				"gateway_openai_ws_scheduler_layered_probe_interval_seconds":           "1",
				"gateway_openai_ws_scheduler_layered_probe_max_failures":               "1",
				"gateway_openai_ws_scheduler_layered_probe_timeout_seconds":            "1",
				"gateway_openai_ws_scheduler_layered_probe_temp_unschedulable_seconds": "999",
			}}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, rateLimitService: &RateLimitService{settingService: &SettingService{settingRepo: repo}}}
			scheduler := svc.getOpenAIAccountSchedulerWithContext(context.Background())
			if enabled {
				require.IsType(t, &defaultOpenAIAccountScheduler{}, scheduler)
				require.Same(t, scheduler, svc.getOpenAIAccountScheduler())
			} else {
				require.Nil(t, scheduler, "old mode must not override the upstream advanced switch")
			}
		})
	}
}
