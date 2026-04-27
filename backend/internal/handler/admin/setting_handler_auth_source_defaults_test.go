package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type settingHandlerRepoStub struct {
	values      map[string]string
	lastUpdates map[string]string
}

func (s *settingHandlerRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *settingHandlerRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if key == service.SettingKeyGatewayRuntimeSettings {
		return "", service.ErrSettingNotFound
	}
	panic("unexpected GetValue call")
}

func (s *settingHandlerRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingHandlerRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingHandlerRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	s.lastUpdates = make(map[string]string, len(settings))
	for key, value := range settings {
		s.lastUpdates[key] = value
		if s.values == nil {
			s.values = map[string]string{}
		}
		s.values[key] = value
	}
	return nil
}

func (s *settingHandlerRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *settingHandlerRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type failingAuthSourceSettingsRepoStub struct {
	values map[string]string
	err    error
}

func (s *failingAuthSourceSettingsRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *failingAuthSourceSettingsRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if key == service.SettingKeyGatewayRuntimeSettings {
		return "", service.ErrSettingNotFound
	}
	panic("unexpected GetValue call")
}

func (s *failingAuthSourceSettingsRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *failingAuthSourceSettingsRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *failingAuthSourceSettingsRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if _, ok := settings[service.SettingKeyAuthSourceDefaultEmailBalance]; ok {
		return s.err
	}
	for key, value := range settings {
		if s.values == nil {
			s.values = map[string]string{}
		}
		s.values[key] = value
	}
	return nil
}

func (s *failingAuthSourceSettingsRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *failingAuthSourceSettingsRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingHandler_GetSettings_InjectsAuthSourceDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeyRegistrationEnabled:                 "true",
			service.SettingKeyPromoCodeEnabled:                    "true",
			service.SettingKeyAuthSourceDefaultEmailBalance:       "9.5",
			service.SettingKeyAuthSourceDefaultEmailConcurrency:   "8",
			service.SettingKeyAuthSourceDefaultEmailSubscriptions: `[{"group_id":31,"validity_days":15}]`,
			service.SettingKeyForceEmailOnThirdPartySignup:        "true",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)

	handler.GetSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, 9.5, data["auth_source_default_email_balance"])
	require.Equal(t, float64(8), data["auth_source_default_email_concurrency"])
	require.Equal(t, true, data["force_email_on_third_party_signup"])

	subscriptions, ok := data["auth_source_default_email_subscriptions"].([]any)
	require.True(t, ok)
	require.Len(t, subscriptions, 1)
}

func TestSettingHandler_UpdateSettings_PreservesOmittedAuthSourceDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeyRegistrationEnabled:                    "false",
			service.SettingKeyPromoCodeEnabled:                       "true",
			service.SettingKeyAuthSourceDefaultEmailBalance:          "9.5",
			service.SettingKeyAuthSourceDefaultEmailConcurrency:      "8",
			service.SettingKeyAuthSourceDefaultEmailSubscriptions:    `[{"group_id":31,"validity_days":15}]`,
			service.SettingKeyAuthSourceDefaultEmailGrantOnSignup:    "true",
			service.SettingKeyAuthSourceDefaultEmailGrantOnFirstBind: "false",
			service.SettingKeyForceEmailOnThirdPartySignup:           "true",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"registration_enabled":              true,
		"promo_code_enabled":                true,
		"auth_source_default_email_balance": 12.75,
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "12.75000000", repo.values[service.SettingKeyAuthSourceDefaultEmailBalance])
	require.Equal(t, "8", repo.values[service.SettingKeyAuthSourceDefaultEmailConcurrency])
	require.Equal(t, `[{"group_id":31,"validity_days":15}]`, repo.values[service.SettingKeyAuthSourceDefaultEmailSubscriptions])
	require.Equal(t, "true", repo.values[service.SettingKeyForceEmailOnThirdPartySignup])

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, 12.75, data["auth_source_default_email_balance"])
	require.Equal(t, float64(8), data["auth_source_default_email_concurrency"])
	require.Equal(t, true, data["force_email_on_third_party_signup"])
}

