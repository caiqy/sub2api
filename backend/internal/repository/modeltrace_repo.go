package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type modelTraceRepository struct{ db *sql.DB }

func stripModelTraceLatestExtra(extra map[string]any) map[string]any {
	_, latest := extra[service.ModelTraceLatestExtraKey]
	_, quarantined := extra["modeltrace_quarantined"]
	if !latest && !quarantined {
		return extra
	}
	extra = copyJSONMap(extra)
	delete(extra, service.ModelTraceLatestExtraKey)
	delete(extra, "modeltrace_quarantined")
	return extra
}

func modelTraceExtraMergeSQL(expression string, updates map[string]any) string {
	if enabled, ok := updates[service.ModelTraceQuarantineEnabledExtraKey].(bool); ok && !enabled {
		expression = "(" + expression + ") - 'modeltrace_quarantined'"
	}
	if value, exists := updates[service.ModelTraceIntervalExtraKey]; exists && value == nil {
		expression = "(" + expression + ") - 'modeltrace_interval_minutes'"
	}
	if value, exists := updates[service.ModelTraceModelExtraKey]; exists && value == nil {
		expression = "(" + expression + ") - 'modeltrace_model'"
	}
	return expression
}

func NewModelTraceRepository(db *sql.DB) service.ModelTraceRepository {
	return &modelTraceRepository{db: db}
}

const modelTraceColumns = `id, account_id, source, status, model, target_model, model_override, rounds,
    completed_rounds, result, winner, probabilities, version, created_at, started_at,
    finished_at, duration_ms, error`

// CASE guards the cast: malformed or out-of-range persisted JSON must use the global interval.
const modelTraceIntervalSQL = `CASE WHEN jsonb_typeof(a.extra -> 'modeltrace_interval_minutes') = 'number'
    AND a.extra ->> 'modeltrace_interval_minutes' ~ '^[0-9]{1,5}$'
    THEN CASE WHEN (a.extra ->> 'modeltrace_interval_minutes')::int BETWEEN 5 AND 10080
        THEN (a.extra ->> 'modeltrace_interval_minutes')::int ELSE $1 END
    ELSE $1 END`

func scanModelTrace(row interface{ Scan(...any) error }) (*service.ModelTraceTask, error) {
	var t service.ModelTraceTask
	var probabilities []byte
	err := row.Scan(&t.ID, &t.AccountID, &t.Source, &t.Status, &t.Model, &t.TargetModel,
		&t.ModelOverride, &t.Rounds, &t.CompletedRounds, &t.Result, &t.Winner, &probabilities, &t.Version,
		&t.CreatedAt, &t.StartedAt, &t.FinishedAt, &t.DurationMS, &t.Error)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(probabilities, &t.Probabilities); err != nil {
		return nil, err
	}
	if t.Probabilities == nil {
		t.Probabilities = map[string]float64{}
	}
	return &t, nil
}

func (r *modelTraceRepository) Register(ctx context.Context, owner string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO modeltrace_instances(id, expires_at) VALUES ($1, NOW() + INTERVAL '45 seconds')`, owner)
	return err
}

func (r *modelTraceRepository) Renew(ctx context.Context, owner string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE modeltrace_instances SET expires_at = NOW() + INTERVAL '45 seconds'
        WHERE id = $1 AND expires_at > NOW()`, owner)
	return modelTraceAffected(result, err)
}

