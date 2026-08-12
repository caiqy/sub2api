//go:build unit

package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Saving settings is a whole-document PUT. A client that sends only the field it
// cares about must not reset everything else: a payload as small as
// `{"risk_control_enabled":true}` used to clear site_name, after which
// getStringOrDefault rendered the empty value as the built-in default and the
// login page silently changed name.

func TestUpdateSettingsPartialPayloadKeepsUnsentKeys(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeySiteName:         "Example Gateway",
		service.SettingKeySiteSubtitle:     "Example Gateway Platform",
		service.SettingKeySMTPHost:         "smtp.example.com",
		service.SettingKeySMTPFrom:         "noreply@example.com",
		service.SettingKeyTurnstileEnabled: "true",
	})

	rec := doUpdateSettings(t, h, map[string]any{"risk_control_enabled": true}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "true", repo.values[service.SettingKeyRiskControlEnabled],
		"the field the caller actually sent must be written")

	require.Equal(t, "Example Gateway", repo.values[service.SettingKeySiteName])
	require.Equal(t, "Example Gateway Platform", repo.values[service.SettingKeySiteSubtitle])
	require.Equal(t, "smtp.example.com", repo.values[service.SettingKeySMTPHost])
	require.Equal(t, "noreply@example.com", repo.values[service.SettingKeySMTPFrom])
	require.Equal(t, "true", repo.values[service.SettingKeyTurnstileEnabled])
}

// A full payload keeps whole-document semantics: fields explicitly set to their
// zero value are still cleared.
func TestUpdateSettingsFullPayloadStillClearsSentEmptyFields(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeySiteName: "Example Gateway",
	})

	rec := doUpdateSettings(t, h, map[string]any{"site_name": ""}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "", repo.values[service.SettingKeySiteName],
		"an explicitly sent empty value is a deliberate clear, not an omission")
}

// smtp_from_email is the one request field whose JSON name differs from its
// setting key; the alias keeps it from being treated as always-omitted.
func TestUpdateSettingsSMTPFromAliasIsWritable(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeySMTPFrom: "old@example.com",
	})

	rec := doUpdateSettings(t, h, map[string]any{"smtp_from_email": "new@example.com"}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "new@example.com", repo.values[service.SettingKeySMTPFrom])
}

func TestUpdateSettingsGrokDefaultBaseURLModeIsWritable(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyGrokDefaultTextModel:           "grok-stored",
		service.SettingKeyGrokCrossClientModelMapEnabled: "true",
		service.SettingKeyGrokDefaultBaseURLMode:         service.GrokDefaultBaseURLModeCLI,
		service.SettingKeySiteName:                       "Example Gateway",
	})

	rec := doUpdateSettings(t, h, map[string]any{
		"grok_default_text_model":             "grok-4.3",
		"grok_cross_client_model_map_enabled": false,
		"grok_default_base_url_mode":          service.GrokDefaultBaseURLModeEUWest1,
	}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "grok-4.3", repo.values[service.SettingKeyGrokDefaultTextModel])
	require.Equal(t, "false", repo.values[service.SettingKeyGrokCrossClientModelMapEnabled])
	require.Equal(t, service.GrokDefaultBaseURLModeEUWest1, repo.values[service.SettingKeyGrokDefaultBaseURLMode])
	require.Equal(t, "Example Gateway", repo.values[service.SettingKeySiteName])
	requireGrokSettingsResponse(t, rec, "grok-4.3", false, service.GrokDefaultBaseURLModeEUWest1)

	getRec := httptest.NewRecorder()
	getContext, _ := gin.CreateTestContext(getRec)
	getContext.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	h.GetSettings(getContext)
	require.Equal(t, http.StatusOK, getRec.Code)
	requireGrokSettingsResponse(t, getRec, "grok-4.3", false, service.GrokDefaultBaseURLModeEUWest1)

	rec = doUpdateSettings(t, h, map[string]any{"grok_default_text_model": ""}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "grok-4.5", repo.values[service.SettingKeyGrokDefaultTextModel])
	require.Equal(t, "false", repo.values[service.SettingKeyGrokCrossClientModelMapEnabled])
	require.Equal(t, service.GrokDefaultBaseURLModeEUWest1, repo.values[service.SettingKeyGrokDefaultBaseURLMode])
	require.Equal(t, "Example Gateway", repo.values[service.SettingKeySiteName])
}

func TestDiffSettingsIncludesGrokDefaults(t *testing.T) {
	changed := diffSettings(
		&service.SystemSettings{
			GrokDefaultTextModel:           "grok-4.1",
			GrokCrossClientModelMapEnabled: true,
			GrokDefaultBaseURLMode:         service.GrokDefaultBaseURLModeCLI,
		},
		&service.SystemSettings{
			GrokDefaultTextModel:           "grok-4.3",
			GrokCrossClientModelMapEnabled: false,
			GrokDefaultBaseURLMode:         service.GrokDefaultBaseURLModeEUWest1,
		},
		&service.AuthSourceDefaultSettings{},
		&service.AuthSourceDefaultSettings{},
		UpdateSettingsRequest{},
	)

	require.Contains(t, changed, service.SettingKeyGrokDefaultTextModel)
	require.Contains(t, changed, service.SettingKeyGrokCrossClientModelMapEnabled)
	require.Contains(t, changed, service.SettingKeyGrokDefaultBaseURLMode)
}