func TestSettingHandler_UpdateSettings_PreservesOmittedBackendModeFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeyPromoCodeEnabled:               "true",
			service.SettingKeyAllowUngroupedKeyScheduling:    "true",
			service.SettingKeyBackendModeEnabled:             "true",
			service.SettingKeyRegistrationEnabled:            "false",
			service.SettingKeyInvitationCodeEnabled:          "false",
			service.SettingKeyPasswordResetEnabled:           "false",
			service.SettingKeyEmailVerifyEnabled:             "false",
			service.SettingKeyHideCcsImportButton:            "false",
			service.SettingKeyPurchaseSubscriptionEnabled:    "false",
			service.SettingKeyEnableModelFallback:            "false",
			service.SettingKeyEnableIdentityPatch:            "true",
			service.SettingKeyOpsMonitoringEnabled:           "true",
			service.SettingKeyOpsRealtimeMonitoringEnabled:   "true",
			service.SettingKeyAccountQuotaNotifyEnabled:      "false",
			service.SettingKeyBalanceLowNotifyEnabled:        "false",
			service.SettingPaymentVisibleMethodAlipayEnabled: "false",
			service.SettingPaymentVisibleMethodWxpayEnabled:  "false",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"promo_code_enabled": false,
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "true", repo.values[service.SettingKeyAllowUngroupedKeyScheduling])
	require.Equal(t, "true", repo.values[service.SettingKeyBackendModeEnabled])

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, data["allow_ungrouped_key_scheduling"])
	require.Equal(t, true, data["backend_mode_enabled"])
}

func TestSettingHandler_UpdateSettings_PersistsPaymentVisibleMethodsAndAdvancedScheduler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeyPromoCodeEnabled: "true",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"promo_code_enabled":                    true,
		"payment_visible_method_alipay_source":  "easypay",
		"payment_visible_method_wxpay_source":   "wxpay",
		"payment_visible_method_alipay_enabled": true,
		"payment_visible_method_wxpay_enabled":  false,
		"openai_advanced_scheduler_enabled":     true,
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.VisibleMethodSourceEasyPayAlipay, repo.values[service.SettingPaymentVisibleMethodAlipaySource])
	require.Equal(t, service.VisibleMethodSourceOfficialWechat, repo.values[service.SettingPaymentVisibleMethodWxpaySource])
	require.Equal(t, "true", repo.values[service.SettingPaymentVisibleMethodAlipayEnabled])
	require.Equal(t, "false", repo.values[service.SettingPaymentVisibleMethodWxpayEnabled])
	require.Equal(t, "true", repo.values["openai_advanced_scheduler_enabled"])

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, service.VisibleMethodSourceEasyPayAlipay, data["payment_visible_method_alipay_source"])
	require.Equal(t, service.VisibleMethodSourceOfficialWechat, data["payment_visible_method_wxpay_source"])
	require.Equal(t, true, data["payment_visible_method_alipay_enabled"])
	require.Equal(t, false, data["payment_visible_method_wxpay_enabled"])
	require.Equal(t, true, data["openai_advanced_scheduler_enabled"])
}

func TestSettingHandler_UpdateSettings_AllowsWeightedSchedulerWithoutLayeredValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeyPromoCodeEnabled: "true",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"promo_code_enabled":               false,
		"gateway_openai_ws_scheduler_mode": "weighted",
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "weighted", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerMode])
	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "weighted", data["gateway_openai_ws_scheduler_mode"])
}

