//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"math/big"
	"strings"
	"testing"
	"testing/fstest"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

// filtered FS simulates upgrading the main@16c07d806 (0.1.169.3) local migration set; Task 9 Git-object identity evidence separately protects published migration blobs.
func TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade(t *testing.T) {
	filenames := []string{
		"191_passkey_credentials.sql",
		"191_subscription_quota_advance_receipts.sql",
		"192_subscription_cache_invalidation_outbox.sql",
		"192_group_profit_control.sql",
		"193_group_profit_control_auth_cache_invalidation.sql",
	}

	completeFS := dbmigrations.FS
	baselineFS := fstest.MapFS{}
	expectedChecksums := make(map[string]string, len(filenames))
	excludedFromBaseline := map[string]struct{}{
		"192_group_profit_control.sql":                         {},
		"193_group_profit_control_auth_cache_invalidation.sql": {},
	}
	files, err := fs.Glob(completeFS, "*.sql")
	require.NoError(t, err)
	for _, filename := range files {
		content, err := completeFS.ReadFile(filename)
		require.NoError(t, err)
		if _, excluded := excludedFromBaseline[filename]; !excluded {
			baselineFS[filename] = &fstest.MapFile{Data: content}
		}
		for _, expectedFilename := range filenames {
			if filename == expectedFilename {
				sum := sha256.Sum256([]byte(strings.TrimSpace(string(content))))
				expectedChecksums[filename] = hex.EncodeToString(sum[:])
			}
		}
	}
	require.Contains(t, baselineFS, "191_passkey_credentials.sql")
	require.Contains(t, baselineFS, "191_subscription_quota_advance_receipts.sql")
	require.Contains(t, baselineFS, "192_subscription_cache_invalidation_outbox.sql")
	require.NotContains(t, baselineFS, "192_group_profit_control.sql")
	require.NotContains(t, baselineFS, "193_group_profit_control_auth_cache_invalidation.sql")
	require.Len(t, expectedChecksums, len(filenames))

	ctx := context.Background()
	profitColumns := []struct {
		name        string
		dataType    string
		defaultKind string
	}{
		{"profit_control_enabled", "boolean", "false"},
		{"profit_min_margin", "numeric", "zero"},
		{"profit_safety_buffer", "numeric", "zero"},
	}
	assertProfitColumnsAbsent := func(db *sql.DB) {
		t.Helper()
		for _, expected := range profitColumns {
			var exists bool
			require.NoError(t, db.QueryRowContext(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM information_schema.columns
	WHERE table_schema = 'public'
	  AND table_name = 'groups'
	  AND column_name = $1
)`, expected.name).Scan(&exists))
			require.False(t, exists, "expected baseline groups.%s to be absent", expected.name)
		}
	}
	assertProfitColumns := func(db *sql.DB) {
		t.Helper()
		for _, expected := range profitColumns {
			var dataType, nullable string
			var defaultValue sql.NullString
			require.NoError(t, db.QueryRowContext(ctx, `
SELECT data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'groups'
  AND column_name = $1`, expected.name).Scan(&dataType, &nullable, &defaultValue))
			require.Equal(t, expected.dataType, dataType)
			require.Equal(t, "NO", nullable)
			require.True(t, defaultValue.Valid, "expected groups.%s to have a default", expected.name)
			var evaluatedDefault string
			require.NoError(t, db.QueryRowContext(ctx, fmt.Sprintf("SELECT (%s)::text", defaultValue.String)).Scan(&evaluatedDefault))
			if expected.defaultKind == "false" {
				require.Equal(t, "false", evaluatedDefault)
				continue
			}
			defaultNumber, ok := new(big.Rat).SetString(evaluatedDefault)
			require.True(t, ok, "expected groups.%s default to be numeric, got %q", expected.name, evaluatedDefault)
			require.Zero(t, defaultNumber.Sign(), "expected groups.%s default to equal zero", expected.name)
		}
	}
	assertProfitFunction := func(db *sql.DB, expected bool) {
		t.Helper()
		var definition string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT pg_get_functiondef('enqueue_group_auth_cache_invalidation()'::regprocedure)").Scan(&definition))
		for _, column := range profitColumns {
			if expected {
				require.Contains(t, definition, column.name)
				continue
			}
			require.NotContains(t, definition, column.name)
		}
	}
	assertCompleteState := func(db *sql.DB) {
		t.Helper()
		for _, filename := range filenames {
			var count int
			var checksum string
			require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*), MIN(checksum) FROM schema_migrations WHERE filename = $1", filename).Scan(&count, &checksum))
			require.Equal(t, 1, count, "expected one schema_migrations row for %s", filename)
			require.Equal(t, expectedChecksums[filename], checksum)
		}
		assertProfitColumns(db)
		assertProfitFunction(db, true)
		for _, relation := range []string{
			"passkey_user_handles",
			"passkey_credentials",
			"subscription_quota_advance_receipts",
			"subscription_cache_invalidation_outbox",
			"subscription_cache_version_watermarks",
		} {
			var regclass sql.NullString
			require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass($1)", "public."+relation).Scan(&regclass))
			require.True(t, regclass.Valid, "expected %s to exist", relation)
		}
	}

	upgradeDB := newEmptyIsolatedMigrationDB(t)
	// baselineFS -> completeFS -> completeFS mirrors the v0.1.170 upgrade and idempotency path.
	require.NoError(t, applyMigrationsFS(ctx, upgradeDB, baselineFS))
	for _, filename := range filenames[:3] {
		var checksum string
		require.NoError(t, upgradeDB.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename = $1", filename).Scan(&checksum))
		require.Equal(t, expectedChecksums[filename], checksum)
	}
	for _, filename := range filenames[3:] {
		var count int
		require.NoError(t, upgradeDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE filename = $1", filename).Scan(&count))
		require.Zero(t, count, "expected %s to be absent from the baseline", filename)
	}
	assertProfitColumnsAbsent(upgradeDB)
	assertProfitFunction(upgradeDB, false)
	require.NoError(t, applyMigrationsFS(ctx, upgradeDB, completeFS))
	require.NoError(t, applyMigrationsFS(ctx, upgradeDB, completeFS))
	assertCompleteState(upgradeDB)

	emptyDB := newEmptyIsolatedMigrationDB(t)
	require.NoError(t, applyMigrationsFS(ctx, emptyDB, completeFS))
	require.NoError(t, applyMigrationsFS(ctx, emptyDB, completeFS))
	assertCompleteState(emptyDB)
}
