//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate(t *testing.T) {
	db := newIsolatedMigrationDB(t)
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	// Re-apply migrations to verify idempotency (no errors, no duplicate rows).
	require.NoError(t, ApplyMigrations(context.Background(), db))

	// schema_migrations should have at least the current migration set.
	var applied int
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM schema_migrations").Scan(&applied))
	require.GreaterOrEqual(t, applied, 7, "expected schema_migrations to contain applied migrations")
	requireMigrationApplied(t, tx, "077_add_usage_log_details.sql")

	// users: columns required by repository queries
	requireColumn(t, tx, "users", "username", "character varying", 100, false)
	requireColumn(t, tx, "users", "notes", "text", 0, false)

	// accounts: schedulable and rate-limit fields
	requireColumn(t, tx, "accounts", "notes", "text", 0, true)
	requireColumn(t, tx, "accounts", "schedulable", "boolean", 0, false)
	requireColumn(t, tx, "accounts", "rate_limited_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "rate_limit_reset_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "overload_until", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "session_window_status", "character varying", 20, true)

	// api_keys: key length should be 128
	requireColumn(t, tx, "api_keys", "key", "character varying", 128, false)

	// redeem_codes: subscription fields
	requireColumn(t, tx, "redeem_codes", "group_id", "bigint", 0, true)
	requireColumn(t, tx, "redeem_codes", "validity_days", "integer", 0, false)

	// usage_logs: billing_type used by filters/stats
	requireColumn(t, tx, "usage_logs", "billing_type", "smallint", 0, false)
	requireColumn(t, tx, "usage_logs", "request_type", "smallint", 0, false)
	requireColumn(t, tx, "usage_logs", "openai_ws_mode", "boolean", 0, false)

	// usage_log_details: detail snapshot table for payload retention
	var usageLogDetailsRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.usage_log_details')").Scan(&usageLogDetailsRegclass))
	require.True(t, usageLogDetailsRegclass.Valid, "expected usage_log_details table to exist")
	requireColumn(t, tx, "usage_log_details", "usage_log_id", "bigint", 0, false)
	requireColumn(t, tx, "usage_log_details", "request_headers", "text", 0, false)
	requireColumn(t, tx, "usage_log_details", "request_body", "text", 0, false)
	requireColumn(t, tx, "usage_log_details", "response_headers", "text", 0, false)
	requireColumn(t, tx, "usage_log_details", "response_body", "text", 0, false)
	requireColumn(t, tx, "usage_log_details", "created_at", "timestamp with time zone", 0, false)
	requireUniqueConstraintOnColumn(t, tx, "usage_log_details", "usage_log_id")
	requireIndexOnColumn(t, tx, "usage_log_details", "created_at")
	requireForeignKey(t, tx, "usage_log_details", "usage_log_id", "usage_logs", "id", true)

	// usage_billing_dedup: billing idempotency narrow table
	var usageBillingDedupRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.usage_billing_dedup')").Scan(&usageBillingDedupRegclass))
	require.True(t, usageBillingDedupRegclass.Valid, "expected usage_billing_dedup table to exist")
	requireColumn(t, tx, "usage_billing_dedup", "request_fingerprint", "character varying", 64, false)
	requireIndex(t, tx, "usage_billing_dedup", "idx_usage_billing_dedup_request_api_key")
	requireIndex(t, tx, "usage_billing_dedup", "idx_usage_billing_dedup_created_at_brin")

	var usageBillingDedupArchiveRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.usage_billing_dedup_archive')").Scan(&usageBillingDedupArchiveRegclass))
	require.True(t, usageBillingDedupArchiveRegclass.Valid, "expected usage_billing_dedup_archive table to exist")
	requireColumn(t, tx, "usage_billing_dedup_archive", "request_fingerprint", "character varying", 64, false)
	requireIndex(t, tx, "usage_billing_dedup_archive", "usage_billing_dedup_archive_pkey")

	// settings table should exist
	var settingsRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.settings')").Scan(&settingsRegclass))
	require.True(t, settingsRegclass.Valid, "expected settings table to exist")

	// security_secrets table should exist
	var securitySecretsRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.security_secrets')").Scan(&securitySecretsRegclass))
	require.True(t, securitySecretsRegclass.Valid, "expected security_secrets table to exist")

	// user_allowed_groups table should exist
	var uagRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.user_allowed_groups')").Scan(&uagRegclass))
	require.True(t, uagRegclass.Valid, "expected user_allowed_groups table to exist")

	// user_subscriptions: deleted_at for soft delete support (migration 012)
	requireColumn(t, tx, "user_subscriptions", "deleted_at", "timestamp with time zone", 0, true)

	// orphan_allowed_groups_audit table should exist (migration 013)
	var orphanAuditRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.orphan_allowed_groups_audit')").Scan(&orphanAuditRegclass))
	require.True(t, orphanAuditRegclass.Valid, "expected orphan_allowed_groups_audit table to exist")

	// account_groups: created_at should be timestamptz
	requireColumn(t, tx, "account_groups", "created_at", "timestamp with time zone", 0, false)

	// user_allowed_groups: created_at should be timestamptz
	requireColumn(t, tx, "user_allowed_groups", "created_at", "timestamp with time zone", 0, false)
}

