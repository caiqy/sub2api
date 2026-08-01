CREATE TABLE IF NOT EXISTS subscription_quota_advance_receipts (
    id BIGSERIAL PRIMARY KEY,
    scope VARCHAR(128) NOT NULL,
    idempotency_key_hash CHAR(64) NOT NULL,
    request_fingerprint CHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    subscription_id BIGINT NOT NULL,
    daily BOOLEAN NOT NULL DEFAULT FALSE,
    weekly BOOLEAN NOT NULL DEFAULT FALSE,
    monthly BOOLEAN NOT NULL DEFAULT FALSE,
    response_status SMALLINT NOT NULL DEFAULT 200,
    response_body TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (scope, idempotency_key_hash)
);

CREATE INDEX IF NOT EXISTS idx_subscription_quota_advance_receipts_expires_at
    ON subscription_quota_advance_receipts (expires_at);
