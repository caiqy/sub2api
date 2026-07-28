package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestListRequestDetailsIncludesModelTriplet(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	repo := &opsRepository{db: db}
	mock.ExpectQuery(`(?s)WITH combined AS.*SELECT COUNT\(1\) FROM combined`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)WITH combined AS.*requested_model.*upstream_model.*FROM combined`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"kind", "created_at", "request_id", "platform", "model", "requested_model", "upstream_model",
			"duration_ms", "status_code", "error_id", "phase", "severity", "message",
			"user_id", "api_key_id", "account_id", "group_id", "stream",
		}).AddRow(
			"error", time.Now(), "req_1", "openai", "gpt-5", "public-alias", "gpt-5.2",
			nil, 502, int64(8), "upstream", "P1", "upstream failed",
			int64(1), int64(2), int64(3), int64(4), false,
		))

	items, total, err := repo.ListRequestDetails(context.Background(), &service.OpsRequestDetailFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "public-alias", items[0].RequestedModel)
	require.Equal(t, "gpt-5", items[0].Model)
	require.Equal(t, "gpt-5.2", items[0].UpstreamModel)
	require.NoError(t, mock.ExpectationsWereMet())
}
