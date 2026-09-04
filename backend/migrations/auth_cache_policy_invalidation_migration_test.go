package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthCachePolicyInvalidationMigration(t *testing.T) {
	content, err := FS.ReadFile("234_auth_cache_policy_invalidation.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "OLD.restrict_public_groups IS NOT DISTINCT FROM NEW.restrict_public_groups")
	require.Contains(t, sql, "OLD.force_openai_fast IS NOT DISTINCT FROM NEW.force_openai_fast")
	require.Contains(t, sql, "OLD.free_openai_fast IS NOT DISTINCT FROM NEW.free_openai_fast")
	require.Contains(t, sql, "OLD.max_reasoning_effort_over_limit IS NOT DISTINCT FROM NEW.max_reasoning_effort_over_limit")
	require.Contains(t, sql, "CREATE OR REPLACE FUNCTION enqueue_allowed_group_auth_cache_invalidation()")
	require.Contains(t, sql, "WHERE k.user_id = target_user_id")
	require.NotContains(t, sql, "g.is_exclusive")
}
