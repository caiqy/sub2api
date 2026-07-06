package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

type pageSettingRepoStub struct{ values map[string]string }

func (s *pageSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}
func (s *pageSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}
func (s *pageSettingRepoStub) Set(context.Context, string, string) error { return nil }
func (s *pageSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}
func (s *pageSettingRepoStub) SetMultiple(context.Context, map[string]string) error { return nil }
func (s *pageSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}
func (s *pageSettingRepoStub) Delete(context.Context, string) error { return nil }

func TestPageHandlerGetPageContentRejectsHiddenCustomMenu(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "pages"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "pages", "docs.md"), []byte("# docs"), 0644))

	settings := &pageSettingRepoStub{values: map[string]string{
		service.SettingKeyCustomMenuItems: `[{"id":"docs","label":"Docs","url":"md:docs","visibility":"user"}]`,
	}}
	db, err := sql.Open("sqlite", "file:page_hidden_custom_menu?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE user_avatars (
		user_id INTEGER PRIMARY KEY,
		storage_provider TEXT NOT NULL,
		storage_key TEXT NOT NULL DEFAULT '',
		url TEXT NOT NULL,
		content_type TEXT NOT NULL,
		byte_size INTEGER NOT NULL,
		sha256 TEXT NOT NULL
	)`)
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	user, err := client.User.Create().
		SetEmail("hidden-custom-menu@example.com").
		SetPasswordHash("hash").
		SetUsername("hidden-custom-menu").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(context.Background())
	require.NoError(t, err)
	require.NoError(t, client.UserResourceOverride.Create().
		SetUserID(user.ID).
		SetResourceType("ui").
		SetResourceID(service.CustomMenuResourceID("docs")).
		SetEffect("deny").
		Exec(context.Background()))

	settingSvc := service.NewSettingService(settings, &config.Config{})
	userSvc := service.NewUserService(repository.NewUserRepository(client, db), settings, nil, nil)
	h := NewPageHandler(dataDir, settingSvc, userSvc)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "slug", Value: "docs"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/pages/docs", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	h.GetPageContent(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestPageHandlerGetPageContentAllowsAdminWithHiddenCustomMenu(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "pages"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "pages", "docs.md"), []byte("# docs"), 0644))

	settings := &pageSettingRepoStub{values: map[string]string{
		service.SettingKeyCustomMenuItems: `[{"id":"docs","label":"Docs","url":"md:docs","visibility":"user"}]`,
	}}
	settingSvc := service.NewSettingService(settings, &config.Config{})
	h := NewPageHandler(dataDir, settingSvc, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "slug", Value: "docs"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/pages/docs", nil)
	c.Set(string(middleware2.ContextKeyUserRole), service.RoleAdmin)

	h.GetPageContent(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "# docs", recorder.Body.String())
}
