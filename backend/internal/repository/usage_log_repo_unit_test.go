//go:build unit

package repository

import (
	"context"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSafeDateFormat(t *testing.T) {
	tests := []struct {
		name        string
		granularity string
		expected    string
	}{
		// 合法值
		{"hour", "hour", "YYYY-MM-DD HH24:00"},
		{"day", "day", "YYYY-MM-DD"},
		{"week", "week", "IYYY-IW"},
		{"month", "month", "YYYY-MM"},

		// 非法值回退到默认
		{"空字符串", "", "YYYY-MM-DD"},
		{"未知粒度 year", "year", "YYYY-MM-DD"},
		{"未知粒度 minute", "minute", "YYYY-MM-DD"},

		// 恶意字符串
		{"SQL 注入尝试", "'; DROP TABLE users; --", "YYYY-MM-DD"},
		{"带引号", "day'", "YYYY-MM-DD"},
		{"带括号", "day)", "YYYY-MM-DD"},
		{"Unicode", "日", "YYYY-MM-DD"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := safeDateFormat(tc.granularity)
			require.Equal(t, tc.expected, got, "safeDateFormat(%q)", tc.granularity)
		})
	}
}

func TestBuildUsageLogBatchInsertQuery_UsesConflictDoNothing(t *testing.T) {
	log := &service.UsageLog{
		UserID:       1,
		APIKeyID:     2,
		AccountID:    3,
		RequestID:    "req-batch-no-update",
		Model:        "gpt-5",
		InputTokens:  10,
		OutputTokens: 5,
		TotalCost:    1.2,
		ActualCost:   1.2,
		CreatedAt:    time.Now().UTC(),
	}
	prepared := prepareUsageLogInsert(log)

	query, _ := buildUsageLogBatchInsertQuery([]string{usageLogBatchKey(log.RequestID, log.APIKeyID)}, map[string]usageLogInsertPrepared{
		usageLogBatchKey(log.RequestID, log.APIKeyID): prepared,
	})

	require.Contains(t, query, "ON CONFLICT (request_id, api_key_id) DO NOTHING")
	require.NotContains(t, strings.ToUpper(query), "DO UPDATE")
}

func TestListUsageLogsWithPagination_UsesHasDetailWhenDetailTableExists(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM usage_logs WHERE user_id = $1")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT to_regclass('public.usage_log_details') IS NOT NULL")).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT "+usageLogListSelectColumns+" FROM usage_logs WHERE user_id = $1 ORDER BY id DESC LIMIT $2 OFFSET $3")).
		WithArgs(int64(7), 10, 0).
		WillReturnRows(sqlmock.NewRows(usageLogListRowColumns()).AddRow(usageLogListRowValues(true)...))

	logs, page, err := repo.listUsageLogsWithPagination(context.Background(), "WHERE user_id = $1", []any{int64(7)}, pagination.PaginationParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.True(t, logs[0].HasDetail)
	require.NotNil(t, page)
	require.Equal(t, int64(1), page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListUsageLogsWithPagination_DegradesWhenDetailTableMissing(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM usage_logs WHERE user_id = $1")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT to_regclass('public.usage_log_details') IS NOT NULL")).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT "+usageLogSelectColumns+", FALSE AS has_detail FROM usage_logs WHERE user_id = $1 ORDER BY id DESC LIMIT $2 OFFSET $3")).
		WithArgs(int64(7), 10, 0).
		WillReturnRows(sqlmock.NewRows(usageLogListRowColumns()).AddRow(usageLogListRowValues(false)...))

	logs, page, err := repo.listUsageLogsWithPagination(context.Background(), "WHERE user_id = $1", []any{int64(7)}, pagination.PaginationParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.False(t, logs[0].HasDetail)
	require.NotNil(t, page)
	require.Equal(t, int64(1), page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListImageHistoryByUser_AppliesImageFilters(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	params := pagination.PaginationParams{Page: 1, PageSize: 10, SortBy: "created_at", SortOrder: pagination.SortOrderDesc}
	filters := service.ImageHistoryListFilters{
		APIKeyID: 11,
		Mode:     string(service.ImageHistoryModeEdit),
		Status:   string(service.ImageHistoryStatusError),
	}
	whereClause := "WHERE user_id = $1 AND (COALESCE(inbound_endpoint, '') LIKE '%/images/generations%' OR COALESCE(inbound_endpoint, '') LIKE '%/images/edits%') AND api_key_id = $2 AND COALESCE(inbound_endpoint, '') LIKE $3 AND image_count <= 0"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM usage_logs "+whereClause)).
		WithArgs(int64(7), int64(11), "%/images/edits%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT to_regclass('public.usage_log_details') IS NOT NULL")).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT "+usageLogListSelectColumns+" FROM usage_logs "+whereClause+" ORDER BY created_at DESC, id DESC LIMIT $4 OFFSET $5")).
		WithArgs(int64(7), int64(11), "%/images/edits%", 10, 0).
		WillReturnRows(sqlmock.NewRows(usageLogListRowColumns()))

	logs, page, err := repo.ListImageHistoryByUser(context.Background(), 7, params, filters)
	require.NoError(t, err)
	require.Empty(t, logs)
	require.NotNil(t, page)
	require.Zero(t, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func usageLogListRowColumns() []string {
	return []string{
		"id", "user_id", "api_key_id", "account_id", "request_id", "model", "requested_model", "upstream_model", "group_id", "subscription_id",
		"input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "cache_creation_5m_tokens", "cache_creation_1h_tokens",
		"image_output_tokens", "image_output_cost", "input_cost", "output_cost", "cache_creation_cost", "cache_read_cost", "total_cost", "actual_cost", "rate_multiplier",
		"account_rate_multiplier", "billing_type", "request_type", "stream", "openai_ws_mode", "duration_ms", "first_token_ms",
		"user_agent", "ip_address", "image_count", "image_size", "service_tier", "reasoning_effort",
		"inbound_endpoint", "upstream_endpoint", "cache_ttl_overridden", "channel_id", "model_mapping_chain", "billing_tier", "billing_mode", "account_stats_cost", "created_at", "has_detail",
	}
}

func usageLogListRowValues(hasDetail bool) []driver.Value {
	createdAt := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	return []driver.Value{
		int64(101), int64(7), int64(8), int64(9), "req-list", "claude-3", nil, nil, nil, nil,
		10, 20, 0, 0, 0, 0,
		0, 0.0, 0.1, 0.2, 0.0, 0.0, 0.3, 0.3, 1.0,
		nil, int16(0), int16(service.RequestTypeSync), false, false, nil, nil,
		nil, nil, 0, nil, nil, nil,
		nil, nil, false, nil, nil, nil, nil, nil, createdAt, hasDetail,
	}
}
