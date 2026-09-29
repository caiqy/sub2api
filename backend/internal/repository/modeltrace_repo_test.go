package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestModelTraceManagedExtraCannotBeSubmitted(t *testing.T) {
	incoming := map[string]any{"modeltrace_quarantined": true, service.ModelTraceLatestExtraKey: "forged", "ordinary": true}
	clean := stripModelTraceLatestExtra(incoming)
	require.Equal(t, map[string]any{"ordinary": true}, clean)
	require.Contains(t, incoming, "modeltrace_quarantined")
}

func TestModelTraceGroupCapacityPredicates(t *testing.T) {
	require.Contains(t, groupAccountAvailableSQL, "modeltrace_quarantined")
	require.Contains(t, groupAccountTemporarilyLimitedSQL, "modeltrace_quarantined")
}

func TestModelTraceCandidatesUseSafeAccountInterval(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(_, actual string) error {
		for _, part := range []string{
			"jsonb_typeof(a.extra -> 'modeltrace_interval_minutes') = 'number'",
			"'^[0-9]{1,5}$'", "::int BETWEEN 5 AND 10080", "ELSE $1 END",
			"last_auto_finished_at <= NOW()", "modeltrace_enabled",
		} {
			if !strings.Contains(actual, part) {
				return sql.ErrNoRows
			}
		}
		if strings.Contains(actual, "modeltrace_quarantined") {
			return sql.ErrNoRows
		}
		return nil
	})))
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectQuery("candidates").WithArgs(60, int64(10)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(11)))
	ids, err := (&modelTraceRepository{db: db}).Candidates(context.Background(), 60, 10)
	require.NoError(t, err)
	require.Equal(t, []int64{11}, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelTraceEnqueueRechecksLockedAccountInterval(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT a.platform.*modeltrace_interval_minutes.*FOR NO KEY UPDATE`).
		WithArgs(60, int64(42)).WillReturnRows(sqlmock.NewRows([]string{"platform", "enabled", "interval"}).AddRow(service.PlatformOpenAI, true, 120))
	mock.ExpectQuery(`SELECT .* FROM modeltrace_tasks`).WithArgs(int64(42)).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT NOT EXISTS`).WithArgs(int64(42), 120).WillReturnRows(sqlmock.NewRows([]string{"due"}).AddRow(false))
	mock.ExpectRollback()
	task, err := (&modelTraceRepository{db: db}).Enqueue(context.Background(), &service.ModelTraceTask{AccountID: 42, Source: "auto"}, "owner", 60)
	require.NoError(t, err)
	require.Nil(t, task)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelTraceRetryEnqueueKeepsAccountGuardAndUsesFailedTask(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint("enabled-", enabled), func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			repo := &modelTraceRepository{db: db}
			mock.ExpectBegin()
			mock.ExpectQuery(`(?s)SELECT a.platform.*FOR NO KEY UPDATE`).
				WithArgs(60, int64(42)).WillReturnRows(sqlmock.NewRows([]string{"platform", "enabled", "interval"}).AddRow(service.PlatformOpenAI, enabled, 60))
			mock.ExpectQuery(`SELECT .* FROM modeltrace_tasks`).
				WithArgs(int64(42)).WillReturnError(sql.ErrNoRows)
			if !enabled {
				mock.ExpectRollback()
				task, err := repo.Enqueue(context.Background(), &service.ModelTraceTask{AccountID: 42, Source: "auto", Retry: true, RetryAfterID: 7}, "owner", 60)
				require.NoError(t, err)
				require.Nil(t, task)
			} else {
				mock.ExpectQuery(`(?s)SELECT EXISTS.*status = 'failed'.*finished_at <= NOW\(\) - INTERVAL '5 minutes'.*newer.source = 'auto'`).
					WithArgs(int64(7), int64(42)).WillReturnRows(sqlmock.NewRows([]string{"due"}).AddRow(true))
				mock.ExpectQuery(`INSERT INTO modeltrace_tasks`).
					WithArgs(int64(42), "auto", "gpt-6-astra", "gpt-6-astra", false, 1, "v", "owner", 90, 300).
					WillReturnError(sql.ErrConnDone)
				mock.ExpectRollback()
				_, err := repo.Enqueue(context.Background(), &service.ModelTraceTask{AccountID: 42, Source: "auto", Model: "gpt-6-astra", TargetModel: "gpt-6-astra", Retry: true, RetryAfterID: 7, Rounds: 1, Version: "v"}, "owner", 60)
				require.ErrorIs(t, err, sql.ErrConnDone)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestModelTraceFinishOnlyLatestEnabledResultChangesQuarantine(t *testing.T) {
	for _, tt := range []struct {
		name, status, result                string
		enabled, quarantined, latest, event bool
	}{
		{"degraded", "completed", "degraded", true, false, true, true},
		{"recovered", "completed", "normal", true, true, true, true},
		{"unchanged", "completed", "degraded", true, true, true, false},
		{"disabled", "completed", "degraded", false, false, true, false},
		{"disabled retained", "completed", "normal", false, true, true, false},
		{"old", "completed", "degraded", true, false, false, false},
		{"failed", "failed", "", true, true, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT COALESCE\(extra -> 'modeltrace_quarantine_enabled'`).WithArgs(int64(42)).
				WillReturnRows(sqlmock.NewRows([]string{"enabled", "quarantined"}).AddRow(tt.enabled, tt.quarantined))
			winner := "target"
			if tt.status == "failed" {
				winner = ""
			}
			mock.ExpectQuery(`UPDATE modeltrace_tasks SET status`).WithArgs(int64(7), "owner", tt.status, 1, tt.result, winner, "{}", "").
				WillReturnRows(sqlmock.NewRows([]string{"finished_at", "duration_ms"}).AddRow(time.Now(), int64(20)))
			mock.ExpectExec(`INSERT INTO modeltrace_account_state`).WithArgs(int64(42), "manual").WillReturnResult(sqlmock.NewResult(0, 1))
			if tt.status == "completed" {
				affected := int64(0)
				if tt.latest {
					affected = 1
				}
				mock.ExpectExec(`UPDATE modeltrace_account_state SET latest_task_id`).WithArgs(int64(42), int64(7)).WillReturnResult(sqlmock.NewResult(0, affected))
				if tt.latest {
					mock.ExpectExec(`(?s)UPDATE accounts SET extra =.*modeltrace_quarantined.*modeltrace_latest`).
						WithArgs(int64(42), sqlmock.AnyArg(), tt.enabled, tt.result == "degraded").WillReturnResult(sqlmock.NewResult(0, 1))
					if tt.event {
						mock.ExpectExec(`INSERT INTO scheduler_outbox`).WillReturnResult(sqlmock.NewResult(0, 1))
					}
				}
			}
			mock.ExpectCommit()
			task := &service.ModelTraceTask{ID: 7, AccountID: 42, Source: "manual", Status: tt.status,
				Rounds: 1, CompletedRounds: 1, Result: tt.result, Winner: "target", Probabilities: map[string]float64{}}
			require.NoError(t, (&modelTraceRepository{db: db}).Finish(context.Background(), task, "owner"))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestModelTraceFinishRollsBackWhenSchedulerOutboxFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COALESCE\(extra -> 'modeltrace_quarantine_enabled'`).WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "quarantined"}).AddRow(true, false))
	mock.ExpectQuery(`UPDATE modeltrace_tasks SET status`).
		WillReturnRows(sqlmock.NewRows([]string{"finished_at", "duration_ms"}).AddRow(time.Now(), int64(20)))
	mock.ExpectExec(`INSERT INTO modeltrace_account_state`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE modeltrace_account_state SET latest_task_id`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE accounts SET extra`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO scheduler_outbox`).WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()
	task := &service.ModelTraceTask{ID: 7, AccountID: 42, Source: "manual", Status: "completed",
		Rounds: 1, CompletedRounds: 1, Result: "degraded", Winner: "target", Probabilities: map[string]float64{}}
	require.ErrorIs(t, (&modelTraceRepository{db: db}).Finish(context.Background(), task, "owner"), sql.ErrConnDone)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelTraceFinishRejectsUnknownResultBeforeMutation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COALESCE\(extra -> 'modeltrace_quarantine_enabled'`).WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "quarantined"}).AddRow(true, true))
	mock.ExpectRollback()
	task := &service.ModelTraceTask{AccountID: 42, Status: "completed", Rounds: 1, CompletedRounds: 1,
		Result: "unknown", Winner: "target"}
	require.ErrorContains(t, (&modelTraceRepository{db: db}).Finish(context.Background(), task, "owner"), "incomplete ModelTrace result")
	require.NoError(t, mock.ExpectationsWereMet())
}

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
			mock.ExpectQuery(`(?s)UPDATE modeltrace_tasks SET status = 'running'.*deadline = NOW\(\) \+ \(task_timeout_seconds \* INTERVAL '1 second'\).*t.status = 'queued' AND t.owner = \$1.*SKIP LOCKED LIMIT 1`).
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
		WithArgs(int64(1), "expired", 2, "fingerprint-version").WillReturnResult(sqlmock.NewResult(0, 0))
	require.Error(t, repo.Progress(context.Background(), &service.ModelTraceTask{ID: 1, CompletedRounds: 2, Version: "fingerprint-version"}, "expired"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelTraceMaintenanceUsesExpiredLeasesAndKeepsIndependentState(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := &modelTraceRepository{db: db}
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)WITH interrupted AS.*status = 'queued' AND NOT EXISTS.*i.expires_at > NOW\(\).*status = 'running' AND LEAST\(deadline, COALESCE.*NOW\(\) - \(\(t.probe_timeout_seconds \+ 10\) \* INTERVAL '1 second'\).*INSERT INTO modeltrace_account_state.*source = 'auto'.*RETURNING account_id.*SELECT interrupted.id, interrupted.account_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "account_id", "finished_at"}).AddRow(7, 1, time.Now()))
	mock.ExpectExec(`DELETE FROM modeltrace_tasks WHERE finished_at < NOW\(\) - INTERVAL '30 days'`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)DELETE FROM modeltrace_instances i WHERE expires_at <= NOW\(\).*NOT EXISTS`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	interrupted, err := repo.Maintain(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(7), interrupted[0].ID)
	require.Equal(t, int64(1), interrupted[0].AccountID)
	require.False(t, interrupted[0].FinishedAt.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelTraceFinishedReadsOnlyTerminalAutoTasks(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	finishedAt := time.Now()
	mock.ExpectQuery(`(?s)SELECT id, account_id, status, finished_at FROM modeltrace_tasks.*id = ANY\(\$1\) AND source = 'auto' AND status IN`).
		WithArgs(pq.Array([]int64{7})).WillReturnRows(sqlmock.NewRows([]string{"id", "account_id", "status", "finished_at"}).AddRow(int64(7), int64(1), "failed", finishedAt))
	tasks, err := (&modelTraceRepository{db: db}).Finished(context.Background(), []int64{7})
	require.NoError(t, err)
	require.Equal(t, []service.ModelTraceFinishedTask{{ID: 7, AccountID: 1, Status: "failed", FinishedAt: finishedAt}}, tasks)
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
