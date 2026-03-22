//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type failedUsageLogRepoStub struct {
	UsageLogRepository
	createCalls     int
	createBestCalls int
	lastLog         *UsageLog
}

func (s *failedUsageLogRepoStub) CreateBestEffort(ctx context.Context, log *UsageLog) error {
	s.createBestCalls++
	s.lastLog = log
	return nil
}

func (s *failedUsageLogRepoStub) Create(ctx context.Context, log *UsageLog) (bool, error) {
	s.createCalls++
	s.lastLog = log
	return true, nil
}

func TestWriteFailedUsageLogBestEffort_CreatesZeroCostUsageLog(t *testing.T) {
	repo := &failedUsageLogRepoStub{}
	detail := &UsageLogDetailSnapshot{
		RequestHeaders:  "Authorization: Bearer test",
		RequestBody:     `{"model":"gpt-5.4"}`,
		ResponseHeaders: "Content-Type: application/json",
		ResponseBody:    `{"error":{"type":"upstream_error","message":"boom"}}`,
	}
	groupID := int64(11)
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "failed-client-req")

	WriteFailedUsageLogBestEffort(ctx, repo, &FailedUsageLogInput{
		APIKey:           &APIKey{ID: 101, GroupID: &groupID},
		User:             &User{ID: 202},
		Account:          &Account{ID: 303},
		Model:            "gpt-5.4",
		Stream:           false,
		InboundEndpoint:  "/v1/responses",
		UpstreamEndpoint: "/v1/responses",
		UserAgent:        "curl/8.0",
		IPAddress:        "127.0.0.1",
		DetailSnapshot:   detail,
		Duration:         time.Second,
	}, "service.test")

	require.Equal(t, 1, repo.createBestCalls)
	require.NotNil(t, repo.lastLog)
	require.Equal(t, int64(202), repo.lastLog.UserID)
	require.Equal(t, int64(101), repo.lastLog.APIKeyID)
	require.Equal(t, int64(303), repo.lastLog.AccountID)
	require.Equal(t, "client:failed-client-req", repo.lastLog.RequestID)
	require.Equal(t, "gpt-5.4", repo.lastLog.Model)
	require.Equal(t, 0, repo.lastLog.InputTokens)
	require.Equal(t, 0, repo.lastLog.OutputTokens)
	require.Equal(t, 0.0, repo.lastLog.TotalCost)
	require.Equal(t, 0.0, repo.lastLog.ActualCost)
	require.NotNil(t, repo.lastLog.InboundEndpoint)
	require.Equal(t, "/v1/responses", *repo.lastLog.InboundEndpoint)
	require.NotNil(t, repo.lastLog.UpstreamEndpoint)
	require.Equal(t, "/v1/responses", *repo.lastLog.UpstreamEndpoint)
	require.NotNil(t, repo.lastLog.UserAgent)
	require.Equal(t, "curl/8.0", *repo.lastLog.UserAgent)
	require.NotNil(t, repo.lastLog.IPAddress)
	require.Equal(t, "127.0.0.1", *repo.lastLog.IPAddress)
	require.NotNil(t, repo.lastLog.DetailSnapshot)
	require.Equal(t, detail.RequestBody, repo.lastLog.DetailSnapshot.RequestBody)
	require.Equal(t, detail.ResponseBody, repo.lastLog.DetailSnapshot.ResponseBody)
}
