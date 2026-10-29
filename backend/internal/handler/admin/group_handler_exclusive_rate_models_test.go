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

type exclusiveRateModelsAdminStub struct {
	service.AdminService
	calls   int
	configs []service.GroupExclusiveRateModels
}

func (s *exclusiveRateModelsAdminStub) BatchSetGroupRateMultipliers(_ context.Context, _ int64, _ []service.GroupRateMultiplierInput, cfg ...service.GroupExclusiveRateModels) error {
	s.calls++
	s.configs = cfg
	return nil
}
func TestExclusiveRateModelsHandlerValidation(t *testing.T) {
	for _, body := range []string{
		`{"entries":[],"exclusive_rate_models":{"enabled":true,"models":[]}}`,
		`{"entries":[],"exclusive_rate_models":{"enabled":true,"models":["", "   "]}}`,
	} {
		svc := &exclusiveRateModelsAdminStub{}
		router := gin.New()
		router.PUT("/groups/:id/rate-multipliers", NewGroupHandler(svc, nil, nil).BatchSetGroupRateMultipliers)
		request := httptest.NewRequest(http.MethodPut, "/groups/2/rate-multipliers", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusBadRequest, response.Code)
		require.Contains(t, response.Body.String(), "至少选择一个模型")
		require.Zero(t, svc.calls, "invalid policy must leave existing storage untouched")
	}
	for _, body := range []string{`{"entries":[]}`, `{"entries":[],"exclusive_rate_models":{"enabled":false,"models":["client-A"]}}`} {
		svc := &exclusiveRateModelsAdminStub{}
		router := gin.New()
		router.PUT("/groups/:id/rate-multipliers", NewGroupHandler(svc, nil, nil).BatchSetGroupRateMultipliers)
		request := httptest.NewRequest(http.MethodPut, "/groups/2/rate-multipliers", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, 1, svc.calls)
		if body == `{"entries":[]}` {
			require.Empty(t, svc.configs)
		} else {
			require.Equal(t, []string{"client-A"}, svc.configs[0].Models)
		}
	}
}
