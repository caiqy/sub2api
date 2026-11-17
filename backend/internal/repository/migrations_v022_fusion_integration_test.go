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

func TestMigrationsV022Fusion(t *testing.T) {
	ctx := context.Background()
	files, err := fs.Glob(dbmigrations.FS, "*.sql")
	require.NoError(t, err)
	baseline := fstest.MapFS{}
	var names []string
	for _, name := range files {
		if name == "251_group_force_openai_ultrafast.sql" || name == "251_group_usage_rollup_nonblocking_build.sql" {
			continue
		}
		content, err := fs.ReadFile(dbmigrations.FS, name)
		require.NoError(t, err)
		baseline[name] = &fstest.MapFile{Data: content}
		names = append(names, name)
	}
	t.Run("fork_upgrade_and_replay", func(t *testing.T) {
		db := newEmptyIsolatedMigrationDB(t)
		require.NoError(t, applyMigrationsFS(ctx, db, baseline))
		before, count := migrationChecksumsAndCount(t, db, names)
		require.Contains(t, before, "253_enable_user_traffic_billing_by_default.sql")
		_, err := db.ExecContext(ctx, `
INSERT INTO groups (id, name, platform, force_openai_fast) VALUES (900022, 'v022-upgrade', 'openai', TRUE);
INSERT INTO users (id, email, password_hash, balance) VALUES (900022, 'v022@example.test', 'synthetic', 17.5);
INSERT INTO accounts (id, name, platform, type) VALUES (900022, 'v022-upgrade', 'openai', 'apikey');
INSERT INTO api_keys (id, user_id, key, name) VALUES (900022, 900022, 'synthetic-v022', 'upgrade');
INSERT INTO usage_logs (id, user_id, api_key_id, account_id, group_id, request_id, model, actual_cost, traffic_cost)
VALUES (900022, 900022, 900022, 900022, 900022, 'v022-before-upgrade', 'gpt-6-luna', 0.125, 0.25);`)
		require.NoError(t, err)
		snapshot := func(table string) string {
			t.Helper()
			var row string
			require.NoError(t, db.QueryRowContext(ctx, `SELECT (to_jsonb(r) - 'force_openai_ultrafast')::text FROM `+table+` r WHERE id=900022`).Scan(&row))
			return row
		}
		rows := map[string]string{}
		for _, table := range []string{"groups", "users", "accounts", "api_keys", "usage_logs"} {
			rows[table] = snapshot(table)
		}
		for replay := 0; replay < 2; replay++ {
			require.NoError(t, ApplyMigrations(ctx, db))
			after, afterCount := migrationChecksumsAndCount(t, db, names)
			require.Equal(t, before, after)
			require.Equal(t, count+2, afterCount)
			for table, row := range rows {
				require.JSONEq(t, row, snapshot(table), table)
			}
			var fast, ultrafast bool
			require.NoError(t, db.QueryRowContext(ctx, `SELECT force_openai_fast, force_openai_ultrafast FROM groups WHERE id=900022`).Scan(&fast, &ultrafast))
			require.True(t, fast)
			require.False(t, ultrafast)
			var columns int
			require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_name='usage_group_rollup_state' AND column_name IN ('expired_before', 'building_until', 'build_dirty_from')`).Scan(&columns))
			require.Equal(t, 3, columns)
		}
	})
}
