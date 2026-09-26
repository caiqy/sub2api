//go:build integration || modeltrace_postgres

package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestModelTracePersistenceLeasesRetentionAndDeletion(t *testing.T) {
	ctx := context.Background()
	repo := &modelTraceRepository{db: integrationDB}
	accounts := newAccountRepositoryWithSQL(testEntClient(t), integrationDB, nil)
	account := &service.Account{Name: fmt.Sprintf("modeltrace-%d", time.Now().UnixNano()), Platform: service.PlatformOpenAI,
		Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true,
		Credentials: map[string]any{"api_key": "not-saved-in-history"}, Extra: map[string]any{service.ModelTraceEnabledExtraKey: true}}
	require.NoError(t, accounts.Create(ctx, account))
	owner := fmt.Sprintf("modeltrace-owner-%d", time.Now().UnixNano())
	other := owner + "-other"
	require.NoError(t, repo.Register(ctx, owner))
	require.NoError(t, repo.Register(ctx, other))
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM modeltrace_tasks WHERE account_id = $1", account.ID)
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM modeltrace_account_state WHERE account_id = $1", account.ID)
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM scheduler_outbox WHERE account_id = $1", account.ID)
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM accounts WHERE id = $1", account.ID)
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM modeltrace_instances WHERE id IN ($1,$2)", owner, other)
	})
	makeTask := func(source string) *service.ModelTraceTask {
		return &service.ModelTraceTask{AccountID: account.ID, Source: source, Model: "gpt-6-astra", TargetModel: "gpt-6-astra", Rounds: 1, Version: modeltrace.Version}
	}

	// Multiple creators reuse the same persisted task, even across instances.
	var wg sync.WaitGroup
	tasks := make([]*service.ModelTraceTask, 12)
	errs := make([]error, len(tasks))
	for i := range tasks {
		wg.Add(1)
		go func(i int) { defer wg.Done(); tasks[i], errs[i] = repo.Enqueue(ctx, makeTask("manual"), owner, 60) }(i)
	}
	wg.Wait()
	for i := range tasks {
		require.NoError(t, errs[i])
		require.Equal(t, tasks[0].ID, tasks[i].ID)
	}
	reused, err := repo.Enqueue(ctx, makeTask("auto"), other, 60)
	require.NoError(t, err)
	require.Equal(t, tasks[0].ID, reused.ID)
	// A new process cannot adopt a queued task from the prior process.
	unclaimed, err := repo.Claim(ctx, other)
	require.NoError(t, err)
	require.Nil(t, unclaimed)
	task, err := repo.Claim(ctx, owner)
	require.NoError(t, err)
	require.NotNil(t, task)
	require.NoError(t, repo.Maintain(ctx))
	history, err := repo.History(ctx, account.ID, 1, 20)
	require.NoError(t, err)
	require.Equal(t, "running", history.Active.Status)
	task.Status, task.Result, task.Winner, task.CompletedRounds = "completed", "normal", task.TargetModel, 1
	task.Probabilities = map[string]float64{task.TargetModel: 1}
	require.NoError(t, repo.Finish(ctx, task, owner))
	var summary string
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT extra -> 'modeltrace_latest' FROM accounts WHERE id = $1", account.ID).Scan(&summary))

	// Full stale account edits and bulk/extra writes must not erase or forge the summary.
	loaded, err := accounts.GetByID(ctx, account.ID)
	require.NoError(t, err)
	loaded.Extra = map[string]any{service.ModelTraceLatestExtraKey: "forged", "ordinary": true}
	require.NoError(t, accounts.Update(ctx, loaded))
	require.NoError(t, accounts.UpdateExtra(ctx, account.ID, map[string]any{service.ModelTraceLatestExtraKey: nil}))
	_, err = accounts.BulkUpdate(ctx, []int64{account.ID}, service.AccountBulkUpdate{Extra: map[string]any{service.ModelTraceLatestExtraKey: "forged"}})
	require.NoError(t, err)
	var retained string
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT extra -> 'modeltrace_latest' FROM accounts WHERE id = $1", account.ID).Scan(&retained))
	require.Equal(t, summary, retained)

	// Pruning history leaves the independent latest summary and auto schedule state intact.
	_, err = integrationDB.ExecContext(ctx, "UPDATE modeltrace_tasks SET finished_at = NOW() - INTERVAL '31 days' WHERE id = $1", task.ID)
	require.NoError(t, err)
	require.NoError(t, repo.Maintain(ctx))
	history, err = repo.History(ctx, account.ID, 1, 20)
	require.NoError(t, err)
	require.Zero(t, history.Total)
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT extra -> 'modeltrace_latest' FROM accounts WHERE id = $1", account.ID).Scan(&retained))
	require.Equal(t, summary, retained)
	auto, err := repo.Enqueue(ctx, makeTask("auto"), owner, 60)
	require.NoError(t, err)
	require.NotNil(t, auto)
	auto, err = repo.Claim(ctx, owner)
	require.NoError(t, err)
	auto.Status, auto.Error, auto.Probabilities = "failed", "upstream request failed", map[string]float64{}
	require.NoError(t, repo.Finish(ctx, auto, owner))
	repeat, err := repo.Enqueue(ctx, makeTask("auto"), owner, 60)
	require.NoError(t, err)
	require.Nil(t, repeat)
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT extra -> 'modeltrace_latest' FROM accounts WHERE id = $1", account.ID).Scan(&retained))
	require.Equal(t, summary, retained)

	// Manual tasks do not advance the independent automatic schedule clock.
	var clock time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT last_auto_finished_at FROM modeltrace_account_state WHERE account_id = $1", account.ID).Scan(&clock))
	_, err = repo.Enqueue(ctx, makeTask("manual"), owner, 60)
	require.NoError(t, err)
	old, err := repo.Claim(ctx, owner)
	require.NoError(t, err)
	require.NotNil(t, old)
	// Even if an older worker is given a valid completion, a newer summary is authoritative.
	_, err = integrationDB.ExecContext(ctx, "UPDATE modeltrace_account_state SET latest_task_id = $2 WHERE account_id = $1", account.ID, old.ID+100)
	require.NoError(t, err)
	old.Status, old.Result, old.Winner, old.CompletedRounds, old.Probabilities = "completed", "degraded", "other", 1, map[string]float64{"other": 1}
	require.NoError(t, repo.Finish(ctx, old, owner))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT extra -> 'modeltrace_latest' FROM accounts WHERE id = $1", account.ID).Scan(&retained))
	require.Equal(t, summary, retained)
	var sameClock time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT last_auto_finished_at FROM modeltrace_account_state WHERE account_id = $1", account.ID).Scan(&sameClock))
	require.Equal(t, clock, sameClock)

	// Expired leases fail pending work; they cannot resurrect or publish an old result.
	queued, err := repo.Enqueue(ctx, makeTask("manual"), owner, 60)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, "UPDATE modeltrace_instances SET expires_at = NOW() - INTERVAL '1 second' WHERE id = $1", owner)
	require.NoError(t, err)
	require.Error(t, repo.Renew(ctx, owner))
	require.NoError(t, repo.Maintain(ctx))
	history, err = repo.History(ctx, account.ID, 1, 20)
	require.NoError(t, err)
	require.Nil(t, history.Active)
	require.Equal(t, queued.ID, history.Items[0].ID)
	require.Equal(t, "failed", history.Items[0].Status)
	queued.Status, queued.Result, queued.Winner, queued.CompletedRounds, queued.Probabilities = "completed", "normal", queued.TargetModel, 1, map[string]float64{queued.TargetModel: 1}
	require.Error(t, repo.Finish(ctx, queued, owner))
	noReplay, err := repo.Claim(ctx, other)
	require.NoError(t, err)
	require.Nil(t, noReplay)
	_, err = repo.Enqueue(ctx, makeTask("manual"), other, 60)
	require.NoError(t, err)
	expiredRun, err := repo.Claim(ctx, other)
	require.NoError(t, err)
	require.NotNil(t, expiredRun)
	require.NoError(t, repo.Check(ctx, expiredRun.ID, other))
	_, err = integrationDB.ExecContext(ctx, "UPDATE modeltrace_tasks SET deadline = NOW() - INTERVAL '1 second' WHERE id = $1", expiredRun.ID)
	require.NoError(t, err)
	require.NoError(t, repo.Maintain(ctx))
	history, err = repo.History(ctx, account.ID, 1, 20)
	require.NoError(t, err)
	require.NotNil(t, history.Active, "expired running task retains the account slot for the in-flight request")
	require.Error(t, repo.Check(ctx, expiredRun.ID, other))
	_, err = integrationDB.ExecContext(ctx, "UPDATE modeltrace_tasks SET deadline = NOW() - INTERVAL '101 seconds' WHERE id = $1", expiredRun.ID)
	require.NoError(t, err)
	require.NoError(t, repo.Maintain(ctx))
	history, err = repo.History(ctx, account.ID, 1, 20)
	require.NoError(t, err)
	require.Nil(t, history.Active)
	require.Equal(t, "failed", history.Items[0].Status)
	expiredRun.Status, expiredRun.Result, expiredRun.Winner, expiredRun.CompletedRounds, expiredRun.Probabilities = "completed", "normal", expiredRun.TargetModel, 1, map[string]float64{expiredRun.TargetModel: 1}
	require.Error(t, repo.Finish(ctx, expiredRun, other))

	// Lease expiration also retains the slot: another epoch must reuse, not overlap, the old run.
	freshOwner := other + "-recovered"
	require.NoError(t, repo.Register(ctx, freshOwner))
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM modeltrace_instances WHERE id = $1", freshOwner)
	})
	_, err = repo.Enqueue(ctx, makeTask("manual"), other, 60)
	require.NoError(t, err)
	lostRun, err := repo.Claim(ctx, other)
	require.NoError(t, err)
	require.NotNil(t, lostRun)
	_, err = integrationDB.ExecContext(ctx, "UPDATE modeltrace_instances SET expires_at = NOW() - INTERVAL '1 second' WHERE id = $1", other)
	require.NoError(t, err)
	require.Error(t, repo.Check(ctx, lostRun.ID, other))
	require.NoError(t, repo.Maintain(ctx))
	retainedRun, err := repo.Enqueue(ctx, makeTask("manual"), freshOwner, 60)
	require.NoError(t, err)
	require.Equal(t, lostRun.ID, retainedRun.ID)
	_, err = integrationDB.ExecContext(ctx, "UPDATE modeltrace_instances SET expires_at = NOW() - INTERVAL '101 seconds' WHERE id = $1", other)
	require.NoError(t, err)
	require.NoError(t, repo.Maintain(ctx))
	fresh, err := repo.Enqueue(ctx, makeTask("manual"), freshOwner, 60)
	require.NoError(t, err)
	require.Greater(t, fresh.ID, lostRun.ID)
	fresh, err = repo.Claim(ctx, freshOwner)
	require.NoError(t, err)
	require.NotNil(t, fresh)
	require.NoError(t, repo.Check(ctx, fresh.ID, freshOwner))

	// Soft deletion trigger clears history, state and managed extra in the same account transaction.
	_, err = integrationDB.ExecContext(ctx, "UPDATE accounts SET deleted_at = NOW() WHERE id = $1", account.ID)
	require.NoError(t, err)
	_, err = repo.History(ctx, account.ID, 1, 20)
	require.ErrorIs(t, err, service.ErrAccountNotFound)
	var remaining int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM modeltrace_tasks WHERE account_id = $1", account.ID).Scan(&remaining))
	require.Zero(t, remaining)
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM modeltrace_account_state WHERE account_id = $1", account.ID).Scan(&remaining))
	require.Zero(t, remaining)
}
