package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestModelTraceReadFingerprint(t *testing.T) {
	for _, tc := range []struct {
		name   string
		digest string
		want   error
	}{
		{name: "valid", digest: fmt.Sprintf("%x", sha256.Sum256([]byte(`{"models":[]}`)))},
		{name: "corrupt", digest: "wrong", want: errors.New("SHA-256 mismatch")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			mock.ExpectQuery(`SELECT version, data, sha256 FROM modeltrace_fingerprint`).
				WillReturnRows(sqlmock.NewRows([]string{"version", "data", "sha256"}).AddRow("commit", `{"models":[]}`, tc.digest))
			version, data, err := (&modelTraceRepository{db: db}).ReadFingerprint(context.Background())
			if tc.want != nil {
				require.ErrorContains(t, err, tc.want.Error())
				require.Empty(t, version)
				require.Nil(t, data)
			} else {
				require.NoError(t, err)
				require.Equal(t, "commit", version)
				require.Equal(t, []byte(`{"models":[]}`), data)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectQuery(`SELECT version, data, sha256 FROM modeltrace_fingerprint`).
		WillReturnError(sql.ErrNoRows)
	_, _, err = (&modelTraceRepository{db: db}).ReadFingerprint(context.Background())
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelTraceTryUpdateFingerprint(t *testing.T) {
	for _, tc := range []struct {
		name     string
		claimErr error
		fetchErr error
		writeErr error
		updated  bool
	}{
		{name: "already attempted", claimErr: sql.ErrNoRows},
		{name: "claim failed", claimErr: errors.New("database claim failed")},
		{name: "fetch failed", fetchErr: errors.New("invalid snapshot")},
		{name: "write failed", writeErr: errors.New("database write failed")},
		{name: "committed", updated: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			claim := mock.ExpectQuery(`(?s)INSERT INTO modeltrace_fingerprint_attempts.*clock_timestamp\(\) AT TIME ZONE 'Asia/Shanghai'.*ON CONFLICT DO NOTHING RETURNING attempt_day`)
			if tc.claimErr != nil {
				claim.WillReturnError(tc.claimErr)
			} else {
				claim.WillReturnRows(sqlmock.NewRows([]string{"attempt_day"}).AddRow(time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)))
			}
			if tc.claimErr == nil && tc.fetchErr == nil {
				mock.ExpectExec(`INSERT INTO modeltrace_fingerprint`).
					WithArgs("commit", `{"models":[]}`, fmt.Sprintf("%x", sha256.Sum256([]byte(`{"models":[]}`)))).
					WillReturnResult(sqlmock.NewResult(0, 1)).WillReturnError(tc.writeErr)
			}
			called := false
			version, data, updated, err := (&modelTraceRepository{db: db}).TryUpdateFingerprint(context.Background(), func(context.Context) (string, []byte, error) {
				called = true
				return "commit", []byte(`{"models":[]}`), tc.fetchErr
			})
			require.Equal(t, tc.claimErr == nil, called)
			require.Equal(t, tc.updated, updated)
			if tc.claimErr != nil && !errors.Is(tc.claimErr, sql.ErrNoRows) || tc.fetchErr != nil || tc.writeErr != nil {
				want := tc.claimErr
				if tc.fetchErr != nil {
					want = tc.fetchErr
				}
				if tc.writeErr != nil {
					want = tc.writeErr
				}
				require.ErrorIs(t, err, want)
			} else {
				require.NoError(t, err)
			}
			if tc.updated {
				require.Equal(t, "commit", version)
				require.Equal(t, []byte(`{"models":[]}`), data)
			} else {
				require.Empty(t, version)
				require.Nil(t, data)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