func TestSettingHandler_UpdateSettings_PersistsStickyAndLayeredSchedulerSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{service.SettingKeyPromoCodeEnabled: "true"}}
	cfg := &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}}
	svc := service.NewSettingService(repo, cfg)
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"promo_code_enabled":                                          false,
		"gateway_sticky_openai_enabled":                               false,
		"gateway_sticky_gemini_enabled":                               true,
		"gateway_sticky_anthropic_enabled":                            false,
		"gateway_openai_ws_scheduler_mode":                            " Layered ",
		"gateway_openai_ws_scheduler_layered_error_penalty_threshold": 0.6,
		"gateway_openai_ws_scheduler_layered_error_penalty_value":     100,
		"gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier": 12,
		"gateway_openai_ws_scheduler_layered_ttft_penalty_value":      50,
		"gateway_openai_ws_scheduler_layered_probe_cooldown_seconds":  20,
		"gateway_openai_ws_scheduler_layered_probe_interval_seconds":  20,
		"gateway_openai_ws_scheduler_layered_probe_max_failures":      3,
		"gateway_openai_ws_scheduler_layered_probe_timeout_seconds":   15,
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "false", repo.values[service.SettingKeyGatewayStickyOpenAIEnabled])
	require.Equal(t, "true", repo.values[service.SettingKeyGatewayStickyGeminiEnabled])
	require.Equal(t, "false", repo.values[service.SettingKeyGatewayStickyAnthropicEnabled])
	require.Equal(t, "layered", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerMode])
	require.Equal(t, "0.6", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerLayeredErrorPenaltyThreshold])
	require.Equal(t, "100", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerLayeredErrorPenaltyValue])
	require.Equal(t, "12", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerLayeredTTFTPenaltyMultiplier])
	require.Equal(t, "50", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerLayeredTTFTPenaltyValue])
	require.Equal(t, "20", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeCooldownSeconds])
	require.Equal(t, "20", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeIntervalSeconds])
	require.Equal(t, "3", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeMaxFailures])
	require.Equal(t, "15", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerLayeredProbeTimeoutSeconds])
	require.False(t, cfg.Gateway.Sticky.OpenAI.Enabled)
	require.True(t, cfg.Gateway.Sticky.Gemini.Enabled)
	require.False(t, cfg.Gateway.Sticky.Anthropic.Enabled)
	require.Equal(t, "layered", cfg.Gateway.OpenAIWS.SchedulerMode)
}

func TestSettingHandler_UpdateSettings_NormalizesConfiguredSchedulerModeWhenRequestOmitsMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{service.SettingKeyPromoCodeEnabled: "true"}}
	cfg := &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}}
	cfg.Gateway.OpenAIWS.SchedulerMode = " Layered "
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold = 0.3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue = 100
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier = 3
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue = 50
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 60
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 30
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures = 3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds = 15
	svc := service.NewSettingService(repo, cfg)
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	rawBody, err := json.Marshal(map[string]any{"promo_code_enabled": false})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "layered", repo.values[service.SettingKeyGatewayOpenAIWSSchedulerMode])
	require.Equal(t, "layered", cfg.Gateway.OpenAIWS.SchedulerMode)
}

func TestSettingHandler_UpdateSettings_PreservesLegacyBlankPaymentVisibleMethodSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeyPromoCodeEnabled:               "true",
			service.SettingPaymentVisibleMethodAlipayEnabled: "true",
			service.SettingPaymentVisibleMethodAlipaySource:  "",
			service.SettingPaymentVisibleMethodWxpayEnabled:  "false",
			service.SettingPaymentVisibleMethodWxpaySource:   "",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"promo_code_enabled": false,
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "", repo.values[service.SettingPaymentVisibleMethodAlipaySource])
	require.Equal(t, "true", repo.values[service.SettingPaymentVisibleMethodAlipayEnabled])
}

