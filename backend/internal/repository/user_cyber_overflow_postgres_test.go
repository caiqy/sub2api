//go:build unit

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestUserCyberOverflowPostgres(t *testing.T) {
	dsn := os.Getenv("CYBER_OVERFLOW_TEST_DSN")
	if dsn == "" {
		t.Skip("requires independent CYBER_OVERFLOW_TEST_DSN")
	}
	ctx := context.Background()
	base, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer base.Close()
	database := fmt.Sprintf("cyber_overflow_test_%d", time.Now().UnixNano())
	_, err = base.Exec(`CREATE DATABASE ` + database)
	require.NoError(t, err)
	defer base.Exec(`DROP DATABASE ` + database)
	db, err := sql.Open("postgres", dsn+" dbname="+database)
	require.NoError(t, err)
	defer db.Close()
	legacy := fstest.MapFS{}
	names, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	for _, name := range names {
		if name == "250_user_cyber_overflow_exempt.sql" {
			continue
		}
		contents, err := fs.ReadFile(migrations.FS, name)
		require.NoError(t, err)
		legacy[name] = &fstest.MapFile{Data: contents}
	}
	require.NoError(t, applyMigrationsFS(ctx, db, legacy))
	var legacyID int64
	require.NoError(t, db.QueryRow(`INSERT INTO users(email,password_hash) VALUES('legacy@example.com','hash') RETURNING id`).Scan(&legacyID))
	require.NoError(t, ApplyMigrations(ctx, db))
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	defer client.Close()
	users := NewUserRepository(client, db)
	old, err := users.GetByID(ctx, legacyID)
	require.NoError(t, err)
	require.False(t, old.CyberOverflowExempt, "migration preserves blocking for existing users")
	fresh := &service.User{Email: "fresh@example.com", PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive}
	require.NoError(t, users.Create(ctx, fresh))
	require.False(t, fresh.CyberOverflowExempt)
	keys := NewAPIKeyRepository(client, db)
	for i, userID := range []int64{legacyID, legacyID, fresh.ID} {
		key := &service.APIKey{UserID: userID, Key: fmt.Sprintf("sk-cyber-%d", i), Name: "test", Status: service.StatusActive}
		require.NoError(t, keys.Create(ctx, key))
	}
	for _, exempt := range []bool{true, false} {
		old.CyberOverflowExempt = exempt
		require.NoError(t, users.Update(ctx, old, service.UserUpdateFields{CyberOverflowExempt: true}))
		loaded, err := users.GetByID(ctx, legacyID)
		require.NoError(t, err)
		require.Equal(t, exempt, loaded.CyberOverflowExempt)
		// A stale object updating unrelated fields must not overwrite the stored policy.
		require.NoError(t, users.Update(ctx, &service.User{ID: legacyID, Username: "renamed", CyberOverflowExempt: !exempt}, service.UserUpdateFields{Username: true}))
		for i := range 3 {
			auth, err := keys.GetByKeyForAuth(ctx, fmt.Sprintf("sk-cyber-%d", i))
			require.NoError(t, err)
			require.Equal(t, i < 2 && exempt, auth.User.CyberOverflowExempt, "each key uses only its owner's policy")
		}
	}
}
