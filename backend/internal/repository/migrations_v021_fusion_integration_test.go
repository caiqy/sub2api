//go:build integration

package repository

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestMigrationsV021Fusion(t *testing.T) {
	ctx := context.Background()
	const ttlMigration = "250_openai_cache_write_ttl.sql"
	files, err := fs.Glob(dbmigrations.FS, "*.sql")
	require.NoError(t, err)
	baseline := fstest.MapFS{}
	var baselineNames []string
	for _, name := range files {
		if name == usageLogsAPIKeyCostIndexMigration || name == ttlMigration {
			continue
		}
		content, err := fs.ReadFile(dbmigrations.FS, name)
		require.NoError(t, err)
		baseline[name] = &fstest.MapFile{Data: content}
		baselineNames = append(baselineNames, name)
	}

	t.Run("empty", func(t *testing.T) {
		db := newEmptyIsolatedMigrationDB(t)
		require.NoError(t, ApplyMigrations(ctx, db))
		checksums, count := migrationChecksumsAndCount(t, db, files)
		require.Len(t, checksums, len(files))
		require.NoError(t, ApplyMigrations(ctx, db))
		after, afterCount := migrationChecksumsAndCount(t, db, files)
		require.Equal(t, checksums, after)
		require.Equal(t, count, afterCount)
	})

	t.Run("fork253_upgrade_and_invalid_index_recovery", func(t *testing.T) {
		db := newEmptyIsolatedMigrationDB(t)
		require.NoError(t, applyMigrationsFS(ctx, db, baseline))
		before, count := migrationChecksumsAndCount(t, db, baselineNames)
		require.Contains(t, before, "253_enable_user_traffic_billing_by_default.sql")
		require.Contains(t, before, "249_group_exclusive_rate_models.sql")
		require.Contains(t, before, "250_user_cyber_overflow_exempt.sql")
		require.NoError(t, ApplyMigrations(ctx, db))
		after, afterCount := migrationChecksumsAndCount(t, db, baselineNames)
		require.Equal(t, before, after, "historical migration checksums must remain unchanged")
		require.Equal(t, count+2, afterCount)
		var short, long int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE column_name = 'cache_creation_30m_tokens'), COUNT(*) FILTER (WHERE column_name = 'cache_creation_12h_tokens') FROM information_schema.columns WHERE table_name = 'usage_logs'`).Scan(&short, &long))
		require.Equal(t, 1, short)
		require.Equal(t, 1, long)

		// A failed concurrent unique build leaves a real invalid index for the runner to recover.
		_, err := db.ExecContext(ctx, "DROP INDEX "+usageLogsAPIKeyCostIndex)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, "DELETE FROM schema_migrations WHERE filename = $1", usageLogsAPIKeyCostIndexMigration)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, "CREATE TABLE invalid_index_fixture (value integer); INSERT INTO invalid_index_fixture VALUES (1), (1)")
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, "CREATE UNIQUE INDEX CONCURRENTLY "+usageLogsAPIKeyCostIndex+" ON invalid_index_fixture (value)")
		require.Error(t, err)
		invalid, err := indexIsInvalid(ctx, db, usageLogsAPIKeyCostIndex)
		require.NoError(t, err)
		require.True(t, invalid)
		require.NoError(t, ApplyMigrations(ctx, db))
		var valid bool
		var table, definition string
		require.NoError(t, db.QueryRowContext(ctx, `SELECT i.indisvalid, i.indrelid::regclass::text, pg_get_indexdef(i.indexrelid) FROM pg_index i WHERE i.indexrelid = $1::regclass`, usageLogsAPIKeyCostIndex).Scan(&valid, &table, &definition))
		require.True(t, valid)
		require.Equal(t, "usage_logs", table)
		require.Contains(t, definition, "INCLUDE (actual_cost)")
		require.NoError(t, ApplyMigrations(ctx, db))
		final, finalCount := migrationChecksumsAndCount(t, db, baselineNames)
		require.Equal(t, before, final)
		require.Equal(t, afterCount, finalCount)
	})
}
