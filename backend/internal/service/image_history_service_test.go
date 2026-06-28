package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type imageHistoryRepoStub struct {
	UsageLogRepository
	logs       []UsageLog
	detail     *UsageLogDetail
	details    map[int64]*UsageLogDetail
	gotUserID  int64
	gotParams  pagination.PaginationParams
	gotFilters ImageHistoryListFilters
	gotDetail  int64
	gotByID    int64
}

func stringPtr(value string) *string { return &value }

func (s *imageHistoryRepoStub) ListImageHistoryByUser(ctx context.Context, userID int64, params pagination.PaginationParams, filters ImageHistoryListFilters) ([]UsageLog, *pagination.PaginationResult, error) {
	s.gotUserID = userID
	s.gotParams = params
	s.gotFilters = filters
	return s.logs, &pagination.PaginationResult{Total: int64(len(s.logs)), Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (s *imageHistoryRepoStub) GetByID(ctx context.Context, id int64) (*UsageLog, error) {
	s.gotByID = id
	for i := range s.logs {
		if s.logs[i].ID == id {
			return &s.logs[i], nil
		}
	}
	return nil, ErrUsageLogNotFound
}

func (s *imageHistoryRepoStub) GetDetailByUsageLogID(ctx context.Context, usageLogID int64) (*UsageLogDetail, error) {
	s.gotDetail = usageLogID
	if s.details != nil {
		if detail, ok := s.details[usageLogID]; ok {
			return detail, nil
		}
		return nil, ErrUsageLogDetailNotFound
	}
	if s.detail == nil {
		return nil, ErrUsageLogDetailNotFound
	}
	return s.detail, nil
}

func TestImageHistoryServiceList_IncludesPromptAndDuration(t *testing.T) {
	t.Parallel()

	durationMs := 2140
	repo := &imageHistoryRepoStub{
		logs: []UsageLog{{
			ID:              51,
			UserID:          7,
			APIKeyID:        4,
			Model:           "gpt-image-2",
			ImageCount:      1,
			DurationMs:      &durationMs,
			InboundEndpoint: stringPtr("/v1/images/generations"),
			CreatedAt:       time.Date(2026, 4, 26, 8, 0, 0, 0, time.UTC),
		}},
		details: map[int64]*UsageLogDetail{
			51: {
				RequestHeaders: "Content-Type: application/json\n",
				RequestBody:    `{"prompt":"draw a compact workbench"}`,
			},
		},
	}
	svc := NewImageHistoryService(repo)

	out, _, err := svc.List(context.Background(), 7, ImageHistoryListQuery{})

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "draw a compact workbench", out[0].Prompt)
	require.NotNil(t, out[0].DurationMs)
	require.Equal(t, 2140, *out[0].DurationMs)
}

func TestImageHistoryServiceList_IgnoresMissingDetailForPrompt(t *testing.T) {
	t.Parallel()

	repo := &imageHistoryRepoStub{logs: []UsageLog{{
		ID:              52,
		UserID:          7,
		APIKeyID:        4,
		Model:           "gpt-image-2",
		ImageCount:      1,
		InboundEndpoint: stringPtr("/v1/images/generations"),
		CreatedAt:       time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC),
	}}}
	svc := NewImageHistoryService(repo)

	out, _, err := svc.List(context.Background(), 7, ImageHistoryListQuery{})

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Empty(t, out[0].Prompt)
	require.Nil(t, out[0].DurationMs)
}

func TestImageHistoryServiceGetDetail_IncludesDuration(t *testing.T) {
	t.Parallel()

	durationMs := 987
	repo := &imageHistoryRepoStub{
		logs: []UsageLog{{
			ID:              53,
			UserID:          7,
			APIKeyID:        4,
			Model:           "gpt-image-2",
			ImageCount:      1,
			DurationMs:      &durationMs,
			InboundEndpoint: stringPtr("/v1/images/generations"),
		}},
		detail: &UsageLogDetail{
			RequestHeaders: "Content-Type: application/json\n",
			RequestBody:    `{"prompt":"draw a fast result"}`,
			ResponseBody:   `{"created":1,"data":[{"b64_json":"QUJD"}]}`,
		},
	}
	svc := NewImageHistoryService(repo)

	detail, err := svc.GetDetail(context.Background(), 7, 53)

	require.NoError(t, err)
	require.NotNil(t, detail.DurationMs)
	require.Equal(t, 987, *detail.DurationMs)
}

