//go:build integration || modeltrace_postgres

package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestModelTraceFingerprintPostgres(t *testing.T) {
	ctx := context.Background()
	first := &modelTraceRepository{db: integrationDB}
	second := &modelTraceRepository{db: integrationDB}
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM modeltrace_fingerprint WHERE id = 1")
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM modeltrace_fingerprint_attempts")
	})

	_, _, err := first.ReadFingerprint(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)
	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		_, _, _, err := first.TryUpdateFingerprint(ctx, func(context.Context) (string, []byte, error) {
			close(entered)
			<-release
			return "first-commit", []byte(`{"models":[1]}`), nil
		})
		finished <- err
	}()
	<-entered
	var claimedToday bool
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT attempt_day = (clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date
		FROM modeltrace_fingerprint_attempts`).Scan(&claimedToday))
	require.True(t, claimedToday)
	called := false
	contenderCtx, stopContender := context.WithTimeout(ctx, 5*time.Second)
	defer stopContender()
	_, _, updated, err := second.TryUpdateFingerprint(contenderCtx, func(context.Context) (string, []byte, error) {
		called = true
		return "other", nil, nil
	})
	close(release)
	require.NoError(t, err)
	require.False(t, updated)
	require.False(t, called)
	require.NoError(t, <-finished)
	version, data, err := second.ReadFingerprint(ctx)
	require.NoError(t, err)
	require.Equal(t, "first-commit", version)
	require.Equal(t, []byte(`{"models":[1]}`), data)

	_, _, updated, err = second.TryUpdateFingerprint(ctx, func(context.Context) (string, []byte, error) {
		called = true
		return "duplicate", nil, nil
	})
	require.NoError(t, err)
	require.False(t, updated)
	require.False(t, called)
	version, data, err = first.ReadFingerprint(ctx)
	require.NoError(t, err)
	require.Equal(t, "first-commit", version)
	require.Equal(t, []byte(`{"models":[1]}`), data)

	_, err = integrationDB.ExecContext(ctx, "DELETE FROM modeltrace_fingerprint_attempts")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `UPDATE modeltrace_fingerprint
		SET checked_at = (date_trunc('day', NOW() AT TIME ZONE 'Asia/Shanghai') AT TIME ZONE 'Asia/Shanghai') - INTERVAL '1 microsecond'
		WHERE id = 1`)
	require.NoError(t, err)
	failed := errors.New("snapshot rejected")
	_, _, updated, err = second.TryUpdateFingerprint(ctx, func(context.Context) (string, []byte, error) {
		return "bad", []byte(`{}`), failed
	})
	require.ErrorIs(t, err, failed)
	require.False(t, updated)
	version, data, err = first.ReadFingerprint(ctx)
	require.NoError(t, err)
	require.Equal(t, "first-commit", version)
	require.Equal(t, []byte(`{"models":[1]}`), data)
	var attempts int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM modeltrace_fingerprint_attempts").Scan(&attempts))
	require.Equal(t, 1, attempts)
	called = false
	_, _, updated, err = first.TryUpdateFingerprint(ctx, func(context.Context) (string, []byte, error) {
		called = true
		return "duplicate", nil, nil
	})
	require.NoError(t, err)
	require.False(t, updated)
	require.False(t, called)

	_, err = integrationDB.ExecContext(ctx, "DELETE FROM modeltrace_fingerprint_attempts")
	require.NoError(t, err)
	writeCtx, cancel := context.WithCancel(ctx)
	_, _, updated, err = second.TryUpdateFingerprint(writeCtx, func(context.Context) (string, []byte, error) {
		cancel()
		return "not-saved", []byte(`{}`), nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, updated)
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM modeltrace_fingerprint_attempts").Scan(&attempts))
	require.Equal(t, 1, attempts)
	version, data, err = first.ReadFingerprint(ctx)
	require.NoError(t, err)
	require.Equal(t, "first-commit", version)
	require.Equal(t, []byte(`{"models":[1]}`), data)

	_, err = integrationDB.ExecContext(ctx, "DELETE FROM modeltrace_fingerprint_attempts")
	require.NoError(t, err)

	version, data, updated, err = second.TryUpdateFingerprint(ctx, func(context.Context) (string, []byte, error) {
		return "next-commit", []byte(`{ "models": [2] }`), nil
	})
	require.NoError(t, err)
	require.True(t, updated)
	require.Equal(t, "next-commit", version)
	require.Equal(t, []byte(`{ "models": [2] }`), data)
	var checkedToday bool
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT (checked_at AT TIME ZONE 'Asia/Shanghai')::date =
		(clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date FROM modeltrace_fingerprint WHERE id = 1`).Scan(&checkedToday))
	require.True(t, checkedToday)
	version, data, err = first.ReadFingerprint(ctx)
	require.NoError(t, err)
	require.Equal(t, "next-commit", version)
	require.Equal(t, []byte(`{ "models": [2] }`), data)
	var rows int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM modeltrace_fingerprint").Scan(&rows))
	require.Equal(t, 1, rows)

	_, err = integrationDB.ExecContext(ctx, "UPDATE modeltrace_fingerprint SET data = '{}' WHERE id = 1")
	require.NoError(t, err)
	_, _, err = second.ReadFingerprint(ctx)
	require.ErrorContains(t, err, "SHA-256 mismatch")
}
