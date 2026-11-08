//go:build integration

package repository

import (
	"context"
	"io/fs"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestSessionVersionMigrationDefaults(t *testing.T) {
	ctx := context.Background()
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	// A temporary users table isolates a pre-migration row on this connection.
	_, err = tx.ExecContext(ctx, `CREATE TEMP TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT, password_hash TEXT) ON COMMIT DROP`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `INSERT INTO users(email,password_hash) VALUES ('legacy@example.com','unchanged')`)
	require.NoError(t, err)
	migration, err := fs.ReadFile(migrations.FS, "251_user_session_version.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	var epoch int64
	var hash string
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT session_version,password_hash FROM users WHERE id=1`).Scan(&epoch, &hash))
	require.Zero(t, epoch)
	require.Equal(t, "unchanged", hash)
	require.NoError(t, tx.QueryRowContext(ctx, `INSERT INTO users(email,password_hash) VALUES ('new@example.com','hash') RETURNING session_version`).Scan(&epoch))
	require.Zero(t, epoch)
}

func (s *UserRepoSuite) TestSessionRevocationPersistsAcrossInstances() {
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "session-version-integration-test", ExpireHour: 1}}
	first := service.NewAuthService(nil, s.repo, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	password, err := first.HashPassword("test-password")
	s.Require().NoError(err)
	u := s.mustCreateUser(&service.User{PasswordHash: password})
	s.Require().Zero(u.SessionVersion)
	old, err := first.GenerateToken(s.ctx, u)
	s.Require().NoError(err)
	// Independent repositories/services share only the database.
	secondRepo := newUserRepositoryWithSQL(s.client, integrationDB)
	second := service.NewAuthService(nil, secondRepo, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, auth := range []*service.AuthService{first, second} {
		wg.Add(1)
		go func(a *service.AuthService) { defer wg.Done(); results <- a.RevokeAllUserTokens(s.ctx, u.ID) }(auth)
	}
	wg.Wait()
	close(results)
	for err := range results {
		s.Require().NoError(err)
	}
	loaded, err := secondRepo.GetByID(s.ctx, u.ID)
	s.Require().NoError(err)
	s.Require().Equal(int64(2), loaded.SessionVersion)
	s.Require().Equal(password, loaded.PasswordHash)
	_, err = second.RefreshToken(s.ctx, old)
	s.Require().ErrorIs(err, service.ErrTokenRevoked)
	fresh, _, err := second.Login(s.ctx, u.Email, "test-password")
	s.Require().NoError(err)
	_, err = first.RefreshToken(s.ctx, fresh)
	s.Require().NoError(err)
	// An unrelated update from a stale snapshot cannot roll the generation back.
	u.Username = "new-name"
	s.Require().NoError(s.repo.Update(s.ctx, u, service.UserUpdateFields{Username: true}))
	loaded, err = secondRepo.GetByID(s.ctx, u.ID)
	s.Require().NoError(err)
	s.Require().Equal(int64(2), loaded.SessionVersion)
}
