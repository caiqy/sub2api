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

// filtered FS simulates an upgrade from the required migration set through 193; Git-object identity evidence separately protects published migration blobs.
func TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade(t *testing.T) {
	filenames := []string{
		"191_passkey_credentials.sql",
		"191_subscription_quota_advance_receipts.sql",
		"192_subscription_cache_invalidation_outbox.sql",
		"192_group_profit_control.sql",
		"193_group_profit_control_auth_cache_invalidation.sql",
		"194_add_usage_log_upstream_response_model.sql",
		"195_add_usage_log_upstream_model_mismatch_index_notx.sql",
	}

	completeFS := dbmigrations.FS
	baselineFS := fstest.MapFS{}
	expectedChecksums := make(map[string]string, len(filenames))
	excludedFromBaseline := map[string]struct{}{
		"194_add_usage_log_upstream_response_model.sql":            {},
		"195_add_usage_log_upstream_model_mismatch_index_notx.sql": {},
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
	for _, filename := range filenames[:5] {
		require.Contains(t, baselineFS, filename)
	}
	for _, filename := range filenames[5:] {
		require.NotContains(t, baselineFS, filename)
	}
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
		for _, column := range []struct {
			name     string
			dataType string
			maxLen   int64
		}{
			{"upstream_response_model", "character varying", 200},
			{"upstream_model_mismatch", "boolean", 0},
		} {
			var actualType, nullable string
			var actualMaxLen sql.NullInt64
			require.NoError(t, db.QueryRowContext(ctx, `
SELECT data_type, character_maximum_length, is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'usage_logs'
  AND column_name = $1`, column.name).Scan(&actualType, &actualMaxLen, &nullable))
			require.Equal(t, column.dataType, actualType)
			require.Equal(t, "YES", nullable)
			if column.maxLen > 0 {
				require.True(t, actualMaxLen.Valid)
				require.Equal(t, column.maxLen, actualMaxLen.Int64)
			}
		}

		var mismatchIndexValid bool
		var mismatchIndexDefinition string
		require.NoError(t, db.QueryRowContext(ctx, `
SELECT i.indisvalid, pg_get_indexdef(i.indexrelid)
FROM pg_class idx
JOIN pg_index i ON i.indexrelid = idx.oid
JOIN pg_class tbl ON tbl.oid = i.indrelid
JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
WHERE ns.nspname = 'public'
  AND tbl.relname = 'usage_logs'
  AND idx.relname = $1`, usageLogsUpstreamModelMismatchIndex).Scan(&mismatchIndexValid, &mismatchIndexDefinition))
		require.True(t, mismatchIndexValid)
		require.Contains(t, mismatchIndexDefinition, "created_at DESC")
		require.Contains(t, mismatchIndexDefinition, "id DESC")
		require.Contains(t, mismatchIndexDefinition, "WHERE (upstream_model_mismatch IS TRUE)")
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
	// baselineFS -> completeFS -> completeFS applies 194/195 and verifies idempotency.
	require.NoError(t, applyMigrationsFS(ctx, upgradeDB, baselineFS))
	for _, filename := range filenames[:5] {
		var checksum string
		require.NoError(t, upgradeDB.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename = $1", filename).Scan(&checksum))
		require.Equal(t, expectedChecksums[filename], checksum)
	}
	for _, filename := range filenames[5:] {
		var count int
		require.NoError(t, upgradeDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE filename = $1", filename).Scan(&count))
		require.Zero(t, count, "expected %s to be absent from the baseline", filename)
	}
	for _, column := range []string{"upstream_response_model", "upstream_model_mismatch"} {
		var exists bool
		require.NoError(t, upgradeDB.QueryRowContext(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM information_schema.columns
	WHERE table_schema = 'public'
	  AND table_name = 'usage_logs'
	  AND column_name = $1
)`, column).Scan(&exists))
		require.False(t, exists, "expected baseline usage_logs.%s to be absent", column)
	}
	var mismatchIndex sql.NullString
	require.NoError(t, upgradeDB.QueryRowContext(ctx, "SELECT to_regclass($1)", "public."+usageLogsUpstreamModelMismatchIndex).Scan(&mismatchIndex))
	require.False(t, mismatchIndex.Valid, "expected baseline usage_logs mismatch index to be absent")
	require.NoError(t, applyMigrationsFS(ctx, upgradeDB, completeFS))
	require.NoError(t, applyMigrationsFS(ctx, upgradeDB, completeFS))
	assertCompleteState(upgradeDB)

	emptyDB := newEmptyIsolatedMigrationDB(t)
	require.NoError(t, applyMigrationsFS(ctx, emptyDB, completeFS))
	require.NoError(t, applyMigrationsFS(ctx, emptyDB, completeFS))
	assertCompleteState(emptyDB)
}
