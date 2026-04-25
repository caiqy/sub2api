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
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type gatewayRuntimeHandlerRepoStub struct {
	getValueByKey map[string]string
	getErrByKey   map[string]error
	setErr        error
	setCalls      []gatewayRuntimeHandlerRepoSetCall
}

type gatewayRuntimeHandlerRepoSetCall struct {
	key   string
	value string
}

func (s *gatewayRuntimeHandlerRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *gatewayRuntimeHandlerRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if err, ok := s.getErrByKey[key]; ok {
		return "", err
	}
	if value, ok := s.getValueByKey[key]; ok {
		return value, nil
	}
	return "", nil
}

func (s *gatewayRuntimeHandlerRepoStub) Set(ctx context.Context, key, value string) error {
	s.setCalls = append(s.setCalls, gatewayRuntimeHandlerRepoSetCall{key: key, value: value})
	return s.setErr
}

func (s *gatewayRuntimeHandlerRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *gatewayRuntimeHandlerRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *gatewayRuntimeHandlerRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *gatewayRuntimeHandlerRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type gatewayRuntimeHTTPUpstreamStub struct {
	invalidateCalls int
}

func (s *gatewayRuntimeHTTPUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	panic("unexpected Do call")
}

func (s *gatewayRuntimeHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	panic("unexpected DoWithTLS call")
}

func (s *gatewayRuntimeHTTPUpstreamStub) InvalidateIdleClients() {
	s.invalidateCalls++
}

type gatewayRuntimeEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type gatewayRuntimePayload struct {
	ResponseHeaderTimeout             int `json:"response_header_timeout"`
	StreamDataIntervalTimeout         int `json:"stream_data_interval_timeout"`
	UsageLogDetailRetentionLimit      int `json:"usage_log_detail_retention_limit"`
	ImageUsageLogDetailRetentionLimit int `json:"image_usage_log_detail_retention_limit"`
}

func newGatewayRuntimeHandlerTestConfig(responseHeaderTimeout, streamDataIntervalTimeout int) *config.Config {
	return &config.Config{
		Gateway: config.GatewayConfig{
			ResponseHeaderTimeout:             responseHeaderTimeout,
			StreamDataIntervalTimeout:         streamDataIntervalTimeout,
			UsageLogDetailRetentionLimit:      service.UsageLogDetailRetentionLimitDefault,
			ImageUsageLogDetailRetentionLimit: service.ImageUsageLogDetailRetentionLimitDefault,
		},
	}
}

func newGatewayRuntimeTestRouter(t *testing.T, repo *gatewayRuntimeHandlerRepoStub, cfg *config.Config, httpUpstream service.HTTPUpstream) (*gin.Engine, *service.SettingService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	settingService := service.ProvideSettingService(repo, nil, nil, cfg, httpUpstream)
	handler := NewSettingHandler(settingService, nil, nil, nil, nil, nil)

	router := gin.New()
	router.GET("/api/v1/admin/settings/gateway-runtime", handler.GetGatewayRuntimeSettings)
	router.PUT("/api/v1/admin/settings/gateway-runtime", handler.UpdateGatewayRuntimeSettings)
	return router, settingService
}

func decodeGatewayRuntimeResponse(t *testing.T, recorder *httptest.ResponseRecorder) gatewayRuntimeEnvelope {
	t.Helper()
	var resp gatewayRuntimeEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp
}

func TestSettingHandler_GetGatewayRuntimeSettings_FallsBackToCurrentConfig(t *testing.T) {
	repo := &gatewayRuntimeHandlerRepoStub{}
	cfg := newGatewayRuntimeHandlerTestConfig(120, 60)
	httpUpstream := &gatewayRuntimeHTTPUpstreamStub{}
	router, _ := newGatewayRuntimeTestRouter(t, repo, cfg, httpUpstream)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/gateway-runtime", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	resp := decodeGatewayRuntimeResponse(t, recorder)
	require.Equal(t, 0, resp.Code)

	var payload gatewayRuntimePayload
	require.NoError(t, json.Unmarshal(resp.Data, &payload))
	require.Equal(t, gatewayRuntimePayload{
		ResponseHeaderTimeout:             120,
		StreamDataIntervalTimeout:         60,
		UsageLogDetailRetentionLimit:      service.UsageLogDetailRetentionLimitDefault,
		ImageUsageLogDetailRetentionLimit: service.ImageUsageLogDetailRetentionLimitDefault,
	}, payload)
	require.Empty(t, repo.setCalls)
	require.Zero(t, httpUpstream.invalidateCalls)
}

func TestSettingHandler_UpdateGatewayRuntimeSettings_UpdatesConfigAndInvalidatesIdleClients(t *testing.T) {
	repo := &gatewayRuntimeHandlerRepoStub{}
	cfg := newGatewayRuntimeHandlerTestConfig(120, 60)
	httpUpstream := &gatewayRuntimeHTTPUpstreamStub{}
	router, _ := newGatewayRuntimeTestRouter(t, repo, cfg, httpUpstream)

	body := bytes.NewBufferString(`{"response_header_timeout":180,"stream_data_interval_timeout":0,"usage_log_detail_retention_limit":9,"image_usage_log_detail_retention_limit":0}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/gateway-runtime", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	resp := decodeGatewayRuntimeResponse(t, recorder)
	require.Equal(t, 0, resp.Code)

	var payload gatewayRuntimePayload
	require.NoError(t, json.Unmarshal(resp.Data, &payload))
	require.Equal(t, gatewayRuntimePayload{
		ResponseHeaderTimeout:             180,
		StreamDataIntervalTimeout:         0,
		UsageLogDetailRetentionLimit:      9,
		ImageUsageLogDetailRetentionLimit: 0,
	}, payload)
	require.Equal(t, 180, cfg.Gateway.ResponseHeaderTimeout)
	require.Equal(t, 0, cfg.Gateway.StreamDataIntervalTimeout)
	require.Equal(t, 9, cfg.Gateway.UsageLogDetailRetentionLimit)
	require.Equal(t, 0, cfg.Gateway.ImageUsageLogDetailRetentionLimit)
	require.Len(t, repo.setCalls, 1)
	require.Equal(t, service.SettingKeyGatewayRuntimeSettings, repo.setCalls[0].key)
	require.Equal(t, 1, httpUpstream.invalidateCalls)
}

func TestSettingHandler_UpdateGatewayRuntimeSettings_PreservesOmittedOptionalRuntimeFields(t *testing.T) {
	repo := &gatewayRuntimeHandlerRepoStub{}
	cfg := newGatewayRuntimeHandlerTestConfig(120, 60)
	cfg.Gateway.UsageLogDetailRetentionLimit = 11
	cfg.Gateway.ImageUsageLogDetailRetentionLimit = 22
	httpUpstream := &gatewayRuntimeHTTPUpstreamStub{}
	router, _ := newGatewayRuntimeTestRouter(t, repo, cfg, httpUpstream)

	body := bytes.NewBufferString(`{"response_header_timeout":180}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/gateway-runtime", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	resp := decodeGatewayRuntimeResponse(t, recorder)
	require.Equal(t, 0, resp.Code)

	var payload gatewayRuntimePayload
	require.NoError(t, json.Unmarshal(resp.Data, &payload))
	require.Equal(t, gatewayRuntimePayload{
		ResponseHeaderTimeout:             180,
		StreamDataIntervalTimeout:         60,
		UsageLogDetailRetentionLimit:      11,
		ImageUsageLogDetailRetentionLimit: 22,
	}, payload)
	require.Equal(t, 180, cfg.Gateway.ResponseHeaderTimeout)
	require.Equal(t, 60, cfg.Gateway.StreamDataIntervalTimeout)
	require.Equal(t, 11, cfg.Gateway.UsageLogDetailRetentionLimit)
	require.Equal(t, 22, cfg.Gateway.ImageUsageLogDetailRetentionLimit)
}

func TestSettingHandler_UpdateGatewayRuntimeSettings_ReturnsBadRequestForInvalidPayload(t *testing.T) {
	repo := &gatewayRuntimeHandlerRepoStub{}
	cfg := newGatewayRuntimeHandlerTestConfig(120, 60)
	httpUpstream := &gatewayRuntimeHTTPUpstreamStub{}
	router, _ := newGatewayRuntimeTestRouter(t, repo, cfg, httpUpstream)

	body := bytes.NewBufferString(`{"response_header_timeout":0,"stream_data_interval_timeout":60}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/gateway-runtime", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	resp := decodeGatewayRuntimeResponse(t, recorder)
	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Equal(t, "response_header_timeout must be positive", resp.Message)
	require.Empty(t, repo.setCalls)
	require.Equal(t, 120, cfg.Gateway.ResponseHeaderTimeout)
	require.Equal(t, 60, cfg.Gateway.StreamDataIntervalTimeout)
	require.Zero(t, httpUpstream.invalidateCalls)
}

func TestSettingHandler_UpdateGatewayRuntimeSettings_ReturnsInternalErrorWhenPersistFails(t *testing.T) {
	repo := &gatewayRuntimeHandlerRepoStub{setErr: errors.New("db write failed")}
	cfg := newGatewayRuntimeHandlerTestConfig(120, 60)
	httpUpstream := &gatewayRuntimeHTTPUpstreamStub{}
	router, _ := newGatewayRuntimeTestRouter(t, repo, cfg, httpUpstream)

	body := bytes.NewBufferString(`{"response_header_timeout":180,"stream_data_interval_timeout":90}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/gateway-runtime", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	resp := decodeGatewayRuntimeResponse(t, recorder)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.Len(t, repo.setCalls, 1)
	require.Equal(t, 120, cfg.Gateway.ResponseHeaderTimeout)
	require.Equal(t, 60, cfg.Gateway.StreamDataIntervalTimeout)
	require.Zero(t, httpUpstream.invalidateCalls)
}
