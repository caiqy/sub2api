package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestRemainingQuotaExclusionsSurviveDatabaseReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "quota-settings.sqlite")
	open := func() (*dbent.Client, *service.SettingService) {
		db, err := sql.Open("sqlite", path)
		require.NoError(t, err)
		db.SetMaxOpenConns(1)
		_, err = db.Exec("PRAGMA foreign_keys = ON")
		require.NoError(t, err)
		client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.SQLite, db)))
		t.Cleanup(func() { _ = client.Close() })
		return client, service.NewSettingService(NewSettingRepository(client), &config.Config{})
	}
	client, settings := open()
	require.NoError(t, client.Schema.Create(ctx))
	ids, err := settings.GetRemainingQuotaExclusions(ctx)
	require.NoError(t, err)
	require.Equal(t, []int64{}, ids)
	ids, err = settings.SetRemainingQuotaExclusions(ctx, []int64{7, 2, 7})
	require.NoError(t, err)
	require.Equal(t, []int64{7, 2}, ids)
	require.NoError(t, client.Close())
	client, settings = open()
	ids, err = settings.GetRemainingQuotaExclusions(ctx)
	require.NoError(t, err)
	require.Equal(t, []int64{7, 2}, ids)
	another := service.NewSettingService(NewSettingRepository(client), &config.Config{})
	_, err = another.SetRemainingQuotaExclusions(ctx, []int64{})
	require.NoError(t, err)
	ids, err = settings.GetRemainingQuotaExclusions(ctx)
	require.NoError(t, err)
	require.Equal(t, []int64{}, ids)
	require.NoError(t, client.Close())
	_, settings = open()
	ids, err = settings.GetRemainingQuotaExclusions(ctx)
	require.NoError(t, err)
	require.Equal(t, []int64{}, ids)
}
