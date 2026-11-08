package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSettingHandlerRetiredSchedulerSettingsAreIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	legacy := map[string]any{
		"gateway_openai_ws_scheduler_mode":                                     "layered",
		"gateway_openai_ws_scheduler_layered_error_penalty_threshold":          0.6,
		"gateway_openai_ws_scheduler_layered_error_penalty_value":              100,
		"gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier":          12,
		"gateway_openai_ws_scheduler_layered_ttft_penalty_value":               50,
		"gateway_openai_ws_scheduler_layered_probe_cooldown_seconds":           20,
		"gateway_openai_ws_scheduler_layered_probe_interval_seconds":           20,
		"gateway_openai_ws_scheduler_layered_probe_max_failures":               3,
		"gateway_openai_ws_scheduler_layered_probe_timeout_seconds":            15,
		"gateway_openai_ws_scheduler_layered_probe_temp_unschedulable_seconds": 900,
	}
	repo := &settingHandlerRepoStub{values: map[string]string{}}
	for key := range legacy {
		repo.values[key] = "historical-value"
	}
	cfg := &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}}
	handler := NewSettingHandler(service.NewSettingService(repo, cfg), nil, nil, nil, nil, nil, nil)
	body := map[string]any{
		"promo_code_enabled":                    true,
		"gateway_sticky_openai_enabled":         false,
		"openai_advanced_scheduler_enabled":     true,
		"openai_advanced_scheduler_lb_top_k":    "7",
		"openai_advanced_scheduler_weight_ttft": "0.2",
	}
	for key, value := range legacy {
		body[key] = value
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	for _, method := range []string{http.MethodPut, http.MethodGet} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(method, "/api/v1/admin/settings", bytes.NewReader(raw))
		c.Request.Header.Set("Content-Type", "application/json")
		if method == http.MethodPut {
			handler.UpdateSettings(c)
		} else {
			handler.GetSettings(c)
		}
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var result response.Response
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		data := result.Data.(map[string]any)
		for key := range legacy {
			require.NotContains(t, data, key)
			require.Equal(t, "historical-value", repo.values[key], "historical keys must not be written or deleted")
		}
		require.Equal(t, false, data["gateway_sticky_openai_enabled"])
		require.Equal(t, true, data["openai_advanced_scheduler_enabled"])
		require.Equal(t, "7", data["openai_advanced_scheduler_lb_top_k"])
		require.Equal(t, "0.2", data["openai_advanced_scheduler_weight_ttft"])
	}
	require.False(t, cfg.Gateway.Sticky.OpenAI.Enabled)
}
