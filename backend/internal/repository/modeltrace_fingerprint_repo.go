package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func (r *modelTraceRepository) ReadFingerprint(ctx context.Context) (version string, data []byte, err error) {
	var content, digest string
	err = r.db.QueryRowContext(ctx, `SELECT version, data, sha256 FROM modeltrace_fingerprint WHERE id = 1`).
		Scan(&version, &content, &digest)
	if err != nil {
		return "", nil, err
	}
	data = []byte(content)
	if fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
		return "", nil, errors.New("ModelTrace fingerprint SHA-256 mismatch")
	}
	return version, data, nil
}

func (r *modelTraceRepository) TryUpdateFingerprint(ctx context.Context, fetch func(context.Context) (string, []byte, error)) (version string, data []byte, updated bool, err error) {
	var attemptDay time.Time
	// Finish the autocommit claim before fetch, so a failed download cannot release today's attempt.
	err = r.db.QueryRowContext(ctx, `INSERT INTO modeltrace_fingerprint_attempts (attempt_day)
		VALUES ((clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date)
		ON CONFLICT DO NOTHING RETURNING attempt_day`).Scan(&attemptDay)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, err
	}

	version, data, err = fetch(ctx)
	if err != nil {
		return "", nil, false, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO modeltrace_fingerprint (id, version, data, sha256, checked_at)
		VALUES (1, $1, $2, $3, clock_timestamp())
		ON CONFLICT (id) DO UPDATE SET version = EXCLUDED.version, data = EXCLUDED.data,
			sha256 = EXCLUDED.sha256, checked_at = clock_timestamp(), updated_at = clock_timestamp()`, version, string(data), fmt.Sprintf("%x", sha256.Sum256(data)))
	if err != nil {
		return "", nil, false, err
	}
	return version, data, true, nil
}
