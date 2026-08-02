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

// filtered FS 仅模拟从本地 0.1.165.4 集合升级；历史本地 migration 不变性由 build ledger 的 base-ref Git blob OID 断言单独证明。
func TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade(t *testing.T) {
	const passkeyMigration = "191_passkey_credentials.sql"
	filenames := []string{
		passkeyMigration,
		"191_subscription_quota_advance_receipts.sql",
		"192_subscription_cache_invalidation_outbox.sql",
	}

	completeFS := dbmigrations.FS
	baselineFS := fstest.MapFS{}
	expectedChecksums := make(map[string]string, len(filenames))
	files, err := fs.Glob(completeFS, "*.sql")
	require.NoError(t, err)
	for _, filename := range files {
		content, err := completeFS.ReadFile(filename)
		require.NoError(t, err)
		if filename != passkeyMigration {
			baselineFS[filename] = &fstest.MapFile{Data: content}
		}
		for _, expectedFilename := range filenames {
			if filename == expectedFilename {
				sum := sha256.Sum256([]byte(strings.TrimSpace(string(content))))
				expectedChecksums[filename] = hex.EncodeToString(sum[:])
			}
		}
	}
	require.Contains(t, baselineFS, "191_subscription_quota_advance_receipts.sql")
	require.Contains(t, baselineFS, "192_subscription_cache_invalidation_outbox.sql")
	require.NotContains(t, baselineFS, passkeyMigration)
	require.Len(t, expectedChecksums, len(filenames))

	ctx := context.Background()
	db := newEmptyIsolatedMigrationDB(t)

	// baselineFS -> completeFS -> completeFS mirrors the only supported upgrade path.
	require.NoError(t, applyMigrationsFS(ctx, db, baselineFS))
	for _, filename := range filenames[1:] {
		var checksum string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename = $1", filename).Scan(&checksum))
		require.Equal(t, expectedChecksums[filename], checksum)
	}
	var passkeyBaselineCount int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE filename = $1", passkeyMigration).Scan(&passkeyBaselineCount))
	require.Zero(t, passkeyBaselineCount)

	require.NoError(t, applyMigrationsFS(ctx, db, completeFS))
	for _, filename := range filenames {
		var checksum string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename = $1", filename).Scan(&checksum))
		require.Equal(t, expectedChecksums[filename], checksum)
	}

	require.NoError(t, applyMigrationsFS(ctx, db, completeFS))
	for _, filename := range filenames {
		var count int
		var checksum string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*), MIN(checksum) FROM schema_migrations WHERE filename = $1", filename).Scan(&count, &checksum))
		require.Equal(t, 1, count, "expected one schema_migrations row for %s", filename)
		require.Equal(t, expectedChecksums[filename], checksum)
	}
	for _, table := range []string{
		"passkey_credentials",
		"subscription_quota_advance_receipts",
		"subscription_cache_invalidation_outbox",
	} {
		var relation sql.NullString
		require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass($1)", "public."+table).Scan(&relation))
		require.True(t, relation.Valid, "expected %s to exist", table)
	}
}
