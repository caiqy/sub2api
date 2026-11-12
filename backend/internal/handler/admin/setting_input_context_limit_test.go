//go:build unit

package admin

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"testing"
)

type inputContextGroupReader struct{}

func (inputContextGroupReader) GetByID(_ context.Context, id int64) (*service.Group, error) {
	if id == 1 {
		return &service.Group{ID: 1}, nil
	}
	return nil, service.ErrGroupNotFound
}
func TestInputContextLimitAdminSettings(t *testing.T) {
	repo := &settingHandlerRepoStub{}
	svc := service.NewSettingService(repo, nil)
	svc.SetDefaultSubscriptionGroupReader(inputContextGroupReader{})
	h := &SettingHandler{settingService: svc}
	router := gin.New()
	router.GET("/settings/input-context-limit", h.GetInputContextLimitSettings)
	router.PUT("/settings/input-context-limit", h.UpdateInputContextLimitSettings)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/settings/input-context-limit", nil))
	require.Equal(t, 200, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"enabled":false`)
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"enabled":true,"group_limits":{"1":100}}`, 200},
		{`{"enabled":true,"group_limits":{"1":null}}`, 400},
		{`{"enabled":true,"group_limits":{"1":-1}}`, 400},
		{`{"enabled":true,"group_limits":{"1":1.5}}`, 400},
		{`{"enabled":true,"group_limits":{"1":2147483648}}`, 400},
		{`{"enabled":true,"group_limits":{"2":1}}`, 400},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/settings/input-context-limit", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		require.Equal(t, tc.status, rec.Code, rec.Body.String())
	}
	require.Equal(t, `{"enabled":true,"group_limits":{"1":100}}`, repo.values[service.SettingKeyInputContextLimit])
}
