ALTER TABLE modeltrace_tasks
    ADD COLUMN IF NOT EXISTS accepted_models TEXT[] NOT NULL DEFAULT '{}';
