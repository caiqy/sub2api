package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// The authenticated request must not adopt a newer revocation generation.
// The synchronous hook chooses an exact interleaving without sleeps or data races.
type mcpAuthorizationGate struct {
	service.MCPOAuthRepository
	beforeSave func()
}

func (r *mcpAuthorizationGate) CreateAuthorization(ctx context.Context, g *service.MCPOAuthGrant, c *service.MCPOAuthCode, limit int) error {
	r.beforeSave()
	return r.MCPOAuthRepository.CreateAuthorization(ctx, g, c, limit)
}

func TestHTTPMCPRejectsJWTRevokedAfterMiddleware(t *testing.T) {
	for _, stage := range []string{"middleware", "save"} {
		t.Run(stage, func(t *testing.T) { testHTTPMCPRevocationGate(t, stage) })
	}
}
func testHTTPMCPRevocationGate(t *testing.T, stage string) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	oauth, repo, users := newMCPRouteOAuthService(service.RoleUser)
	now := time.Now()
	users.user.LastActiveAt = &now
	cfg := &config.Config{}
	cfg.JWT.Secret = "independent-toctou-review-secret-0000000"
	cfg.JWT.ExpireHour = 1
	cfg.MCP.PublicURL = "https://review.example/api/v1/mcp"
	settings := service.NewSettingService(&mcpRouteSettingsRepo{values: map[string]string{}}, cfg)
	auth := service.NewAuthService(nil, users, nil, nil, cfg, settings, nil, nil, nil, nil, nil, nil, nil)
	auth.SetMCPGrantRevoker(oauth)
	us := service.NewUserService(users, nil, nil, nil)
	ah := handler.NewAuthHandler(cfg, auth, us, settings, nil, nil, nil, nil)
	gate := &mcpAuthorizationGate{MCPOAuthRepository: repo}
	if stage == "save" {
		oauth = service.NewMCPOAuthService(gate, users)
		auth.SetMCPGrantRevoker(oauth)
	}
	mh := handler.NewMCPOAuthHandler(oauth, nil, us, settings, cfg)
	token, err := auth.GenerateToken(ctx, users.user)
	require.NoError(t, err)
	client, err := oauth.Register(ctx, service.MCPOAuthRegisterInput{RedirectURIs: []string{"http://127.0.0.1:37777/callback"}})
	require.NoError(t, err)
	a := mcpOAuthRouteAuthorization(client.ClientID, cfg.MCP.PublicURL)
	body, err := json.Marshal(service.MCPOAuthConsentInput{Authorization: a, Approved: true, Scope: "user"})
	require.NoError(t, err)
	r := gin.New()
	call := func(path, bearer string, body []byte) *httptest.ResponseRecorder {
		q := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		q.Header.Set("Content-Type", "application/json")
		if path == "/token" {
			q.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if bearer != "" {
			q.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, q)
		return w
	}
	secured := r.Group("", gin.HandlerFunc(middleware.NewJWTAuthMiddleware(auth, us, settings, nil)))
	secured.POST("/revoke", ah.RevokeAllSessions)
	revoke := func() {
		w := call("/revoke", token, nil)
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Equal(t, int64(1), users.user.SessionVersion)
	}
	gate.beforeSave = revoke
	secured.POST("/authorize", func(c *gin.Context) {
		if stage == "middleware" {
			revoke()
		}
		c.Next()
	}, mh.Decide)
	r.POST("/token", mh.Token)
	w := call("/authorize", token, body)
	if stage == "middleware" {
		require.Equal(t, 401, w.Code, w.Body.String())
		require.Empty(t, repo.grants)
		return
	}
	// Revocation after service validation may save a code, but it cannot upgrade the principal.
	require.Equal(t, 200, w.Code, w.Body.String())
	var result struct {
		Data struct {
			Redirect string `json:"redirect_url"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	redirect, err := url.Parse(result.Data.Redirect)
	require.NoError(t, err)
	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ClientID}, "code": {redirect.Query().Get("code")}, "redirect_uri": {a.RedirectURI}, "code_verifier": {mcpOAuthRouteVerifier}, "resource": {cfg.MCP.PublicURL}}
	w = call("/token", "", []byte(form.Encode()))
	require.Equal(t, 400, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "invalid_grant")
	require.Empty(t, repo.credentials)
}
