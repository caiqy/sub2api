-- Add usage_log_details snapshot table for request/response payload retention.
-- 幂等执行：允许线上已手工补过部分结构时重复运行。

CREATE TABLE IF NOT EXISTS usage_log_details (
    id BIGSERIAL PRIMARY KEY,
    usage_log_id BIGINT NOT NULL,
    request_headers TEXT NOT NULL DEFAULT '',
    request_body TEXT NOT NULL DEFAULT '',
    response_headers TEXT NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS id BIGSERIAL;
ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS usage_log_id BIGINT;
ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS request_headers TEXT;
ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS request_body TEXT;
ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS response_headers TEXT;
ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS response_body TEXT;
ALTER TABLE usage_log_details ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'usage_log_details'
          AND column_name = 'id'
          AND column_default IS NULL
    ) THEN
        EXECUTE 'CREATE SEQUENCE IF NOT EXISTS usage_log_details_id_seq';
        EXECUTE 'ALTER SEQUENCE usage_log_details_id_seq OWNED BY usage_log_details.id';
        EXECUTE 'ALTER TABLE usage_log_details ALTER COLUMN id SET DEFAULT nextval(''usage_log_details_id_seq'')';
    END IF;
END $$;

UPDATE usage_log_details
SET id = nextval('usage_log_details_id_seq')
WHERE id IS NULL;

UPDATE usage_log_details
SET request_headers = ''
WHERE request_headers IS NULL;

UPDATE usage_log_details
SET request_body = ''
WHERE request_body IS NULL;

UPDATE usage_log_details
SET response_headers = ''
WHERE response_headers IS NULL;

UPDATE usage_log_details
SET response_body = ''
WHERE response_body IS NULL;

UPDATE usage_log_details
SET created_at = NOW()
WHERE created_at IS NULL;

ALTER TABLE usage_log_details ALTER COLUMN id SET NOT NULL;
ALTER TABLE usage_log_details ALTER COLUMN usage_log_id SET NOT NULL;
ALTER TABLE usage_log_details ALTER COLUMN request_headers SET DEFAULT '';
ALTER TABLE usage_log_details ALTER COLUMN request_headers SET NOT NULL;
ALTER TABLE usage_log_details ALTER COLUMN request_body SET DEFAULT '';
ALTER TABLE usage_log_details ALTER COLUMN request_body SET NOT NULL;
ALTER TABLE usage_log_details ALTER COLUMN response_headers SET DEFAULT '';
ALTER TABLE usage_log_details ALTER COLUMN response_headers SET NOT NULL;
ALTER TABLE usage_log_details ALTER COLUMN response_body SET DEFAULT '';
ALTER TABLE usage_log_details ALTER COLUMN response_body SET NOT NULL;
ALTER TABLE usage_log_details ALTER COLUMN created_at SET DEFAULT NOW();
ALTER TABLE usage_log_details ALTER COLUMN created_at SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE n.nspname = 'public'
          AND t.relname = 'usage_log_details'
          AND c.contype = 'p'
    ) THEN
        ALTER TABLE usage_log_details
            ADD CONSTRAINT usage_log_details_pkey PRIMARY KEY (id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        JOIN unnest(c.conkey) AS cols(attnum) ON TRUE
        JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = cols.attnum
        WHERE n.nspname = 'public'
          AND t.relname = 'usage_log_details'
          AND c.contype = 'u'
        GROUP BY c.oid
        HAVING COUNT(*) = 1 AND BOOL_AND(a.attname = 'usage_log_id')
    ) THEN
        ALTER TABLE usage_log_details
            ADD CONSTRAINT usage_log_details_usage_log_id_key UNIQUE (usage_log_id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_usage_log_details_created_at
    ON usage_log_details (created_at);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = c.conkey[1]
        WHERE n.nspname = 'public'
          AND t.relname = 'usage_log_details'
          AND c.contype = 'f'
          AND array_length(c.conkey, 1) = 1
          AND a.attname = 'usage_log_id'
    ) THEN
        ALTER TABLE usage_log_details
            ADD CONSTRAINT usage_log_details_usage_logs_detail
            FOREIGN KEY (usage_log_id) REFERENCES usage_logs(id) ON DELETE CASCADE;
    END IF;
END $$;
