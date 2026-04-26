package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type imageHistoryHandlerRepoStub struct {
	service.UsageLogRepository
	logs       []service.UsageLog
	detail     *service.UsageLogDetail
	details    map[int64]*service.UsageLogDetail
	gotUserID  int64
	gotParams  pagination.PaginationParams
	gotFilters service.ImageHistoryListFilters
	gotByID    int64
	gotDetail  int64
}

type imageHistoryAPIKeyRepoStub struct {
	service.APIKeyRepository
	apiKeys  map[int64]*service.APIKey
	getByID  int64
	getByErr error
}

func imageHistoryStringPtr(value string) *string { return &value }

func newImageHistoryHandlerForTest(repo *imageHistoryHandlerRepoStub, apiKeyRepo *imageHistoryAPIKeyRepoStub) *ImageHistoryHandler {
	var apiKeyService *service.APIKeyService
	if apiKeyRepo != nil {
		apiKeyService = service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, nil)
	}
	return NewImageHistoryHandler(service.NewImageHistoryService(repo), apiKeyService)
}

func (s *imageHistoryHandlerRepoStub) ListImageHistoryByUser(_ context.Context, userID int64, params pagination.PaginationParams, filters service.ImageHistoryListFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	s.gotUserID = userID
	s.gotParams = params
	s.gotFilters = filters
	return s.logs, &pagination.PaginationResult{Total: int64(len(s.logs)), Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (s *imageHistoryHandlerRepoStub) GetByID(_ context.Context, id int64) (*service.UsageLog, error) {
	s.gotByID = id
	for i := range s.logs {
		if s.logs[i].ID == id {
			return &s.logs[i], nil
		}
	}
	return nil, service.ErrUsageLogNotFound
}

func (s *imageHistoryHandlerRepoStub) GetDetailByUsageLogID(_ context.Context, usageLogID int64) (*service.UsageLogDetail, error) {
	s.gotDetail = usageLogID
	if s.details != nil {
		if detail, ok := s.details[usageLogID]; ok {
			return detail, nil
		}
		return nil, service.ErrUsageLogDetailNotFound
	}
	if s.detail == nil {
		return nil, service.ErrUsageLogDetailNotFound
	}
	return s.detail, nil
}

func (s *imageHistoryAPIKeyRepoStub) GetByID(_ context.Context, id int64) (*service.APIKey, error) {
	s.getByID = id
	if s.getByErr != nil {
		return nil, s.getByErr
	}
	if apiKey, ok := s.apiKeys[id]; ok {
		cloned := *apiKey
		return &cloned, nil
	}
	return nil, service.ErrAPIKeyNotFound
}

func TestImageHistoryHandlerListSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	durationMs := 2140
	repo := &imageHistoryHandlerRepoStub{logs: []service.UsageLog{{
		ID:              31,
		UserID:          7,
		APIKeyID:        9,
		Model:           "gpt-image-2",
		RequestedModel:  "gpt-image-2",
		ImageCount:      1,
		ImageSize:       imageHistoryStringPtr("1024x1024"),
		ActualCost:      0.42,
		DurationMs:      &durationMs,
		InboundEndpoint: imageHistoryStringPtr("/v1/images/edits"),
		CreatedAt:       time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC),
		APIKey:          &service.APIKey{Name: "primary", Key: "sk-test-1234567890"},
	}}, details: map[int64]*service.UsageLogDetail{
		31: {
			RequestHeaders: "Content-Type: application/json\n",
			RequestBody:    `{"prompt":"draw a neon list row"}`,
		},
	}}
	h := newImageHistoryHandlerForTest(repo, &imageHistoryAPIKeyRepoStub{apiKeys: map[int64]*service.APIKey{
		9: {ID: 9, UserID: 7},
	}})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/history?tab=edit&status=success&api_key_id=9&page=2&page_size=1", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

	h.List(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(7), repo.gotUserID)
	require.Equal(t, int64(9), repo.gotFilters.APIKeyID)
	require.Equal(t, "edit", repo.gotFilters.Mode)
	require.Equal(t, "success", repo.gotFilters.Status)
	require.Equal(t, 2, repo.gotParams.Page)
	require.Equal(t, 1, repo.gotParams.PageSize)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []map[string]any `json:"items"`
			Total int64            `json:"total"`
			Page  int              `json:"page"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(1), resp.Data.Total)
	require.Equal(t, 2, resp.Data.Page)
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, float64(31), resp.Data.Items[0]["id"])
	require.Equal(t, "edit", resp.Data.Items[0]["mode"])
	require.Equal(t, "success", resp.Data.Items[0]["status"])
	require.Equal(t, "primary", resp.Data.Items[0]["api_key_name"])
	require.Equal(t, "draw a neon list row", resp.Data.Items[0]["prompt"])
	require.Equal(t, float64(2140), resp.Data.Items[0]["duration_ms"])
	require.NotContains(t, resp.Data.Items[0], "request_body")
}

func TestImageHistoryHandlerGetByIDUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImageHistoryHandlerForTest(&imageHistoryHandlerRepoStub{}, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/history/31", nil)
	c.Params = gin.Params{{Key: "id", Value: "31"}}

	h.GetByID(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, http.StatusUnauthorized, resp.Code)
	require.Equal(t, "User not authenticated", resp.Message)
}

func TestImageHistoryHandlerGetByIDSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	durationMs := 987
	repo := &imageHistoryHandlerRepoStub{
		logs: []service.UsageLog{{
			ID:              31,
			UserID:          7,
			APIKeyID:        9,
			Model:           "gpt-image-2",
			ImageCount:      1,
			DurationMs:      &durationMs,
			InboundEndpoint: imageHistoryStringPtr("/v1/images/generations"),
			CreatedAt:       time.Date(2026, 4, 23, 11, 0, 0, 0, time.UTC),
			APIKey:          &service.APIKey{Name: "primary", Key: "sk-test-1234567890"},
		}},
		detail: &service.UsageLogDetail{
			RequestHeaders: "Content-Type: application/json\n",
			RequestBody:    `{"prompt":"draw a neon fox","output_format":"png"}`,
			ResponseBody:   `{"created":1,"data":[{"b64_json":"QUJD","revised_prompt":"draw a neon fox"}]}`,
		},
	}
	h := newImageHistoryHandlerForTest(repo, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/history/31", nil)
	c.Params = gin.Params{{Key: "id", Value: "31"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

	h.GetByID(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(31), repo.gotByID)
	require.Equal(t, int64(31), repo.gotDetail)

	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, float64(31), resp.Data["id"])
	require.Equal(t, "generate", resp.Data["mode"])
	require.Equal(t, "success", resp.Data["status"])
	require.Equal(t, "draw a neon fox", resp.Data["prompt"])
	require.Equal(t, float64(987), resp.Data["duration_ms"])
	require.NotContains(t, resp.Data, "request_body")

	images, ok := resp.Data["images"].([]any)
	require.True(t, ok)
	require.Len(t, images, 1)

	replay, ok := resp.Data["replay"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, replay["requires_source_image_upload"])
	_, exists := resp.Data["response_body"]
	require.False(t, exists)
}

func TestImageHistoryHandlerListUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImageHistoryHandlerForTest(&imageHistoryHandlerRepoStub{}, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/history", nil)

	h.List(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestImageHistoryHandlerListRejectsInvalidAPIKeyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImageHistoryHandlerForTest(&imageHistoryHandlerRepoStub{}, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/history?api_key_id=abc", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

	h.List(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Equal(t, "Invalid api_key_id", resp.Message)
}

func TestImageHistoryHandlerListRejectsInvalidTab(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImageHistoryHandlerForTest(&imageHistoryHandlerRepoStub{}, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/history?tab=chat", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

	h.List(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestImageHistoryHandlerListRejectsInvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImageHistoryHandlerForTest(&imageHistoryHandlerRepoStub{}, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/history?status=pending", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

	h.List(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestImageHistoryHandlerListRejectsForeignAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImageHistoryHandlerForTest(&imageHistoryHandlerRepoStub{}, &imageHistoryAPIKeyRepoStub{apiKeys: map[int64]*service.APIKey{
		9: {ID: 9, UserID: 8},
	}})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/history?api_key_id=9", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

	h.List(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestImageHistoryHandlerListPropagatesAPIKeyLookupError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImageHistoryHandlerForTest(&imageHistoryHandlerRepoStub{}, &imageHistoryAPIKeyRepoStub{getByErr: errors.New("boom")})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/history?api_key_id=9", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

	h.List(c)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestImageHistoryHandlerGetByIDRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImageHistoryHandlerForTest(&imageHistoryHandlerRepoStub{}, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/images/history/bad", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

	h.GetByID(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Equal(t, "Invalid image history ID", resp.Message)
}
