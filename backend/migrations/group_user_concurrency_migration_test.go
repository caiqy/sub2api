package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration108GroupUserConcurrencyIsIdempotent(t *testing.T) {
	content, err := FS.ReadFile("108_add_group_user_concurrency.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS user_concurrency_enabled")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS user_concurrency_limit")
}
