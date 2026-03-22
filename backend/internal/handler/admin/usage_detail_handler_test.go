package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type usageDetailRepoStub struct {
	service.UsageLogRepository
	detail     *service.UsageLogDetail
	detailErr  error
	usageLogID int64
}

func (s *usageDetailRepoStub) GetDetailByUsageLogID(ctx context.Context, usageLogID int64) (*service.UsageLogDetail, error) {
	s.usageLogID = usageLogID
	if s.detailErr != nil {
		return nil, s.detailErr
	}
	return s.detail, nil
}

func newUsageDetailTestRouter(repo *usageDetailRepoStub, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(repo, nil, nil, nil)
	handler := NewUsageHandler(usageSvc, nil, nil, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if role != "admin" {
			response.Forbidden(c, "Forbidden")
			c.Abort()
			return
		}
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
		c.Set(string(middleware.ContextKeyUserRole), role)
		c.Next()
	})
	router.GET("/api/v1/admin/usage/:id/detail", handler.Detail)
	return router
}

func TestUsageHandlerDetailSuccess(t *testing.T) {
	repo := &usageDetailRepoStub{detail: &service.UsageLogDetail{
		UsageLogID:             123,
		RequestHeaders:         "req-h",
		RequestBody:            "req-b",
		UpstreamRequestHeaders: "upstream-req-h",
		UpstreamRequestBody:    "upstream-req-b",
		ResponseHeaders:        "resp-h",
		ResponseBody:           "resp-b",
		CreatedAt:              time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
	}}
	router := newUsageDetailTestRouter(repo, "admin")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/usage/123/detail", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(123), repo.usageLogID)

	var got response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, 0, got.Code)

	dataBytes, err := json.Marshal(got.Data)
	require.NoError(t, err)
	var detail map[string]any
	require.NoError(t, json.Unmarshal(dataBytes, &detail))
	require.Equal(t, float64(123), detail["usage_log_id"])
	require.Equal(t, "req-h", detail["request_headers"])
	require.Equal(t, "req-b", detail["request_body"])
	require.Equal(t, "upstream-req-h", detail["upstream_request_headers"])
	require.Equal(t, "upstream-req-b", detail["upstream_request_body"])
	require.Equal(t, "resp-h", detail["response_headers"])
	require.Equal(t, "resp-b", detail["response_body"])
	require.Equal(t, "2026-03-20T12:00:00Z", detail["created_at"])
}

func TestUsageHandlerDetailSuccess_LegacyEmptyUpstreamFieldsRemainCompatible(t *testing.T) {
	repo := &usageDetailRepoStub{detail: &service.UsageLogDetail{
		UsageLogID:      456,
		RequestHeaders:  "req-h",
		RequestBody:     "req-b",
		ResponseHeaders: "resp-h",
		ResponseBody:    "resp-b",
		CreatedAt:       time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC),
	}}
	router := newUsageDetailTestRouter(repo, "admin")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/usage/456/detail", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var got response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	dataBytes, err := json.Marshal(got.Data)
	require.NoError(t, err)
	var detail map[string]any
	require.NoError(t, json.Unmarshal(dataBytes, &detail))
	require.Equal(t, "", detail["upstream_request_headers"])
	require.Equal(t, "", detail["upstream_request_body"])
}

func TestUsageHandlerDetailNotFound(t *testing.T) {
	repo := &usageDetailRepoStub{detailErr: service.ErrUsageLogDetailNotFound}
	router := newUsageDetailTestRouter(repo, "admin")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/usage/123/detail", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	var got response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusNotFound, got.Code)
	require.Equal(t, "USAGE_LOG_DETAIL_NOT_FOUND", got.Reason)
	require.Equal(t, infraerrors.Message(service.ErrUsageLogDetailNotFound), got.Message)
	require.Contains(t, got.Message, strconv.Itoa(service.UsageLogDetailRetentionLimit))
}

func TestUsageHandlerDetailUsageLogMissing(t *testing.T) {
	repo := &usageDetailRepoStub{detailErr: service.ErrUsageLogNotFound}
	router := newUsageDetailTestRouter(repo, "admin")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/usage/123/detail", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "usage log not found")
}

func TestUsageHandlerDetailForbidden(t *testing.T) {
	repo := &usageDetailRepoStub{}
	router := newUsageDetailTestRouter(repo, "user")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/usage/123/detail", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Zero(t, repo.usageLogID)
}
