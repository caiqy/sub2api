package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func resetUsageLogDetailRetentionLimitsForRepositoryTest(t *testing.T) {
	t.Helper()
	oldNormal, oldImage := service.GetUsageLogDetailRetentionLimits()
	service.SetUsageLogDetailRetentionLimits(service.UsageLogDetailRetentionLimitDefault, service.ImageUsageLogDetailRetentionLimitDefault)
	t.Cleanup(func() { service.SetUsageLogDetailRetentionLimits(oldNormal, oldImage) })
}

func TestUsageLogDetailRepositoryCreate_WritesDetailTypeAndPrunesBothPools(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	createdAt := time.Now().UTC().Truncate(time.Second)
	repo := newUsageLogDetailRepositoryWithSQL(db)
	resetUsageLogDetailRetentionLimitsForRepositoryTest(t)
	service.SetUsageLogDetailRetentionLimits(3, 2)

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO usage_log_details (
			usage_log_id,
			detail_type,
			request_headers,
			request_body,
			upstream_request_headers,
			upstream_request_body,
			response_headers,
			response_body,
			upstream_response_headers,
			upstream_response_body,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`)).
		WithArgs(int64(123), string(service.UsageLogDetailTypeImage), "", "{}", "", "", "", "", "", "", createdAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM usage_log_details`).
		WithArgs(string(service.UsageLogDetailTypeNormal), 3).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`DELETE FROM usage_log_details`).
		WithArgs(string(service.UsageLogDetailTypeImage), 2).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Create(context.Background(), &service.UsageLogDetail{
		UsageLogID:  123,
		DetailType:  service.UsageLogDetailTypeImage,
		RequestBody: "{}",
		CreatedAt:   createdAt,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogDetailRepositoryCreate_DisabledDetailTypeSkipsInsertAndPurgesPool(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUsageLogDetailRepositoryWithSQL(db)
	resetUsageLogDetailRetentionLimitsForRepositoryTest(t)
	service.SetUsageLogDetailRetentionLimits(0, 2)

	mock.ExpectExec(`DELETE FROM usage_log_details`).
		WithArgs(string(service.UsageLogDetailTypeNormal)).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err = repo.Create(context.Background(), &service.UsageLogDetail{
		UsageLogID: 123,
		DetailType: service.UsageLogDetailTypeNormal,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogDetailRepositoryCreateBatch_AllDisabledPurgesWithoutInsert(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUsageLogDetailRepositoryWithSQL(db)
	resetUsageLogDetailRetentionLimitsForRepositoryTest(t)
	service.SetUsageLogDetailRetentionLimits(0, 0)

	mock.ExpectExec(`DELETE FROM usage_log_details`).
		WithArgs(string(service.UsageLogDetailTypeNormal)).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(`DELETE FROM usage_log_details`).
		WithArgs(string(service.UsageLogDetailTypeImage)).
		WillReturnResult(sqlmock.NewResult(0, 4))

	err = repo.CreateBatch(context.Background(), []*service.UsageLogDetail{
		{UsageLogID: 123, DetailType: service.UsageLogDetailTypeNormal},
		{UsageLogID: 124, DetailType: service.UsageLogDetailTypeImage},
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogDetailRepositoryPruneByDetailTypeLimit_ZeroDeletesPool(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUsageLogDetailRepositoryWithSQL(db)
	mock.ExpectExec(`DELETE FROM usage_log_details`).
		WithArgs(string(service.UsageLogDetailTypeImage)).
		WillReturnResult(sqlmock.NewResult(0, 4))

	err = repo.pruneDetailTypeToRecentLimit(context.Background(), service.UsageLogDetailTypeImage, 0)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogDetailRepositoryPruneToRecentLimit_ZeroDeletesNormalPool(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUsageLogDetailRepositoryWithSQL(db)
	mock.ExpectExec(`DELETE FROM usage_log_details`).
		WithArgs(string(service.UsageLogDetailTypeNormal)).
		WillReturnResult(sqlmock.NewResult(0, 5))

	err = repo.PruneToRecentLimit(context.Background(), 0)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