func TestSettingHandler_UpdateSettings_PersistsExplicitFalseOIDCCompatibilityFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeyPromoCodeEnabled:               "true",
			service.SettingKeyOIDCConnectEnabled:             "true",
			service.SettingKeyOIDCConnectProviderName:        "OIDC",
			service.SettingKeyOIDCConnectClientID:            "oidc-client",
			service.SettingKeyOIDCConnectClientSecret:        "oidc-secret",
			service.SettingKeyOIDCConnectIssuerURL:           "https://issuer.example.com",
			service.SettingKeyOIDCConnectAuthorizeURL:        "https://issuer.example.com/auth",
			service.SettingKeyOIDCConnectTokenURL:            "https://issuer.example.com/token",
			service.SettingKeyOIDCConnectUserInfoURL:         "https://issuer.example.com/userinfo",
			service.SettingKeyOIDCConnectJWKSURL:             "https://issuer.example.com/jwks",
			service.SettingKeyOIDCConnectScopes:              "openid email profile",
			service.SettingKeyOIDCConnectRedirectURL:         "https://example.com/api/v1/auth/oauth/oidc/callback",
			service.SettingKeyOIDCConnectFrontendRedirectURL: "/auth/oidc/callback",
			service.SettingKeyOIDCConnectTokenAuthMethod:     "client_secret_post",
			service.SettingKeyOIDCConnectUsePKCE:             "true",
			service.SettingKeyOIDCConnectValidateIDToken:     "true",
			service.SettingKeyOIDCConnectAllowedSigningAlgs:  "RS256",
			service.SettingKeyOIDCConnectClockSkewSeconds:    "120",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"promo_code_enabled":                true,
		"oidc_connect_enabled":              true,
		"oidc_connect_use_pkce":             false,
		"oidc_connect_validate_id_token":    false,
		"oidc_connect_allowed_signing_algs": "",
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "false", repo.values[service.SettingKeyOIDCConnectUsePKCE])
	require.Equal(t, "false", repo.values[service.SettingKeyOIDCConnectValidateIDToken])

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, data["oidc_connect_use_pkce"])
	require.Equal(t, false, data["oidc_connect_validate_id_token"])
}

func TestSettingHandler_UpdateSettings_DoesNotSolidifyImplicitOIDCSecurityDefaultsOnLegacyUpgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeyPromoCodeEnabled:                "true",
			service.SettingKeyOIDCConnectEnabled:              "true",
			service.SettingKeyOIDCConnectProviderName:         "OIDC",
			service.SettingKeyOIDCConnectClientID:             "oidc-client",
			service.SettingKeyOIDCConnectClientSecret:         "oidc-secret",
			service.SettingKeyOIDCConnectIssuerURL:            "https://issuer.example.com",
			service.SettingKeyOIDCConnectAuthorizeURL:         "https://issuer.example.com/auth",
			service.SettingKeyOIDCConnectTokenURL:             "https://issuer.example.com/token",
			service.SettingKeyOIDCConnectUserInfoURL:          "https://issuer.example.com/userinfo",
			service.SettingKeyOIDCConnectJWKSURL:              "https://issuer.example.com/jwks",
			service.SettingKeyOIDCConnectScopes:               "openid email profile",
			service.SettingKeyOIDCConnectRedirectURL:          "https://example.com/api/v1/auth/oauth/oidc/callback",
			service.SettingKeyOIDCConnectFrontendRedirectURL:  "/auth/oidc/callback",
			service.SettingKeyOIDCConnectTokenAuthMethod:      "client_secret_post",
			service.SettingKeyOIDCConnectAllowedSigningAlgs:   "RS256",
			service.SettingKeyOIDCConnectClockSkewSeconds:     "120",
			service.SettingKeyOIDCConnectRequireEmailVerified: "false",
			service.SettingKeyOIDCConnectUserInfoEmailPath:    "",
			service.SettingKeyOIDCConnectUserInfoIDPath:       "",
			service.SettingKeyOIDCConnectUserInfoUsernamePath: "",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{
		Default: config.DefaultConfig{UserConcurrency: 5},
		OIDC: config.OIDCConnectConfig{
			Enabled:             true,
			ProviderName:        "OIDC",
			ClientID:            "oidc-client",
			ClientSecret:        "oidc-secret",
			IssuerURL:           "https://issuer.example.com",
			AuthorizeURL:        "https://issuer.example.com/auth",
			TokenURL:            "https://issuer.example.com/token",
			UserInfoURL:         "https://issuer.example.com/userinfo",
			JWKSURL:             "https://issuer.example.com/jwks",
			Scopes:              "openid email profile",
			RedirectURL:         "https://example.com/api/v1/auth/oauth/oidc/callback",
			FrontendRedirectURL: "/auth/oidc/callback",
			TokenAuthMethod:     "client_secret_post",
			UsePKCE:             true,
			ValidateIDToken:     true,
			AllowedSigningAlgs:  "RS256",
			ClockSkewSeconds:    120,
		},
	})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"promo_code_enabled":   true,
		"oidc_connect_enabled": true,
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "false", repo.values[service.SettingKeyOIDCConnectUsePKCE])
	require.Equal(t, "false", repo.values[service.SettingKeyOIDCConnectValidateIDToken])
}