func TestUpdateSettingsChannelMonitorModeAndThroughputAreWritable(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyChannelMonitorMode:           service.ChannelMonitorModeV1,
		service.SettingKeyChannelMonitorHideThroughput: "true",
	})

	rec := doUpdateSettings(t, h, map[string]any{
		"channel_monitor_mode": service.ChannelMonitorModeV2,
	}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.ChannelMonitorModeV2, repo.values[service.SettingKeyChannelMonitorMode])
	require.Equal(t, "true", repo.values[service.SettingKeyChannelMonitorHideThroughput])

	rec = doUpdateSettings(t, h, map[string]any{
		"channel_monitor_hide_throughput": false,
	}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.ChannelMonitorModeV2, repo.values[service.SettingKeyChannelMonitorMode])
	require.Equal(t, "false", repo.values[service.SettingKeyChannelMonitorHideThroughput])

	rec = httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	h.GetSettings(c)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, service.ChannelMonitorModeV2, data["channel_monitor_mode"])
	require.Equal(t, false, data["channel_monitor_hide_throughput"])
}

func TestDiffSettingsIncludesChannelMonitorModeAndThroughput(t *testing.T) {
	changed := diffSettings(
		&service.SystemSettings{
			ChannelMonitorMode:           service.ChannelMonitorModeV1,
			ChannelMonitorHideThroughput: true,
		},
		&service.SystemSettings{
			ChannelMonitorMode:           service.ChannelMonitorModeV2,
			ChannelMonitorHideThroughput: false,
		},
		&service.AuthSourceDefaultSettings{},
		&service.AuthSourceDefaultSettings{},
		UpdateSettingsRequest{},
	)

	require.Contains(t, changed, "channel_monitor_mode")
	require.Contains(t, changed, "channel_monitor_hide_throughput")
}

func TestUpdateSettingsRegistrationEmailDomainQuotaIsWritableAndPreserved(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyRegistrationEmailDomainQuotaEnabled: "true",
		service.SettingKeySiteName:                            "Example Gateway",
	})

	rec := doUpdateSettings(t, h, map[string]any{"site_name": "Updated Gateway"}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "true", repo.values[service.SettingKeyRegistrationEmailDomainQuotaEnabled])

	for _, enabled := range []bool{false, true} {
		rec = doUpdateSettings(t, h, map[string]any{
			"registration_email_domain_quota_enabled": enabled,
		}, nil)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, jsonBool(enabled), repo.values[service.SettingKeyRegistrationEmailDomainQuotaEnabled])
		require.Equal(t, enabled, settingsResponseData(t, rec)["registration_email_domain_quota_enabled"])
	}
}

func TestUpdateSettingsAccountSchedulingThresholdsAreWritableAndPreserved(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyAccountSchedulingThresholds: `{"openai":85,"anthropic":86,"grok":87}`,
	})

	rec := doUpdateSettings(t, h, map[string]any{
		"account_scheduling_thresholds": map[string]int{"openai": 91},
	}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"openai":91,"anthropic":100,"grok":100}`, repo.values[service.SettingKeyAccountSchedulingThresholds])
	require.Equal(t, map[string]any{
		"openai": float64(91), "anthropic": float64(100), "grok": float64(100),
	}, settingsResponseData(t, rec)["account_scheduling_thresholds"])

	rec = doUpdateSettings(t, h, map[string]any{"site_name": "Threshold Gateway"}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"openai":91,"anthropic":100,"grok":100}`, repo.values[service.SettingKeyAccountSchedulingThresholds])

	getRec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(getRec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	h.GetSettings(c)
	require.Equal(t, http.StatusOK, getRec.Code)
	require.Equal(t, map[string]any{
		"openai": float64(91), "anthropic": float64(100), "grok": float64(100),
	}, settingsResponseData(t, getRec)["account_scheduling_thresholds"])
}

func jsonBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func settingsResponseData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	return data
}

func requireGrokSettingsResponse(t *testing.T, rec *httptest.ResponseRecorder, defaultText string, crossClientEnabled bool, baseURLMode string) {
	t.Helper()
	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, defaultText, data["grok_default_text_model"])
	require.Equal(t, crossClientEnabled, data["grok_cross_client_model_map_enabled"])
	require.Equal(t, baseURLMode, data["grok_default_base_url_mode"])
}

