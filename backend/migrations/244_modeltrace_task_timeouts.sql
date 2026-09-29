ALTER TABLE modeltrace_tasks
    ADD COLUMN IF NOT EXISTS probe_timeout_seconds INTEGER NOT NULL DEFAULT 90
        CHECK (probe_timeout_seconds BETWEEN 10 AND 1800),
    ADD COLUMN IF NOT EXISTS task_timeout_seconds INTEGER NOT NULL DEFAULT 300
        CHECK (task_timeout_seconds BETWEEN 30 AND 7200 AND task_timeout_seconds >= probe_timeout_seconds);
