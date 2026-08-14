//go:build unit

package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type pricingGroupAdminServiceStub struct {
	service.AdminService
	createInput *service.CreateGroupInput
	updateInput *service.UpdateGroupInput
}

func (s *pricingGroupAdminServiceStub) CreateGroup(_ context.Context, input *service.CreateGroupInput) (*service.Group, error) {
	s.createInput = input
	return &service.Group{ID: 1, Name: input.Name, Platform: input.Platform}, nil
}

func (s *pricingGroupAdminServiceStub) UpdateGroup(_ context.Context, _ int64, input *service.UpdateGroupInput) (*service.Group, error) {
	s.updateInput = input
	return &service.Group{ID: 1, Name: input.Name, Platform: service.PlatformGrok}, nil
}

func TestGroupHandlerForwardsPricingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payload := []byte(`{
		"name":"Grok media",
		"platform":"grok",
		"video_model_prices":{"grok-imagine-video-1.5":{"720p":0.14}},
		"search_price_per_1k":12.5,
		"audio_realtime_price_per_min":0.1,
		"audio_tts_price_per_million_chars":15,
		"audio_stt_price_per_hour":0.36
	}`)

	assertInput := func(t *testing.T, videoPrices map[string]map[string]float64, search, realtime, tts, stt *float64) {
		t.Helper()
		require.Equal(t, map[string]map[string]float64{
			"grok-imagine-video-1.5": {"720p": 0.14},
		}, videoPrices)
		require.NotNil(t, search)
		require.NotNil(t, realtime)
		require.NotNil(t, tts)
		require.NotNil(t, stt)
		require.InDelta(t, 12.5, *search, 1e-12)
		require.InDelta(t, 0.1, *realtime, 1e-12)
		require.InDelta(t, 15, *tts, 1e-12)
		require.InDelta(t, 0.36, *stt, 1e-12)
	}

	for _, tc := range []struct {
		name   string
		method string
		path   string
		assert func(*testing.T, *pricingGroupAdminServiceStub)
	}{
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/groups",
			assert: func(t *testing.T, svc *pricingGroupAdminServiceStub) {
				require.NotNil(t, svc.createInput)
				assertInput(t, svc.createInput.VideoModelPrices, svc.createInput.SearchPricePer1k, svc.createInput.AudioRealtimePricePerMin, svc.createInput.AudioTTSPricePerMillionChars, svc.createInput.AudioSTTPricePerHour)
			},
		},
		{
			name:   "update",
			method: http.MethodPut,
			path:   "/groups/1",
			assert: func(t *testing.T, svc *pricingGroupAdminServiceStub) {
				require.NotNil(t, svc.updateInput)
				assertInput(t, svc.updateInput.VideoModelPrices, svc.updateInput.SearchPricePer1k, svc.updateInput.AudioRealtimePricePerMin, svc.updateInput.AudioTTSPricePerMillionChars, svc.updateInput.AudioSTTPricePerHour)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &pricingGroupAdminServiceStub{}
			router := gin.New()
			handler := NewGroupHandler(svc, nil, nil)
			router.POST("/groups", handler.Create)
			router.PUT("/groups/:id", handler.Update)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, tc.path, bytes.NewReader(payload))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			tc.assert(t, svc)
		})
	}
}

func TestGroupHandlerDefaultsLongContextPricingEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &pricingGroupAdminServiceStub{}
	router := gin.New()
	router.POST("/groups", NewGroupHandler(svc, nil, nil).Create)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewBufferString(`{"name":"default long context","platform":"grok"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, svc.createInput)
	require.True(t, svc.createInput.LongContextPricingEnabled)
}

func TestGroupHandlerPreservesExplicitFalseLongContextPricingEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &pricingGroupAdminServiceStub{}
	router := gin.New()
	router.POST("/groups", NewGroupHandler(svc, nil, nil).Create)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewBufferString(`{"name":"disable long context","platform":"grok","long_context_pricing_enabled":false}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, svc.createInput)
	require.False(t, svc.createInput.LongContextPricingEnabled)
}
