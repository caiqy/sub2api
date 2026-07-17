package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type usageLogDetailRepoExecStub struct {
	execCalls int
	execFn    func(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (s *usageLogDetailRepoExecStub) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	s.execCalls++
	if s.execFn != nil {
		return s.execFn(ctx, query, args...)
	}
	return usageLogDetailRepoResult(1), nil
}

func (s *usageLogDetailRepoExecStub) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("unexpected query")
}

type usageLogDetailRepoResult int64

func (r usageLogDetailRepoResult) LastInsertId() (int64, error) { return int64(r), nil }
func (r usageLogDetailRepoResult) RowsAffected() (int64, error) { return int64(r), nil }

func TestUsageLogRepositoryCreateSingle_SkipsDetailPersistenceWhenDisabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUsageLogRepositoryWithSQL(nil, db)
	createdAt := time.Now().UTC().Truncate(time.Second)
	log := &service.UsageLog{
		UserID:       1,
		APIKeyID:     2,
		AccountID:    3,
		RequestID:    "req-tx-skip-detail",
		Model:        "claude-3",
		InputTokens:  10,
		OutputTokens: 20,
		TotalCost:    0.5,
		ActualCost:   0.5,
		CreatedAt:    createdAt,
		DetailSnapshot: (&service.UsageLogDetailSnapshot{
			RequestBody: `{"in_tx":true}`,
		}).Normalize(),
	}

	mock.ExpectQuery(`INSERT INTO usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(123, createdAt))

	inserted, err := repo.createSingle(context.Background(), db, log, false)
	require.NoError(t, err)
	require.True(t, inserted)
	require.Equal(t, int64(123), log.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryFlushCreateBatch_FallbackUsesOriginalRequestContextForMainInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUsageLogRepositoryWithSQL(nil, db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	log := &service.UsageLog{
		UserID:       1,
		APIKeyID:     2,
		AccountID:    3,
		RequestID:    "",
		Model:        "claude-3",
		InputTokens:  10,
		OutputTokens: 20,
		TotalCost:    0.5,
		ActualCost:   0.5,
		CreatedAt:    time.Now().UTC().Truncate(time.Second),
		DetailSnapshot: (&service.UsageLogDetailSnapshot{
			RequestBody: `{"fallback":true}`,
		}).Normalize(),
	}
	req := usageLogCreateRequest{
		ctx:      ctx,
		log:      log,
		prepared: prepareUsageLogInsert(log),
		resultCh: make(chan usageLogCreateResult, 1),
	}

	repo.flushCreateBatch(db, []usageLogCreateRequest{req})

	res := <-req.resultCh
	require.False(t, res.inserted)
	require.Error(t, res.err)
	require.True(t, service.IsUsageLogCreateNotPersisted(res.err))
	require.ErrorIs(t, res.err, context.Canceled)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogDetailRepositoryCreate_WrapsInsertError(t *testing.T) {
	insertErr := errors.New("insert boom")
	repo := newUsageLogDetailRepositoryWithSQL(&usageLogDetailRepoExecStub{
		execFn: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return nil, insertErr
		},
	})

	err := repo.Create(context.Background(), &service.UsageLogDetail{UsageLogID: 123})
	require.Error(t, err)
	require.ErrorIs(t, err, insertErr)
	require.ErrorContains(t, err, "insert usage log detail")
}

func TestUsageLogDetailRepositoryCreate_WrapsPruneError(t *testing.T) {
	pruneErr := errors.New("prune boom")
	repo := newUsageLogDetailRepositoryWithSQL(&usageLogDetailRepoExecStub{
		execFn: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			if query == "" {
				return usageLogDetailRepoResult(1), nil
			}
			if len(args) == 11 {
				return usageLogDetailRepoResult(1), nil
			}
			return nil, pruneErr
		},
	})

	err := repo.Create(context.Background(), &service.UsageLogDetail{UsageLogID: 123})
	require.Error(t, err)
	require.ErrorIs(t, err, pruneErr)
	require.ErrorContains(t, err, "prune usage log detail")
}

func TestUsageLogRepositoryFlushBestEffortBatch_PersistsDetailForInsertedRows(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUsageLogRepositoryWithSQL(nil, db)
	resetUsageLogDetailRetentionLimitsForRepositoryTest(t)
	createdAt := time.Now().UTC().Truncate(time.Second)
	imageEndpoint := "/v1/images/generations"
	log := &service.UsageLog{
		UserID:          1,
		APIKeyID:        2,
		AccountID:       3,
		RequestID:       "req-best-effort-detail",
		Model:           "claude-3",
		InputTokens:     10,
		OutputTokens:    20,
		TotalCost:       0.5,
		ActualCost:      0.5,
		InboundEndpoint: &imageEndpoint,
		CreatedAt:       createdAt,
		DetailSnapshot: (&service.UsageLogDetailSnapshot{
			RequestHeaders:         "Authorization: Bearer best-effort",
			RequestBody:            `{"request":"payload"}`,
			UpstreamRequestHeaders: "X-Upstream: value",
			UpstreamRequestBody:    `{"upstream":"payload"}`,
			ResponseHeaders:        "Content-Type: application/json",
			ResponseBody:           `{"ok":true}`,
		}).Normalize(),
	}
	mock.ExpectQuery(`INSERT INTO usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(123), createdAt))
	mock.ExpectExec(`INSERT INTO usage_log_details`).
		WithArgs(
			int64(123),
			string(service.UsageLogDetailTypeImage),
			"Authorization: Bearer best-effort",
			`{"request":"payload"}`,
			"X-Upstream: value",
			`{"upstream":"payload"}`,
			"Content-Type: application/json",
			`{"ok":true}`,
			"",
			"",
			createdAt,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM usage_log_details`).
		WithArgs(string(service.UsageLogDetailTypeNormal), service.UsageLogDetailRetentionLimitDefault).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`DELETE FROM usage_log_details`).
		WithArgs(string(service.UsageLogDetailTypeImage), service.ImageUsageLogDetailRetentionLimitDefault).
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := usageLogBestEffortRequest{
		prepared: prepareUsageLogInsert(log),
		apiKeyID: log.APIKeyID,
		log:      log,
		resultCh: make(chan error, 1),
	}
	repo.flushBestEffortBatch(db, []usageLogBestEffortRequest{req})

	require.NoError(t, <-req.resultCh)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryFlushBestEffortBatch_DoesNotPersistDetailForDuplicateRows(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUsageLogRepositoryWithSQL(nil, db)
	createdAt := time.Now().UTC().Truncate(time.Second)
	log := &service.UsageLog{
		UserID:       1,
		APIKeyID:     2,
		AccountID:    3,
		RequestID:    "req-best-effort-duplicate",
		Model:        "claude-3",
		InputTokens:  10,
		OutputTokens: 20,
		TotalCost:    0.5,
		ActualCost:   0.5,
		CreatedAt:    createdAt,
		DetailSnapshot: (&service.UsageLogDetailSnapshot{
			RequestBody: `{"request":"payload"}`,
		}).Normalize(),
	}
	mock.ExpectQuery(`INSERT INTO usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}))
	mock.ExpectQuery(`SELECT id, created_at FROM usage_logs`).
		WithArgs(log.RequestID, log.APIKeyID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(456), createdAt))

	req := usageLogBestEffortRequest{
		prepared: prepareUsageLogInsert(log),
		apiKeyID: log.APIKeyID,
		log:      log,
		resultCh: make(chan error, 1),
	}
	repo.flushBestEffortBatch(db, []usageLogBestEffortRequest{req})

	require.NoError(t, <-req.resultCh)
	require.NoError(t, mock.ExpectationsWereMet())
}
