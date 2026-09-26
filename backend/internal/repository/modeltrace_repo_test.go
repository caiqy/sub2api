package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestModelTraceClaimClusterCapAndOwnerFence(t *testing.T) {
	for _, admitted := range []bool{false, true} {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		repo := &modelTraceRepository{db: db}
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT pg_advisory_xact_lock\(714032416\)`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`(?s)SELECT.*expires_at > NOW\(\).*COUNT\(\*\).*status = 'running'\) < 4`).
			WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"admitted"}).AddRow(admitted))
		if admitted {
			// The creator fence is what prevents a restart from adopting pending work.
			mock.ExpectQuery(`(?s)UPDATE modeltrace_tasks SET status = 'running'.*t.status = 'queued' AND t.owner = \$1.*SKIP LOCKED LIMIT 1`).
				WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"id"}))
		}
		mock.ExpectRollback()
		task, err := repo.Claim(context.Background(), "owner")
		require.NoError(t, err)
		require.Nil(t, task)
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	}
}

func TestModelTraceLeaseAndProgressCannotResurrect(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := &modelTraceRepository{db: db}
	mock.ExpectExec(`(?s)UPDATE modeltrace_instances SET expires_at.*WHERE id = \$1 AND expires_at > NOW\(\)`).
		WithArgs("expired").WillReturnResult(sqlmock.NewResult(0, 0))
	require.Error(t, repo.Renew(context.Background(), "expired"))
	mock.ExpectExec(`(?s)UPDATE modeltrace_tasks SET completed_rounds.*owner = \$2 AND status = 'running' AND deadline > NOW\(\).*expires_at > NOW\(\)`).
		WithArgs(int64(1), "expired", 2).WillReturnResult(sqlmock.NewResult(0, 0))
	require.Error(t, repo.Progress(context.Background(), &service.ModelTraceTask{ID: 1, CompletedRounds: 2}, "expired"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelTraceMaintenanceUsesExpiredLeasesAndKeepsIndependentState(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := &modelTraceRepository{db: db}
	mock.ExpectBegin()
	mock.ExpectExec(`(?s)WITH interrupted AS.*status = 'queued' AND NOT EXISTS.*i.expires_at > NOW\(\).*status = 'running' AND LEAST\(deadline, COALESCE.*NOW\(\) - \(\$1 \* INTERVAL '1 second'\).*INSERT INTO modeltrace_account_state.*source = 'auto'`).
		WithArgs(int64(100)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM modeltrace_tasks WHERE finished_at < NOW\(\) - INTERVAL '30 days'`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)DELETE FROM modeltrace_instances i WHERE expires_at <= NOW\(\).*NOT EXISTS`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, repo.Maintain(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelTracePreProbeFenceChecksTaskOwnerDeadlineAndDeletion(t *testing.T) {
	for _, valid := range []bool{true, false} {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		repo := &modelTraceRepository{db: db}
		mock.ExpectQuery(`(?s)SELECT EXISTS.*t.id = \$1 AND t.owner = \$2 AND t.status = 'running'.*t.deadline > NOW\(\) AND i.expires_at > NOW\(\) AND a.deleted_at IS NULL`).
			WithArgs(int64(42), "old-owner").WillReturnRows(sqlmock.NewRows([]string{"valid"}).AddRow(valid))
		err = repo.Check(context.Background(), 42, "old-owner")
		if valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	}
}
