//go:build integration

package repository

import (
	"context"
	"io/fs"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestUserTrafficBillingDefaultsMigration(t *testing.T) {
	ctx := context.Background()
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `CREATE TEMP TABLE users (id BIGSERIAL PRIMARY KEY, allow_large_requests boolean NOT NULL DEFAULT false) ON COMMIT DROP`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `INSERT INTO users (allow_large_requests) VALUES (false), (true)`)
	require.NoError(t, err)
	migration, err := fs.ReadFile(migrations.FS, "253_enable_user_traffic_billing_by_default.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	var disabled int
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE NOT allow_large_requests`).Scan(&disabled))
	require.Zero(t, disabled)
	var enabled bool
	require.NoError(t, tx.QueryRowContext(ctx, `INSERT INTO users DEFAULT VALUES RETURNING allow_large_requests`).Scan(&enabled))
	require.True(t, enabled)
}

func (s *UserRepoSuite) TestTrafficBillingDefaultsAndOptOut() {
	u := s.mustCreateUser(&service.User{Email: "traffic-default@test.com"})
	s.Require().True(u.AllowLargeRequests)
	loaded, err := s.repo.GetByID(s.ctx, u.ID)
	s.Require().NoError(err)
	s.Require().True(loaded.AllowLargeRequests)
	u.AllowLargeRequests = false
	s.Require().NoError(s.repo.Update(s.ctx, u, service.UserUpdateFields{AllowLargeRequests: true}))
	loaded, err = s.repo.GetByID(s.ctx, u.ID)
	s.Require().NoError(err)
	s.Require().False(loaded.AllowLargeRequests)
}