func TestSettingHandler_UpdateSettings_RejectsInvalidPaymentVisibleMethodSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeyPromoCodeEnabled: "true",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"promo_code_enabled":                   true,
		"payment_visible_method_alipay_source": "bogus",
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotContains(t, repo.values, service.SettingPaymentVisibleMethodAlipaySource)
}

func TestSettingHandler_UpdateSettings_DoesNotPersistPartialSystemSettingsWhenAuthSourceDefaultsFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &failingAuthSourceSettingsRepoStub{
		values: map[string]string{
			service.SettingKeyRegistrationEnabled:                 "false",
			service.SettingKeyPromoCodeEnabled:                    "true",
			service.SettingKeyAuthSourceDefaultEmailBalance:       "9.5",
			service.SettingKeyAuthSourceDefaultEmailConcurrency:   "8",
			service.SettingKeyAuthSourceDefaultEmailSubscriptions: `[{"group_id":31,"validity_days":15}]`,
		},
		err: errors.New("write auth source defaults failed"),
	}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil)

	body := map[string]any{
		"registration_enabled":              true,
		"promo_code_enabled":                true,
		"auth_source_default_email_balance": 12.75,
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, "false", repo.values[service.SettingKeyRegistrationEnabled])
	require.Equal(t, "9.5", repo.values[service.SettingKeyAuthSourceDefaultEmailBalance])
}

func TestDiffSettings_IncludesAuthSourceDefaultsAndForceEmail(t *testing.T) {
	changed := diffSettings(
		&service.SystemSettings{},
		&service.SystemSettings{},
		&service.AuthSourceDefaultSettings{
			Email: service.ProviderDefaultGrantSettings{
				Balance:          0,
				Concurrency:      5,
				Subscriptions:    nil,
				GrantOnSignup:    true,
				GrantOnFirstBind: false,
			},
			ForceEmailOnThirdPartySignup: false,
		},
		&service.AuthSourceDefaultSettings{
			Email: service.ProviderDefaultGrantSettings{
				Balance:          12.5,
				Concurrency:      7,
				Subscriptions:    []service.DefaultSubscriptionSetting{{GroupID: 21, ValidityDays: 30}},
				GrantOnSignup:    false,
				GrantOnFirstBind: true,
			},
			ForceEmailOnThirdPartySignup: true,
		},
		UpdateSettingsRequest{},
	)

	require.Contains(t, changed, "auth_source_default_email_balance")
	require.Contains(t, changed, "auth_source_default_email_concurrency")
	require.Contains(t, changed, "auth_source_default_email_subscriptions")
	require.Contains(t, changed, "auth_source_default_email_grant_on_signup")
	require.Contains(t, changed, "auth_source_default_email_grant_on_first_bind")
	require.Contains(t, changed, "force_email_on_third_party_signup")
}
