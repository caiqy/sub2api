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
		// Seed legacy business rows before the two upstream migrations exist.
		_, err := db.ExecContext(ctx, `
INSERT INTO users (id, email, password_hash, balance) VALUES (900001, 'v021-upgrade@example.test', 'synthetic', 17.5);
INSERT INTO accounts (id, name, platform, type) VALUES (900001, 'v021-upgrade', 'openai', 'apikey');
INSERT INTO api_keys (id, user_id, key, name) VALUES (900001, 900001, 'synthetic-v021-upgrade', 'upgrade');
INSERT INTO usage_logs (id, user_id, api_key_id, account_id, request_id, model, input_tokens, output_tokens,
    cache_creation_tokens, cache_creation_5m_tokens, cache_creation_1h_tokens, actual_cost, traffic_cost,
    traffic_input_bytes, traffic_threshold_bytes, traffic_price_per_mb, traffic_billing_reason)
VALUES (900001, 900001, 900001, 900001, 'v021-before-upgrade', 'gpt-6-luna', 37, 11, 13, 5, 8,
    0.125, 0.25, 15000000, 10000000, 0.05, 'large_request');
INSERT INTO channels (id, name) VALUES (900001, 'v021-upgrade');
INSERT INTO channel_model_pricing (id, channel_id, models, input_price, cache_write_price)
VALUES (900001, 900001, '["configured"]', 0.000001, 0.000002), (900002, 900001, '["unset"]', NULL, NULL);
INSERT INTO channel_pricing_intervals (id, pricing_id, min_tokens, cache_write_price)
VALUES (900001, 900001, 100, 0.000003), (900002, 900002, 200, NULL);
INSERT INTO channel_account_stats_pricing_rules (id, channel_id, name) VALUES (900001, 900001, 'legacy');
INSERT INTO channel_account_stats_model_pricing (id, rule_id, models, cache_write_price)
VALUES (900001, 900001, '["configured"]', 0.000004), (900002, 900001, '["unset"]', NULL);
INSERT INTO channel_account_stats_pricing_intervals (id, pricing_id, min_tokens, cache_write_price)
VALUES (900001, 900001, 100, 0.000005), (900002, 900002, 200, NULL);`)
		require.NoError(t, err)
		pricingTables := []string{"channel_model_pricing", "channel_pricing_intervals", "channel_account_stats_model_pricing", "channel_account_stats_pricing_intervals"}
		tables := append([]string{"users", "accounts", "api_keys", "usage_logs", "channels", "channel_account_stats_pricing_rules"}, pricingTables...)
		snapshot := func(table string) string {
			t.Helper()
			var rows string
			require.NoError(t, db.QueryRowContext(ctx, `SELECT jsonb_agg(to_jsonb(r) - 'cache_creation_30m_tokens' - 'cache_creation_12h_tokens' - 'cache_write_12h_price' ORDER BY id)::text FROM `+table+` r WHERE id IN (900001, 900002)`).Scan(&rows))
			return rows
		}
		legacyRows := make(map[string]string, len(tables))
		for _, table := range tables {
			legacyRows[table] = snapshot(table)
		}
		assertLegacyRows := func() {
			t.Helper()
			for _, table := range tables {
				require.JSONEq(t, legacyRows[table], snapshot(table), table+" legacy rows must survive migration and replay")
			}
			var short, long int
			require.NoError(t, db.QueryRowContext(ctx, `SELECT cache_creation_30m_tokens, cache_creation_12h_tokens FROM usage_logs WHERE id = 900001`).Scan(&short, &long))
			require.Zero(t, short)
			require.Zero(t, long)
			for _, table := range pricingTables {
				var total, configured int
				require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(cache_write_12h_price) FROM `+table+` WHERE id IN (900001, 900002)`).Scan(&total, &configured))
				require.Equal(t, 2, total, table)
				require.Zero(t, configured, table+" new 12h price must remain NULL, including explicit generic prices")
			}
		}
		require.NoError(t, ApplyMigrations(ctx, db))
		after, afterCount := migrationChecksumsAndCount(t, db, baselineNames)
		require.Equal(t, before, after, "historical migration checksums must remain unchanged")
		require.Equal(t, count+2, afterCount)
		var short, long int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE column_name = 'cache_creation_30m_tokens'), COUNT(*) FILTER (WHERE column_name = 'cache_creation_12h_tokens') FROM information_schema.columns WHERE table_name = 'usage_logs'`).Scan(&short, &long))
		require.Equal(t, 1, short)
		require.Equal(t, 1, long)
		assertLegacyRows()

		// A failed concurrent unique build leaves a real invalid index for the runner to recover.
		_, err = db.ExecContext(ctx, "DROP INDEX "+usageLogsAPIKeyCostIndex)
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
		assertLegacyRows()
	})
}
