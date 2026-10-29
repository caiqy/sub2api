//go:build unit

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestExclusiveRateModelsPostgres(t *testing.T) {
	dsn := os.Getenv("EXCLUSIVE_RATE_TEST_DSN")
	if dsn == "" {
		t.Skip("requires independent EXCLUSIVE_RATE_TEST_DSN")
	}
	ctx := context.Background()
	base, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer base.Close()
	database := fmt.Sprintf("exclusive_rate_test_%d", time.Now().UnixNano())
	_, err = base.Exec(`CREATE DATABASE ` + database)
	require.NoError(t, err)
	defer base.Exec(`DROP DATABASE ` + database)
	db, err := sql.Open("postgres", dsn+" dbname="+database)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, ApplyMigrations(ctx, db))
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	defer client.Close()
	groups := NewGroupRepository(client, db)
	group := &service.Group{Name: "rate-test", Platform: service.PlatformOpenAI, RateMultiplier: 0.225, Status: service.StatusActive}
	require.NoError(t, groups.Create(ctx, group))
	require.False(t, group.ExclusiveRateModels.Enabled)
	user, err := client.User.Create().SetEmail("rates@example.com").SetPasswordHash("not-a-real-password").Save(ctx)
	require.NoError(t, err)
	keyRepo := NewAPIKeyRepository(client, db)
	key := &service.APIKey{UserID: user.ID, GroupID: &group.ID, Key: "sk-exclusive-rate-test", Name: "test", Status: service.StatusActive}
	require.NoError(t, keyRepo.Create(ctx, key))
	repo := &userGroupRateRepository{sql: db}
	cfg := service.GroupExclusiveRateModels{Enabled: true, Models: []string{"client-A", "client-B"}}
	entries := []service.GroupRateMultiplierInput{{UserID: user.ID, RateMultiplier: 0.12}}
	require.NoError(t, repo.SyncGroupRateMultipliersWithModels(ctx, group.ID, entries, cfg))
	auth, err := keyRepo.GetByKeyForAuth(ctx, key.Key)
	require.NoError(t, err)
	require.Equal(t, cfg, auth.Group.ExclusiveRateModels, "auth projection and group conversion must preserve both fields")
	loaded, err := groups.GetByIDLite(ctx, group.ID)
	require.NoError(t, err)
	require.Equal(t, cfg, loaded.ExclusiveRateModels)
	var invalidations int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM auth_cache_invalidation_outbox`).Scan(&invalidations))
	require.Greater(t, invalidations, 0)
	// A legacy rates-only caller preserves the shared policy.
	require.NoError(t, repo.SyncGroupRateMultipliers(ctx, group.ID, entries))
	loaded, err = groups.GetByIDLite(ctx, group.ID)
	require.NoError(t, err)
	require.Equal(t, cfg, loaded.ExclusiveRateModels)
	// A bad foreign key after the group UPDATE must roll back both writes.
	bad := service.GroupExclusiveRateModels{Enabled: false, Models: []string{"keep-draft"}}
	err = repo.SyncGroupRateMultipliersWithModels(ctx, group.ID, []service.GroupRateMultiplierInput{{UserID: -1, RateMultiplier: 0.5}}, bad)
	require.Error(t, err)
	loaded, err = groups.GetByIDLite(ctx, group.ID)
	require.NoError(t, err)
	require.Equal(t, cfg, loaded.ExclusiveRateModels)
	rate, err := repo.GetByUserAndGroup(ctx, user.ID, group.ID)
	require.NoError(t, err)
	require.NotNil(t, rate)
	require.Equal(t, 0.12, *rate)
	require.NoError(t, repo.SyncGroupRateMultipliersWithModels(ctx, group.ID, entries, service.GroupExclusiveRateModels{Enabled: false, Models: cfg.Models}))
	loaded, err = groups.GetByIDLite(ctx, group.ID)
	require.NoError(t, err)
	require.False(t, loaded.ExclusiveRateModels.Enabled)
	require.Equal(t, cfg.Models, loaded.ExclusiveRateModels.Models)
}
