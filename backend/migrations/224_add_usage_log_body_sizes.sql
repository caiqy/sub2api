-- Persist request/response body sizes independently of retained usage details.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS request_body_size BIGINT,
    ADD COLUMN IF NOT EXISTS response_body_size BIGINT;

-- Existing rows remain NULL; their detail snapshots may already have been
-- removed and must not make startup scan the high-write usage_logs table.
