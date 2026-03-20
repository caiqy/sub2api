package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
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

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO usage_logs (
			user_id,
			api_key_id,
			account_id,
			request_id,
			model,
			upstream_model,
			group_id,
			subscription_id,
			input_tokens,
			output_tokens,
			cache_creation_tokens,
			cache_read_tokens,
			cache_creation_5m_tokens,
			cache_creation_1h_tokens,
			input_cost,
			output_cost,
			cache_creation_cost,
			cache_read_cost,
			total_cost,
			actual_cost,
			rate_multiplier,
			account_rate_multiplier,
			billing_type,
			request_type,
			stream,
			openai_ws_mode,
			duration_ms,
			first_token_ms,
			user_agent,
			ip_address,
			image_count,
			image_size,
			media_type,
			service_tier,
			reasoning_effort,
			inbound_endpoint,
			upstream_endpoint,
			cache_ttl_overridden,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8,
			$9, $10, $11, $12,
			$13, $14,
			$15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39
		)
		ON CONFLICT (request_id, api_key_id) DO NOTHING
		RETURNING id, created_at
	`)).WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(123, createdAt))

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
			if len(args) == 6 {
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