func TestImageHistoryServiceList_MapsImageModesAndStatuses(t *testing.T) {
	t.Parallel()

	repo := &imageHistoryRepoStub{logs: []UsageLog{
		{
			ID:              11,
			UserID:          7,
			APIKeyID:        3,
			Model:           "gpt-image-2",
			ImageCount:      2,
			ImageSize:       stringPtr("1024x1024"),
			InboundEndpoint: stringPtr("/v1/images/generations"),
			ActualCost:      0.42,
			CreatedAt:       time.Date(2026, 4, 23, 8, 0, 0, 0, time.UTC),
			APIKey:          &APIKey{Name: "main-key", Key: "sk-test-1234567890"},
		},
		{
			ID:              12,
			UserID:          7,
			APIKeyID:        4,
			Model:           "gpt-image-2",
			ImageCount:      0,
			InboundEndpoint: stringPtr("/v1/images/edits"),
			CreatedAt:       time.Date(2026, 4, 23, 9, 0, 0, 0, time.UTC),
			APIKey:          &APIKey{Name: "edit-key", Key: "sk-edit-abcdef123456"},
		},
	}}
	svc := NewImageHistoryService(repo)

	out, page, err := svc.List(context.Background(), 7, ImageHistoryListQuery{APIKeyID: 9, Mode: "edit", Status: "error", Page: 2, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(7), repo.gotUserID)
	require.Equal(t, int64(2), page.Total)
	require.Len(t, out, 2)
	require.Equal(t, int64(9), repo.gotFilters.APIKeyID)
	require.Equal(t, "edit", repo.gotFilters.Mode)
	require.Equal(t, "error", repo.gotFilters.Status)
	require.Equal(t, 2, repo.gotParams.Page)
	require.Equal(t, 10, repo.gotParams.PageSize)
	require.Equal(t, ImageHistoryModeGenerate, out[0].Mode)
	require.Equal(t, ImageHistoryStatusSuccess, out[0].Status)
	require.Equal(t, "main-key", out[0].APIKeyName)
	require.Contains(t, out[0].APIKeyMasked, "...")
	require.Equal(t, "1024x1024", out[0].ImageSize)
	require.Equal(t, ImageHistoryModeEdit, out[1].Mode)
	require.Equal(t, ImageHistoryStatusError, out[1].Status)
}

func TestImageHistoryServiceGetDetail_ParsesJSONGenerateRequestAndResponse(t *testing.T) {
	t.Parallel()

	repo := &imageHistoryRepoStub{
		logs: []UsageLog{{
			ID:              31,
			UserID:          7,
			APIKeyID:        4,
			Model:           "gpt-image-2",
			ImageCount:      1,
			InboundEndpoint: stringPtr("/v1/images/generations"),
			APIKey:          &APIKey{Name: "json-key", Key: "sk-json-abcdef123456"},
		}},
		detail: &UsageLogDetail{
			RequestHeaders: "Content-Type: application/json\n",
			RequestBody:    `{"prompt":"draw a neon fox","size":"1536x1024","quality":"high","background":"transparent","output_format":"png","moderation":"low","n":2}`,
			ResponseBody:   `{"created":1,"data":[{"b64_json":"QUJD","revised_prompt":"draw a neon fox"}]}`,
		},
	}
	svc := NewImageHistoryService(repo)

	detail, err := svc.GetDetail(context.Background(), 7, 31)
	require.NoError(t, err)
	require.Equal(t, int64(31), repo.gotByID)
	require.Equal(t, int64(31), repo.gotDetail)
	require.Equal(t, ImageHistoryModeGenerate, detail.Mode)
	require.Equal(t, ImageHistoryStatusSuccess, detail.Status)
	require.Equal(t, "draw a neon fox", detail.Prompt)
	require.Equal(t, "1536x1024", detail.Size)
	require.Equal(t, "high", detail.Quality)
	require.Equal(t, "transparent", detail.Background)
	require.Equal(t, "png", detail.OutputFormat)
	require.Equal(t, "low", detail.Moderation)
	require.Equal(t, 2, detail.N)
	require.Len(t, detail.Images, 1)
	require.Equal(t, "data:image/png;base64,QUJD", detail.Images[0].DataURL)
	require.Equal(t, "draw a neon fox", detail.Images[0].RevisedPrompt)
	require.False(t, detail.HadSourceImage)
	require.False(t, detail.HadMask)
	require.False(t, detail.Replay.RequiresSourceImageUpload)
	require.False(t, detail.Replay.RequiresMaskUpload)
}

func TestImageHistoryServiceGetDetail_UsesOutputFormatForImageMime(t *testing.T) {
	t.Parallel()

	repo := &imageHistoryRepoStub{
		logs: []UsageLog{{
			ID:              32,
			UserID:          7,
			APIKeyID:        4,
			Model:           "gpt-image-2",
			ImageCount:      1,
			InboundEndpoint: stringPtr("/v1/images/generations"),
		}},
		detail: &UsageLogDetail{
			RequestHeaders: "Content-Type: application/json\n",
			RequestBody:    `{"prompt":"draw a skyline","output_format":"jpeg"}`,
			ResponseBody:   `{"created":1,"data":[{"b64_json":"QUJD"}]}`,
		},
	}
	svc := NewImageHistoryService(repo)

	detail, err := svc.GetDetail(context.Background(), 7, 32)
	require.NoError(t, err)
	require.Len(t, detail.Images, 1)
	require.Equal(t, "data:image/jpeg;base64,QUJD", detail.Images[0].DataURL)
}

func TestImageHistoryServiceGetDetail_ParsesMultipartEditAndMarksReuploads(t *testing.T) {
	t.Parallel()

	repo := &imageHistoryRepoStub{
		logs: []UsageLog{{
			ID:              21,
			UserID:          7,
			APIKeyID:        4,
			Model:           "gpt-image-2",
			ImageCount:      1,
			InboundEndpoint: stringPtr("/v1/images/edits"),
		}},
		detail: &UsageLogDetail{
			RequestHeaders: "Content-Type: multipart/form-data; boundary=abc\n",
			RequestBody:    "--abc\r\nContent-Disposition: form-data; name=\"prompt\"\r\n\r\nmake the sky pink\r\n--abc\r\nContent-Disposition: form-data; name=\"size\"\r\n\r\n1536x1024\r\n--abc\r\nContent-Disposition: form-data; name=\"image\"; filename=\"src.png\"\r\nContent-Type: image/png\r\n\r\npng-bytes\r\n--abc\r\nContent-Disposition: form-data; name=\"mask\"; filename=\"mask.png\"\r\nContent-Type: image/png\r\n\r\nmask-bytes\r\n--abc--\r\n",
			ResponseBody:   `{"created":1,"data":[{"b64_json":"QUJD","revised_prompt":"make the sky pink"}]}`,
		},
	}
	svc := NewImageHistoryService(repo)

	detail, err := svc.GetDetail(context.Background(), 7, 21)
	require.NoError(t, err)
	require.Equal(t, ImageHistoryModeEdit, detail.Mode)
	require.Equal(t, "make the sky pink", detail.Prompt)
	require.True(t, detail.HadSourceImage)
	require.True(t, detail.HadMask)
	require.True(t, detail.Replay.RequiresSourceImageUpload)
	require.True(t, detail.Replay.RequiresMaskUpload)
	require.Equal(t, "data:image/png;base64,QUJD", detail.Images[0].DataURL)
}

func TestImageHistoryServiceGetDetail_ParsesMultipartWithoutFilenameAndBrokenContentType(t *testing.T) {
	t.Parallel()

	repo := &imageHistoryRepoStub{
		logs: []UsageLog{{
			ID:              23,
			UserID:          7,
			APIKeyID:        4,
			Model:           "gpt-image-2",
			ImageCount:      1,
			InboundEndpoint: stringPtr("/v1/images/edits"),
		}},
		detail: &UsageLogDetail{
			RequestHeaders: "Content-Type: garbage\n",
			RequestBody:    "--abc\r\nContent-Disposition: form-data; name=\"prompt\"\r\n\r\nrepair this\r\n--abc\r\nContent-Disposition: form-data; name=\"image\"\r\nContent-Type: image/png\r\n\r\npng-bytes\r\n--abc\r\nContent-Disposition: form-data; name=\"mask\"\r\nContent-Type: image/png\r\n\r\nmask-bytes\r\n--abc\r\nContent-Disposition: form-data; name=\"output_format\"\r\n\r\nwebp\r\n--abc--\r\n",
			ResponseBody:   `{"created":1,"data":[{"b64_json":"QUJD"}]}`,
		},
	}
	svc := NewImageHistoryService(repo)

	detail, err := svc.GetDetail(context.Background(), 7, 23)
	require.NoError(t, err)
	require.Equal(t, "repair this", detail.Prompt)
	require.Equal(t, "webp", detail.OutputFormat)
	require.True(t, detail.HadSourceImage)
	require.True(t, detail.HadMask)
	require.Equal(t, "data:image/webp;base64,QUJD", detail.Images[0].DataURL)
}

func TestImageHistoryServiceGetDetail_ParsesMultipartMetadataSnapshot(t *testing.T) {
	t.Parallel()

	repo := &imageHistoryRepoStub{
		logs: []UsageLog{{
			ID:              24,
			UserID:          7,
			APIKeyID:        4,
			Model:           "gpt-image-2",
			RequestedModel:  "gpt-image-2",
			ImageCount:      1,
			InboundEndpoint: stringPtr("/v1/images/edits"),
		}},
		detail: &UsageLogDetail{
			RequestHeaders: "Content-Type: multipart/form-data; boundary=abc\n",
			RequestBody:    `{"model":"gpt-image-2","prompt":"repair this","size":"1536x1024","quality":"high","background":"transparent","output_format":"webp","moderation":"low","n":2,"had_source_image":true,"had_mask":true}`,
			ResponseBody:   `{"created":1,"data":[{"b64_json":"QUJD","revised_prompt":"repair this"}]}`,
		},
	}
	svc := NewImageHistoryService(repo)

	detail, err := svc.GetDetail(context.Background(), 7, 24)
	require.NoError(t, err)
	require.Equal(t, ImageHistoryModeEdit, detail.Mode)
	require.Equal(t, "repair this", detail.Prompt)
	require.Equal(t, "1536x1024", detail.Size)
	require.Equal(t, "high", detail.Quality)
	require.Equal(t, "transparent", detail.Background)
	require.Equal(t, "webp", detail.OutputFormat)
	require.Equal(t, "low", detail.Moderation)
	require.Equal(t, 2, detail.N)
	require.True(t, detail.HadSourceImage)
	require.True(t, detail.HadMask)
	require.True(t, detail.Replay.RequiresSourceImageUpload)
	require.True(t, detail.Replay.RequiresMaskUpload)
	require.Equal(t, "data:image/webp;base64,QUJD", detail.Images[0].DataURL)
}

func TestImageHistoryServiceGetDetail_RejectsNonImageUsageLog(t *testing.T) {
	t.Parallel()

	repo := &imageHistoryRepoStub{
		logs: []UsageLog{{
			ID:              41,
			UserID:          7,
			APIKeyID:        4,
			Model:           "gpt-4.1",
			InboundEndpoint: stringPtr("/v1/chat/completions"),
		}},
		detail: &UsageLogDetail{
			RequestHeaders: "Content-Type: application/json\n",
			RequestBody:    `{"messages":[{"role":"user","content":"hi"}]}`,
		},
	}
	svc := NewImageHistoryService(repo)

	detail, err := svc.GetDetail(context.Background(), 7, 41)
	require.Nil(t, detail)
	require.ErrorIs(t, err, ErrUsageLogNotFound)
	require.Equal(t, int64(41), repo.gotByID)
	require.Zero(t, repo.gotDetail)
}

func TestImageHistoryServiceGetDetail_EditReplayRequiresSourceUploadWithoutFileSnapshot(t *testing.T) {
	t.Parallel()

	repo := &imageHistoryRepoStub{
		logs: []UsageLog{{
			ID:              22,
			UserID:          7,
			APIKeyID:        4,
			Model:           "gpt-image-2",
			ImageCount:      1,
			InboundEndpoint: stringPtr("/v1/images/edits"),
		}},
		detail: &UsageLogDetail{
			RequestHeaders: "Content-Type: application/json\n",
			RequestBody:    `{"prompt":"recolor this","size":"1024x1024"}`,
			ResponseBody:   `{"created":1,"data":[{"b64_json":"QUJD"}]}`,
		},
	}
	svc := NewImageHistoryService(repo)

	detail, err := svc.GetDetail(context.Background(), 7, 22)
	require.NoError(t, err)
	require.Equal(t, ImageHistoryModeEdit, detail.Mode)
	require.False(t, detail.HadSourceImage)
	require.False(t, detail.HadMask)
	require.True(t, detail.Replay.RequiresSourceImageUpload)
	require.False(t, detail.Replay.RequiresMaskUpload)
}