func newIsolatedMigrationDB(t *testing.T) *sql.DB {
	t.Helper()
	require.NotEmpty(t, integrationDSN, "expected integration dsn to be initialized")

	adminDSN, dbName := isolatedPostgresDSNs(t)
	adminDB, err := openSQLWithRetry(context.Background(), adminDSN, 30*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = adminDB.Close()
	})

	_, err = adminDB.ExecContext(context.Background(), fmt.Sprintf("CREATE DATABASE %s", pqQuoteIdentifier(dbName)))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", pqQuoteIdentifier(dbName)))
	})

	testDSN := isolatedDatabaseDSN(t, dbName)
	db, err := openSQLWithRetry(context.Background(), testDSN, 30*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	require.NoError(t, ApplyMigrations(context.Background(), db))
	return db
}

func isolatedPostgresDSNs(t *testing.T) (adminDSN, dbName string) {
	t.Helper()

	parsed, err := url.Parse(integrationDSN)
	require.NoError(t, err)
	require.NotEmpty(t, strings.TrimPrefix(parsed.Path, "/"), "expected database name in dsn")

	dbName = fmt.Sprintf("sub2api_migrations_%d", time.Now().UnixNano())
	parsed.Path = "/postgres"
	return parsed.String(), dbName
}

func isolatedDatabaseDSN(t *testing.T, dbName string) string {
	t.Helper()

	parsed, err := url.Parse(integrationDSN)
	require.NoError(t, err)
	parsed.Path = "/" + dbName
	return parsed.String()
}

func pqQuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func requireIndex(t *testing.T, tx *sql.Tx, table, index string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1
	FROM pg_indexes
	WHERE schemaname = 'public'
	  AND tablename = $1
	  AND indexname = $2
)
`, table, index).Scan(&exists)
	require.NoError(t, err, "query pg_indexes for %s.%s", table, index)
	require.True(t, exists, "expected index %s on %s", index, table)
}

func requireColumn(t *testing.T, tx *sql.Tx, table, column, dataType string, maxLen int, nullable bool) {
	t.Helper()

	var row struct {
		DataType string
		MaxLen   sql.NullInt64
		Nullable string
	}

	err := tx.QueryRowContext(context.Background(), `
SELECT
  data_type,
  character_maximum_length,
  is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = $1
  AND column_name = $2
`, table, column).Scan(&row.DataType, &row.MaxLen, &row.Nullable)
	require.NoError(t, err, "query information_schema.columns for %s.%s", table, column)
	require.Equal(t, dataType, row.DataType, "data_type mismatch for %s.%s", table, column)

	if maxLen > 0 {
		require.True(t, row.MaxLen.Valid, "expected maxLen for %s.%s", table, column)
		require.Equal(t, int64(maxLen), row.MaxLen.Int64, "maxLen mismatch for %s.%s", table, column)
	}

	if nullable {
		require.Equal(t, "YES", row.Nullable, "nullable mismatch for %s.%s", table, column)
	} else {
		require.Equal(t, "NO", row.Nullable, "nullable mismatch for %s.%s", table, column)
	}
}

func requireMigrationApplied(t *testing.T, tx *sql.Tx, filename string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1 FROM schema_migrations WHERE filename = $1
)
`, filename).Scan(&exists)
	require.NoError(t, err, "query schema_migrations for %s", filename)
	require.True(t, exists, "expected migration %s to be applied", filename)
}

func requireUniqueConstraintOnColumn(t *testing.T, tx *sql.Tx, table, column string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1
	FROM pg_constraint c
	JOIN pg_class tbl ON tbl.oid = c.conrelid
	JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
	JOIN unnest(c.conkey) WITH ORDINALITY AS cols(attnum, ord) ON TRUE
	JOIN pg_attribute attr ON attr.attrelid = tbl.oid AND attr.attnum = cols.attnum
	WHERE ns.nspname = 'public'
	  AND tbl.relname = $1
	  AND c.contype = 'u'
	GROUP BY c.oid
	HAVING COUNT(*) = 1 AND BOOL_AND(attr.attname = $2)
)
`, table, column).Scan(&exists)
	require.NoError(t, err, "query unique constraint for %s.%s", table, column)
	require.True(t, exists, "expected unique constraint on %s.%s", table, column)
}

func requireIndexOnColumn(t *testing.T, tx *sql.Tx, table, column string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1
	FROM pg_indexes
	WHERE schemaname = 'public'
	  AND tablename = $1
	  AND indexdef ILIKE '%' || quote_ident($2) || '%'
)
`, table, column).Scan(&exists)
	require.NoError(t, err, "query index for %s.%s", table, column)
	require.True(t, exists, "expected index on %s.%s", table, column)
}

func requireForeignKey(t *testing.T, tx *sql.Tx, table, column, refTable, refColumn string, onDeleteCascade bool) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1
	FROM pg_constraint c
	JOIN pg_class tbl ON tbl.oid = c.conrelid
	JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
	JOIN pg_class ref_tbl ON ref_tbl.oid = c.confrelid
	JOIN pg_attribute attr ON attr.attrelid = tbl.oid AND attr.attnum = c.conkey[1]
	JOIN pg_attribute ref_attr ON ref_attr.attrelid = ref_tbl.oid AND ref_attr.attnum = c.confkey[1]
	WHERE ns.nspname = 'public'
	  AND c.contype = 'f'
	  AND array_length(c.conkey, 1) = 1
	  AND array_length(c.confkey, 1) = 1
	  AND tbl.relname = $1
	  AND attr.attname = $2
	  AND ref_tbl.relname = $3
	  AND ref_attr.attname = $4
	  AND ($5 = FALSE OR c.confdeltype = 'c')
)
`, table, column, refTable, refColumn, onDeleteCascade).Scan(&exists)
	require.NoError(t, err, "query foreign key for %s.%s", table, column)
	require.True(t, exists, "expected foreign key on %s.%s referencing %s.%s", table, column, refTable, refColumn)
}
