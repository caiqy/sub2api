CREATE TABLE IF NOT EXISTS modeltrace_instances (
    id TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS modeltrace_tasks (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source IN ('manual', 'auto')),
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    model TEXT NOT NULL,
    target_model TEXT NOT NULL,
    rounds INTEGER NOT NULL CHECK (rounds BETWEEN 1 AND 3),
    completed_rounds INTEGER NOT NULL DEFAULT 0 CHECK (completed_rounds BETWEEN 0 AND rounds),
    result TEXT NOT NULL DEFAULT '' CHECK (result IN ('', 'normal', 'degraded')),
    winner TEXT NOT NULL DEFAULT '',
    probabilities JSONB NOT NULL DEFAULT '{}',
    version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    owner TEXT NOT NULL,
    deadline TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS modeltrace_tasks_one_active ON modeltrace_tasks(account_id)
    WHERE status IN ('queued', 'running');
CREATE INDEX IF NOT EXISTS modeltrace_tasks_history ON modeltrace_tasks(account_id, id DESC);
CREATE INDEX IF NOT EXISTS modeltrace_tasks_pending ON modeltrace_tasks(status, id)
    WHERE status IN ('queued', 'running');
CREATE INDEX IF NOT EXISTS modeltrace_tasks_retention ON modeltrace_tasks(finished_at)
    WHERE finished_at IS NOT NULL;

-- This state outlives pruned history; manual tasks never move the auto clock.
CREATE TABLE IF NOT EXISTS modeltrace_account_state (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    last_auto_finished_at TIMESTAMPTZ,
    latest_task_id BIGINT NOT NULL DEFAULT 0
);

CREATE OR REPLACE FUNCTION modeltrace_account_deleted() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.deleted_at IS NOT NULL THEN
        DELETE FROM modeltrace_tasks WHERE account_id = NEW.id;
        DELETE FROM modeltrace_account_state WHERE account_id = NEW.id;
        NEW.extra := COALESCE(NEW.extra, '{}'::jsonb) - 'modeltrace_latest' - 'modeltrace_enabled';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS modeltrace_account_deleted ON accounts;
CREATE TRIGGER modeltrace_account_deleted BEFORE UPDATE OF deleted_at ON accounts
    FOR EACH ROW EXECUTE FUNCTION modeltrace_account_deleted();