func (r *modelTraceRepository) Release(ctx context.Context, owner string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE modeltrace_instances SET expires_at = NOW() WHERE id = $1`, owner)
	if err != nil {
		return err
	}
	_, err = r.Maintain(ctx)
	return err
}

func modelTraceAffected(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("ModelTrace task or instance lease lost")
	}
	return nil
}

func (r *modelTraceRepository) Enqueue(ctx context.Context, task *service.ModelTraceTask, owner string, interval int) (*service.ModelTraceTask, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	// Serialize creation with deletion, summary writes, and other creators of this account.
	var platform string
	var enabled bool
	var effectiveInterval int
	if err := tx.QueryRowContext(ctx, `SELECT a.platform, COALESCE(a.extra -> 'modeltrace_enabled' = 'true'::jsonb, false), `+modelTraceIntervalSQL+`
        FROM accounts a WHERE a.id = $2 AND a.deleted_at IS NULL FOR NO KEY UPDATE`, interval, task.AccountID).Scan(&platform, &enabled, &effectiveInterval); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrAccountNotFound
		}
		return nil, err
	}
	active, err := scanModelTrace(tx.QueryRowContext(ctx, `SELECT `+modelTraceColumns+` FROM modeltrace_tasks
        WHERE account_id = $1 AND status IN ('queued', 'running')`, task.AccountID))
	if err == nil {
		active.Created = false
		return active, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if platform != service.PlatformOpenAI {
		return nil, errors.New("ModelTrace account platform changed")
	}
	if task.Source == "auto" {
		if !enabled {
			return nil, nil
		}
		var due bool
		if task.Retry {
			err = tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM modeltrace_tasks
			    WHERE id = $1 AND account_id = $2 AND source = 'auto' AND status = 'failed'
			      AND finished_at <= NOW() - INTERVAL '5 minutes'
			      AND NOT EXISTS (SELECT 1 FROM modeltrace_tasks newer
			        WHERE newer.account_id = $2 AND newer.source = 'auto' AND newer.id > $1))`, task.RetryAfterID, task.AccountID).Scan(&due)
		} else {
			err = tx.QueryRowContext(ctx, `SELECT NOT EXISTS (SELECT 1 FROM modeltrace_account_state
			    WHERE account_id = $1 AND last_auto_finished_at > NOW() - ($2 * INTERVAL '1 minute'))`, task.AccountID, effectiveInterval).Scan(&due)
		}
		if err != nil {
			return nil, err
		}
		if !due {
			return nil, nil
		}
	}
	task, err = scanModelTrace(tx.QueryRowContext(ctx, `INSERT INTO modeltrace_tasks
        (account_id, source, model, target_model, model_override, rounds, version, owner)
        SELECT $1, $2, $3, $4, $5, $6, $7, $8 WHERE EXISTS
        (SELECT 1 FROM modeltrace_instances WHERE id = $8 AND expires_at > NOW())
        RETURNING `+modelTraceColumns, task.AccountID, task.Source, task.Model, task.TargetModel, task.ModelOverride, task.Rounds, task.Version, owner))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	task.Created = true
	return task, nil
}

func (r *modelTraceRepository) Claim(ctx context.Context, owner string) (*service.ModelTraceTask, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	// The short global claim lock makes the concurrency cap cluster-wide, not per process.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(714032416)`); err != nil {
		return nil, err
	}
	var admitted bool
	if err := tx.QueryRowContext(ctx, `SELECT
        EXISTS (SELECT 1 FROM modeltrace_instances WHERE id = $1 AND expires_at > NOW())
        AND (SELECT COUNT(*) FROM modeltrace_tasks WHERE status = 'running') < 4`, owner).Scan(&admitted); err != nil {
		return nil, err
	}
	if !admitted {
		return nil, nil
	}
	task, err := scanModelTrace(tx.QueryRowContext(ctx, `UPDATE modeltrace_tasks SET status = 'running',
        started_at = NOW(), deadline = NOW() + INTERVAL '5 minutes', owner = $1
        WHERE id = (SELECT t.id FROM modeltrace_tasks t
            JOIN modeltrace_instances i ON i.id = t.owner AND i.expires_at > NOW()
            JOIN accounts a ON a.id = t.account_id AND a.deleted_at IS NULL
            WHERE t.status = 'queued' AND t.owner = $1 ORDER BY t.id FOR UPDATE OF t SKIP LOCKED LIMIT 1)
        RETURNING `+modelTraceColumns, owner))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return task, nil
}

func (r *modelTraceRepository) Progress(ctx context.Context, task *service.ModelTraceTask, owner string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE modeltrace_tasks SET completed_rounds = $3, version = $4
		WHERE id = $1 AND owner = $2 AND status = 'running' AND deadline > NOW()
		AND EXISTS (SELECT 1 FROM modeltrace_instances WHERE id = $2 AND expires_at > NOW())`, task.ID, owner, task.CompletedRounds, task.Version)
	return modelTraceAffected(result, err)
}

func (r *modelTraceRepository) Check(ctx context.Context, taskID int64, owner string) error {
	var valid bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM modeltrace_tasks t
		JOIN modeltrace_instances i ON i.id = t.owner
		JOIN accounts a ON a.id = t.account_id
		WHERE t.id = $1 AND t.owner = $2 AND t.status = 'running'
		AND t.deadline > NOW() AND i.expires_at > NOW() AND a.deleted_at IS NULL
	)`, taskID, owner).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("ModelTrace task or instance lease lost")
	}
	return nil
}

