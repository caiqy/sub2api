package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
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

type hiddenUIUserRepo struct {
	*oauthPendingFlowUserRepo
	user *service.User
}

func (r *hiddenUIUserRepo) GetByID(context.Context, int64) (*service.User, error) {
	user := *r.user
	return &user, nil
}

func (*hiddenUIUserRepo) GetUserAvatar(context.Context, int64) (*service.UserAvatar, error) {
	return nil, nil
}

func TestPageHandlerGetPageContentRejectsHiddenCustomMenu(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "pages"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "pages", "docs.md"), []byte("# docs"), 0644))

	settings := &pageSettingRepoStub{values: map[string]string{
		service.SettingKeyCustomMenuItems: `[{"id":"docs","label":"Docs","url":"md:docs","visibility":"user"}]`,
	}}
	user := &service.User{ID: 1, HiddenCustomMenuResourceIDs: []int64{service.CustomMenuResourceID("docs")}}

	settingSvc := service.NewSettingService(settings, &config.Config{})
	userSvc := service.NewUserService(&hiddenUIUserRepo{oauthPendingFlowUserRepo: &oauthPendingFlowUserRepo{}, user: user}, settings, nil, nil)
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
