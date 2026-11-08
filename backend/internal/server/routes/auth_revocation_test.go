package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func (r *mcpOAuthRouteUsers) IncrementSessionVersion(_ context.Context, id int64) error {
	if r.user == nil || r.user.ID != id {
		return service.ErrUserNotFound
	}
	r.user.SessionVersion++
	return nil
}

func (r *mcpOAuthRouteUsers) GetByEmail(ctx context.Context, email string) (*service.User, error) {
	if r.user == nil || r.user.Email != email {
		return nil, service.ErrUserNotFound
	}
	return r.GetByID(ctx, r.user.ID)
}

func TestHTTPRevokeAllRejectsOldJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	oauth, repo, users := newMCPRouteOAuthService(service.RoleUser)
	now := time.Now()
	users.user.LastActiveAt = &now
	cfg := &config.Config{}
	cfg.JWT.Secret = "review-secret-00000000000000000000000"
	cfg.JWT.ExpireHour = 1
	cfg.JWT.RefreshTokenExpireDays = 30
	cfg.MCP.PublicURL = "https://review.example/api/v1/mcp"
	settings := service.NewSettingService(&mcpRouteSettingsRepo{values: map[string]string{}}, cfg)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	cache := repository.NewRefreshTokenCache(rdb)
	auth := service.NewAuthService(nil, users, nil, cache, cfg, settings, nil, nil, nil, nil, nil, nil, nil)
	auth.SetMCPGrantRevoker(oauth)
	users.user.PasswordHash, _ = auth.HashPassword("test-password")
	oldSnapshot := *users.user
	pair, err := auth.GenerateTokenPair(ctx, users.user, "")
	require.NoError(t, err)
	us := service.NewUserService(users, nil, nil, nil)
	ah := handler.NewAuthHandler(cfg, auth, us, settings, nil, nil, nil, nil)
	mh := handler.NewMCPOAuthHandler(oauth, nil, us, settings, cfg)
	r := gin.New()
	secured := r.Group("", gin.HandlerFunc(middleware.NewJWTAuthMiddleware(auth, us, settings, nil)))
	secured.POST("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	secured.POST("/api/v1/auth/revoke-all-sessions", ah.RevokeAllSessions)
	secured.POST("/api/v1/user/mcp/authorize", mh.Decide)
	r.POST("/api/v1/mcp/oauth/token", mh.Token)
	call := func(path, token string, body []byte) *httptest.ResponseRecorder {
		q := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		q.Header.Set("Content-Type", "application/json")
		if path == "/api/v1/mcp/oauth/token" {
			q.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if token != "" {
			q.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, q)
		return w
	}
	require.Equal(t, http.StatusOK, call("/protected", pair.AccessToken, nil).Code)
	old := issueMCPRouteOAuthToken(t, oauth, cfg.MCP.PublicURL, "user", true)
	w := call("/api/v1/auth/revoke-all-sessions", pair.AccessToken, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	_, err = oauth.Authenticate(ctx, old)
	require.ErrorIs(t, err, service.ErrMCPOAuthInvalidToken)
	_, err = auth.RefreshTokenPair(ctx, pair.RefreshToken)
	require.Error(t, err, "refresh actually deleted")
	require.Equal(t, http.StatusUnauthorized, call("/protected", pair.AccessToken, nil).Code)
	client, err := oauth.Register(ctx, service.MCPOAuthRegisterInput{RedirectURIs: []string{"http://127.0.0.1:37777/callback"}})
	require.NoError(t, err)
	a := mcpOAuthRouteAuthorization(client.ClientID, cfg.MCP.PublicURL)
	b, _ := json.Marshal(service.MCPOAuthConsentInput{Authorization: a, Approved: true, Scope: "user"})
	w = call("/api/v1/user/mcp/authorize", pair.AccessToken, b)
	require.Equal(t, 401, w.Code, w.Body.String())
	// A second instance reads the persisted user, without process-local revocation state.
	auth2 := service.NewAuthService(nil, users, nil, cache, cfg, settings, nil, nil, nil, nil, nil, nil, nil)
	_, err = auth2.RefreshToken(ctx, pair.AccessToken)
	require.ErrorIs(t, err, service.ErrTokenRevoked)
	latePair, err := auth2.GenerateTokenPair(ctx, &oldSnapshot, "")
	require.NoError(t, err)
	require.Equal(t, 401, call("/api/v1/user/mcp/authorize", latePair.AccessToken, b).Code)
	_, err = auth2.RefreshTokenPair(ctx, latePair.RefreshToken)
	require.ErrorIs(t, err, service.ErrTokenRevoked)
	loginToken, loginUser, err := auth2.Login(ctx, users.user.Email, "test-password")
	require.NoError(t, err)
	require.Equal(t, 200, call("/api/v1/user/mcp/authorize", loginToken, b).Code)
	freshPair, err := auth2.GenerateTokenPair(ctx, loginUser, "")
	require.NoError(t, err)
	w = call("/api/v1/user/mcp/authorize", freshPair.AccessToken, b)
	require.Equal(t, 200, w.Code, w.Body.String())
	var response struct {
		Data struct {
			Redirect string `json:"redirect_url"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	u, err := url.Parse(response.Data.Redirect)
	require.NoError(t, err)
	b = []byte(url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ClientID}, "code": {u.Query().Get("code")}, "redirect_uri": {a.RedirectURI}, "code_verifier": {mcpOAuthRouteVerifier}, "resource": {cfg.MCP.PublicURL}}.Encode())
	w = call("/api/v1/mcp/oauth/token", "", b)
	require.Equal(t, 200, w.Code, w.Body.String())
	var tokens service.MCPOAuthTokenResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tokens))
	require.NotEmpty(t, tokens.RefreshToken)
	_, err = oauth.Authenticate(ctx, tokens.AccessToken)
	require.NoError(t, err)
	var fresh *service.MCPOAuthGrant
	for _, g := range repo.grants {
		if g.RevokedAt == nil {
			fresh = g
		}
	}
	require.NotNil(t, fresh)
	require.Greater(t, time.Until(fresh.ExpiresAt), 29*24*time.Hour)
	require.Equal(t, 200, call("/api/v1/auth/revoke-all-sessions", freshPair.AccessToken, nil).Code)
	require.Equal(t, 401, call("/api/v1/user/mcp/authorize", freshPair.AccessToken, b).Code)
	_, err = oauth.Authenticate(ctx, tokens.AccessToken)
	require.ErrorIs(t, err, service.ErrMCPOAuthInvalidToken)
	require.Equal(t, int64(2), users.user.SessionVersion)
	t.Log("repeated revoke blocks old JWT, browser refresh and MCP; fresh password login works; stale refresh snapshot cannot restore access")
}
