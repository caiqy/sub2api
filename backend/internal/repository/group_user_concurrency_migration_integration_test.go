//go:build integration

package repository

import (
	"context"
	"testing"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestMigration108AppliesTwiceToPartialGroupsSchema(t *testing.T) {
	db := newEmptyIsolatedMigrationDB(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `
CREATE TABLE groups (
    id BIGINT PRIMARY KEY,
    user_concurrency_enabled BOOLEAN NOT NULL DEFAULT FALSE
)
`)
	require.NoError(t, err)

	migrationSQL, err := dbmigrations.FS.ReadFile("108_add_group_user_concurrency.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var columns int
	err = db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'groups'
  AND column_name IN ('user_concurrency_enabled', 'user_concurrency_limit')
`).Scan(&columns)
	require.NoError(t, err)
	require.Equal(t, 2, columns)
}