func (r *modelTraceRepository) Finish(ctx context.Context, task *service.ModelTraceTask, owner string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var quarantineEnabled, quarantined bool
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(extra -> 'modeltrace_quarantine_enabled' = 'true'::jsonb, false),
        COALESCE(extra -> 'modeltrace_quarantined' = 'true'::jsonb, false)
        FROM accounts WHERE id = $1 AND deleted_at IS NULL FOR NO KEY UPDATE`, task.AccountID).Scan(&quarantineEnabled, &quarantined); err != nil {
		return err
	}
	probabilities, err := json.Marshal(task.Probabilities)
	if err != nil {
		return err
	}
	if task.Status != "completed" && task.Status != "failed" {
		return errors.New("invalid ModelTrace final status")
	}
	if task.Status == "completed" && (task.CompletedRounds != task.Rounds ||
		(task.Result != "normal" && task.Result != "degraded") || task.Winner == "") {
		return errors.New("incomplete ModelTrace result")
	}
	if task.Status == "failed" {
		task.Result, task.Winner = "", ""
		probabilities = []byte("{}")
	}
	err = tx.QueryRowContext(ctx, `UPDATE modeltrace_tasks SET status = $3, completed_rounds = $4,
        result = $5, winner = $6, probabilities = $7::jsonb, error = $8, finished_at = NOW(),
        duration_ms = GREATEST(0, (EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000)::bigint)
        WHERE id = $1 AND owner = $2 AND status = 'running' AND deadline > NOW()
        AND EXISTS (SELECT 1 FROM modeltrace_instances WHERE id = $2 AND expires_at > NOW())
        RETURNING finished_at, duration_ms`, task.ID, owner, task.Status, task.CompletedRounds, task.Result, task.Winner, string(probabilities), task.Error).Scan(&task.FinishedAt, &task.DurationMS)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO modeltrace_account_state(account_id, last_auto_finished_at)
        VALUES ($1, CASE WHEN $2 = 'auto' THEN NOW() END)
        ON CONFLICT (account_id) DO UPDATE SET last_auto_finished_at =
        CASE WHEN $2 = 'auto' THEN NOW() ELSE modeltrace_account_state.last_auto_finished_at END`, task.AccountID, task.Source); err != nil {
		return err
	}
	if task.Status == "completed" {
		result, err := tx.ExecContext(ctx, `UPDATE modeltrace_account_state SET latest_task_id = $2
            WHERE account_id = $1 AND latest_task_id < $2`, task.AccountID, task.ID)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n == 1 {
			latest, err := json.Marshal(map[string]any{
				"task_id": task.ID, "result": task.Result, "model": task.Model, "target_model": task.TargetModel,
				"winner": task.Winner, "probabilities": task.Probabilities, "finished_at": task.FinishedAt, "version": task.Version,
			})
			if err != nil {
				return err
			}
			shouldQuarantine := task.Result == "degraded"
			if _, err := tx.ExecContext(ctx, `UPDATE accounts SET extra =
                CASE WHEN $3 THEN (COALESCE(extra, '{}'::jsonb) - 'modeltrace_quarantined') ||
                    CASE WHEN $4 THEN '{"modeltrace_quarantined":true}'::jsonb ELSE '{}'::jsonb END
                ELSE COALESCE(extra, '{}'::jsonb) END ||
                jsonb_build_object('modeltrace_latest', $2::jsonb),
                updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, task.AccountID, string(latest), quarantineEnabled, shouldQuarantine); err != nil {
				return err
			}
			if quarantineEnabled && quarantined != shouldQuarantine {
				if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &task.AccountID, nil, nil); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit()
}

func (r *modelTraceRepository) Maintain(ctx context.Context) ([]service.ModelTraceInterruptedTask, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	// Every send is fenced inside a pre-existing 90s context. Keep the unique active slot
	// for 100s after the earlier lease/deadline cutoff, even if that owner was canceled.
	rows, err := tx.QueryContext(ctx, `WITH interrupted AS (
        UPDATE modeltrace_tasks t SET status = 'failed', error = 'detection interrupted or worker lease expired',
            finished_at = NOW(), duration_ms = CASE WHEN started_at IS NULL THEN 0 ELSE
            GREATEST(0, (EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000)::bigint) END
        WHERE (status = 'queued' AND NOT EXISTS (
            SELECT 1 FROM modeltrace_instances i WHERE i.id = t.owner AND i.expires_at > NOW()))
            OR (status = 'running' AND LEAST(deadline, COALESCE(
                (SELECT i.expires_at FROM modeltrace_instances i WHERE i.id = t.owner), deadline))
                <= NOW() - ($1 * INTERVAL '1 second'))
		RETURNING id, account_id, source, finished_at
		), updated_state AS (
		INSERT INTO modeltrace_account_state(account_id, last_auto_finished_at)
		SELECT account_id, MAX(finished_at) FROM interrupted WHERE source = 'auto' GROUP BY account_id
		ON CONFLICT (account_id) DO UPDATE SET last_auto_finished_at =
		GREATEST(modeltrace_account_state.last_auto_finished_at, EXCLUDED.last_auto_finished_at)
		RETURNING account_id
		)
		SELECT interrupted.id, interrupted.account_id, interrupted.finished_at
		FROM interrupted JOIN updated_state USING (account_id)
		WHERE interrupted.source = 'auto'`,
		int64((service.ModelTraceProbeTimeout+10*time.Second)/time.Second))
	if err != nil {
		return nil, err
	}
	interrupted := make([]service.ModelTraceInterruptedTask, 0)
	for rows.Next() {
		var task service.ModelTraceInterruptedTask
		if err := rows.Scan(&task.ID, &task.AccountID, &task.FinishedAt); err != nil {
			rows.Close()
			return nil, err
		}
		interrupted = append(interrupted, task)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if _, err := tx.ExecContext(ctx, `DELETE FROM modeltrace_tasks WHERE finished_at < NOW() - INTERVAL '30 days'`); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM modeltrace_instances i WHERE expires_at <= NOW()
        AND NOT EXISTS (SELECT 1 FROM modeltrace_tasks t WHERE t.owner = i.id AND t.status IN ('queued','running'))`); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return interrupted, nil
}

func (r *modelTraceRepository) Finished(ctx context.Context, ids []int64) ([]service.ModelTraceFinishedTask, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, account_id, status, finished_at FROM modeltrace_tasks
		WHERE id = ANY($1) AND source = 'auto' AND status IN ('completed', 'failed')`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := make([]service.ModelTraceFinishedTask, 0)
	for rows.Next() {
		var task service.ModelTraceFinishedTask
		if err := rows.Scan(&task.ID, &task.AccountID, &task.Status, &task.FinishedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (r *modelTraceRepository) Candidates(ctx context.Context, interval int, after int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT a.id FROM accounts a
        LEFT JOIN modeltrace_account_state s ON s.account_id = a.id
        WHERE a.id > $2 AND a.deleted_at IS NULL AND a.platform = 'openai' AND a.status = 'active'
        AND a.schedulable AND a.extra -> 'modeltrace_enabled' = 'true'::jsonb
        AND (s.last_auto_finished_at IS NULL OR s.last_auto_finished_at <= NOW() - ((`+modelTraceIntervalSQL+`) * INTERVAL '1 minute'))
        AND NOT EXISTS (SELECT 1 FROM modeltrace_tasks t WHERE t.account_id = a.id AND t.status IN ('queued','running'))
        ORDER BY a.id LIMIT 100`, interval, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0, 100)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *modelTraceRepository) History(ctx context.Context, accountID int64, page, pageSize int) (*service.ModelTraceHistory, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND deleted_at IS NULL)`, accountID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, service.ErrAccountNotFound
	}
	h := &service.ModelTraceHistory{Items: []*service.ModelTraceTask{}, Page: page, PageSize: pageSize}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM modeltrace_tasks WHERE account_id = $1
        AND (finished_at IS NULL OR finished_at >= NOW() - INTERVAL '30 days')`, accountID).Scan(&h.Total); err != nil {
		return nil, err
	}
	h.Pages = max(1, int((h.Total+int64(pageSize)-1)/int64(pageSize)))
	rows, err := tx.QueryContext(ctx, `SELECT `+modelTraceColumns+` FROM modeltrace_tasks WHERE account_id = $1
        AND (finished_at IS NULL OR finished_at >= NOW() - INTERVAL '30 days')
        ORDER BY id DESC LIMIT $2 OFFSET $3`, accountID, pageSize, int64(page-1)*int64(pageSize))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		t, err := scanModelTrace(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		h.Items = append(h.Items, t)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	h.Active, err = scanModelTrace(tx.QueryRowContext(ctx, `SELECT `+modelTraceColumns+` FROM modeltrace_tasks
        WHERE account_id = $1 AND status IN ('queued','running')`, accountID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return h, tx.Commit()
}
