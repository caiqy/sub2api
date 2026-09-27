CREATE TABLE IF NOT EXISTS modeltrace_fingerprint (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    version TEXT NOT NULL,
    data TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    checked_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS modeltrace_fingerprint_attempts (
    attempt_day DATE PRIMARY KEY
);
