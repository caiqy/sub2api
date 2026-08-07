//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

// filtered FS 模拟从 main@16c07d806（0.1.169.3）本地 migration 集合升级；历史 migration 不变性由 build ledger 的 Git blob OID 断言单独证明。
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
	assertCompleteState := func(db *sql.DB) {
		t.Helper()
		for _, filename := range filenames {
			var count int
			var checksum string
			require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*), MIN(checksum) FROM schema_migrations WHERE filename = $1", filename).Scan(&count, &checksum))
			require.Equal(t, 1, count, "expected one schema_migrations row for %s", filename)
			require.Equal(t, expectedChecksums[filename], checksum)
		}
		for _, relation := range []string{
			"passkey_user_handles",
			"passkey_credentials",
			"subscription_quota_advance_receipts",
			"subscription_cache_invalidation_outbox",
			"subscription_cache_version_watermarks",
			"groups",
			"auth_cache_invalidation_outbox",
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
	require.NoError(t, applyMigrationsFS(ctx, upgradeDB, completeFS))
	require.NoError(t, applyMigrationsFS(ctx, upgradeDB, completeFS))
	assertCompleteState(upgradeDB)

	emptyDB := newEmptyIsolatedMigrationDB(t)
	require.NoError(t, applyMigrationsFS(ctx, emptyDB, completeFS))
	require.NoError(t, applyMigrationsFS(ctx, emptyDB, completeFS))
	assertCompleteState(emptyDB)
}