func TestUpdateSettingsRejectsTwoCaptchaProviders(t *testing.T) {
	h, _ := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyTurnstileEnabled:   "true",
		service.SettingKeyTurnstileSiteKey:   "site-key",
		service.SettingKeyTurnstileSecretKey: "turnstile-secret",
	})

	rec := doUpdateSettings(t, h, map[string]any{
		"turnstile_enabled":                true,
		"turnstile_site_key":               "site-key",
		"turnstile_secret_key":             "turnstile-secret",
		"tencent_captcha_enabled":          true,
		"tencent_captcha_app_id":           "123456789",
		"tencent_captcha_app_secret_key":   "app-secret",
		"tencent_captcha_cloud_secret_id":  "cloud-secret-id",
		"tencent_captcha_cloud_secret_key": "cloud-secret-key",
	}, nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "cannot be enabled at the same time")
}

func TestUpdateSettingsRejectsClearingEnabledTurnstileSiteKey(t *testing.T) {
	for _, siteKey := range []string{"", "   "} {
		t.Run(siteKey, func(t *testing.T) {
			h, repo := newStepUpSwitchTestHandler(t, map[string]string{
				service.SettingKeyTurnstileEnabled:   "true",
				service.SettingKeyTurnstileSiteKey:   "site-key",
				service.SettingKeyTurnstileSecretKey: "secret-key",
			})

			rec := doUpdateSettings(t, h, map[string]any{"turnstile_site_key": siteKey}, nil)
			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Equal(t, "site-key", repo.values[service.SettingKeyTurnstileSiteKey])
		})
	}
}

func TestUpdateSettingsRequiresFourTencentCaptchaCredentialsWhenEnabled(t *testing.T) {
	h, _ := newStepUpSwitchTestHandler(t, map[string]string{})

	rec := doUpdateSettings(t, h, map[string]any{
		"tencent_captcha_enabled": true,
		"tencent_captcha_app_id":  "123456789",
	}, nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "AppSecretKey")
}

func TestUpdateSettingsRetainsStoredTencentCaptchaCredentialsWhenInputsEmpty(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyTencentCaptchaAppSecretKey:   "stored-app-secret",
		service.SettingKeyTencentCaptchaCloudSecretID:  "stored-cloud-secret-id",
		service.SettingKeyTencentCaptchaCloudSecretKey: "stored-cloud-secret-key",
	})

	rec := doUpdateSettings(t, h, map[string]any{
		"tencent_captcha_enabled":          true,
		"tencent_captcha_app_id":           "123456789",
		"tencent_captcha_app_secret_key":   "",
		"tencent_captcha_cloud_secret_id":  "",
		"tencent_captcha_cloud_secret_key": "",
	}, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "stored-app-secret", repo.values[service.SettingKeyTencentCaptchaAppSecretKey])
	require.Equal(t, "stored-cloud-secret-id", repo.values[service.SettingKeyTencentCaptchaCloudSecretID])
	require.Equal(t, "stored-cloud-secret-key", repo.values[service.SettingKeyTencentCaptchaCloudSecretKey])
}

// 天御站点决定前端加载哪个 SDK 与服务端打哪个接入点，两端必须一致。
// 部分载荷把它重置回中国站，会让已配国际站的部署在下一次任意保存后整体失效。
func TestUpdateSettingsPartialPayloadKeepsTencentCaptchaRegion(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyTencentCaptchaRegion: service.TencentCaptchaRegionINTL,
	})

	rec := doUpdateSettings(t, h, map[string]any{"risk_control_enabled": true}, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.TencentCaptchaRegionINTL,
		repo.values[service.SettingKeyTencentCaptchaRegion])
}

func TestUpdateSettingsNormalizesUnknownTencentCaptchaRegion(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyTencentCaptchaRegion: service.TencentCaptchaRegionINTL,
	})

	rec := doUpdateSettings(t, h, map[string]any{"tencent_captcha_region": "sgp"}, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.TencentCaptchaRegionCN,
		repo.values[service.SettingKeyTencentCaptchaRegion],
		"未知站点必须落回中国站，不能写入无法识别的值")
}

func TestUpdateSettingsWritesTencentCaptchaRegionWhenSent(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{})

	rec := doUpdateSettings(t, h, map[string]any{"tencent_captcha_region": "intl"}, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.TencentCaptchaRegionINTL,
		repo.values[service.SettingKeyTencentCaptchaRegion])
}

func TestUpdateSettingsValidatesTencentCaptchaAppIDWhenEnabledFlagIsOmitted(t *testing.T) {
	h, _ := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyTencentCaptchaEnabled:        "true",
		service.SettingKeyTencentCaptchaAppID:          "123456789",
		service.SettingKeyTencentCaptchaAppSecretKey:   "stored-app-secret",
		service.SettingKeyTencentCaptchaCloudSecretID:  "stored-cloud-secret-id",
		service.SettingKeyTencentCaptchaCloudSecretKey: "stored-cloud-secret-key",
	})

	rec := doUpdateSettings(t, h, map[string]any{
		"tencent_captcha_app_id": "not-a-number",
	}, nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "positive integer")
}
