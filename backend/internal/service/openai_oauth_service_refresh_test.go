package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

type openaiOAuthClientRefreshStub struct {
	refreshCalls int32
	tokenResp    *openai.TokenResponse
	err          error
}

type openAITokenProviderRefreshRepoStub struct {
	AccountRepository
	account *Account
}

func (r *openAITokenProviderRefreshRepoStub) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

func (r *openAITokenProviderRefreshRepoStub) Update(context.Context, *Account) error {
	return nil
}

func (r *openAITokenProviderRefreshRepoStub) UpdateCredentials(_ context.Context, _ int64, credentials map[string]any) error {
	if r.account != nil {
		r.account.Credentials = cloneCredentials(credentials)
	}
	return nil
}

type openAITokenProviderRefreshCacheStub struct{}

func (c *openAITokenProviderRefreshCacheStub) GetAccessToken(context.Context, string) (string, error) {
	return "", nil
}

func (c *openAITokenProviderRefreshCacheStub) SetAccessToken(context.Context, string, string, time.Duration) error {
	return nil
}

func (c *openAITokenProviderRefreshCacheStub) DeleteAccessToken(context.Context, string) error {
	return nil
}

func (c *openAITokenProviderRefreshCacheStub) AcquireRefreshLock(context.Context, string, time.Duration) (bool, error) {
	return true, nil
}

func (c *openAITokenProviderRefreshCacheStub) ReleaseRefreshLock(context.Context, string) error {
	return nil
}

func (s *openaiOAuthClientRefreshStub) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL, clientID string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *openaiOAuthClientRefreshStub) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*openai.TokenResponse, error) {
	atomic.AddInt32(&s.refreshCalls, 1)
	if s.err != nil {
		return nil, s.err
	}
	if s.tokenResp != nil {
		return s.tokenResp, nil
	}
	return nil, errors.New("not implemented")
}

func (s *openaiOAuthClientRefreshStub) RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL string, clientID string) (*openai.TokenResponse, error) {
	atomic.AddInt32(&s.refreshCalls, 1)
	if s.err != nil {
		return nil, s.err
	}
	if s.tokenResp != nil {
		return s.tokenResp, nil
	}
	return nil, errors.New("not implemented")
}

func TestOpenAIOAuthService_RefreshAccountToken_NoRefreshTokenUsesExistingAccessToken(t *testing.T) {
	client := &openaiOAuthClientRefreshStub{}
	svc := NewOpenAIOAuthService(nil, client)
	var privacyClientCalls int32
	svc.SetPrivacyClientFactory(func(proxyURL string) (*req.Client, error) {
		atomic.AddInt32(&privacyClientCalls, 1)
		return nil, errors.New("stop before request")
	})

	expiresAt := time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)
	account := &Account{
		ID:       77,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "existing-access-token",
			"expires_at":   expiresAt,
			"client_id":    "client-id-1",
		},
	}

	info, err := svc.RefreshAccountToken(context.Background(), account)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, "existing-access-token", info.AccessToken)
	require.Equal(t, "client-id-1", info.ClientID)
	require.Zero(t, atomic.LoadInt32(&client.refreshCalls), "existing access token should be reused without calling refresh")
	require.Positive(t, atomic.LoadInt32(&privacyClientCalls), "existing access token should still run enrichment")
}

func TestOpenAITokenRefresher_NeedsRefresh_SkipsAccountWithoutRefreshToken(t *testing.T) {
	refresher := NewOpenAITokenRefresher(nil, nil)
	expiresAt := time.Now().Add(time.Minute).UTC().Format(time.RFC3339)

	withoutRT := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "access-token",
			"expires_at":   expiresAt,
		},
	}
	require.False(t, refresher.NeedsRefresh(withoutRT, 5*time.Minute))

	withRT := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"expires_at":    expiresAt,
		},
	}
	require.True(t, refresher.NeedsRefresh(withRT, 5*time.Minute))
}

func TestOpenAITokenProvider_NoRefreshTokenExpiredAccessTokenReturnsError(t *testing.T) {
	provider := NewOpenAITokenProvider(nil, nil, nil)
	expiresAt := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "expired-access-token",
			"expires_at":   expiresAt,
		},
	}

	token, err := provider.GetAccessToken(context.Background(), account)
	require.Error(t, err)
	require.Empty(t, token)
	require.Contains(t, err.Error(), "refresh_token is missing")
}

func TestOpenAITokenProvider_RefreshesWhenAccessTokenMissingWithFutureExpiry(t *testing.T) {
	client := &openaiOAuthClientRefreshStub{tokenResp: &openai.TokenResponse{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresIn:    3600,
	}}
	repo := &openAITokenProviderRefreshRepoStub{}
	cache := &openAITokenProviderRefreshCacheStub{}
	provider := NewOpenAITokenProvider(repo, cache, NewOpenAIOAuthService(nil, client))
	refreshAPI := NewOAuthRefreshAPI(repo, cache)
	provider.SetRefreshAPI(refreshAPI, NewOpenAITokenRefresher(NewOpenAIOAuthService(nil, client), repo))

	expiresAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	account := &Account{
		ID:       78,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"refresh_token": "old-refresh-token",
			"expires_at":    expiresAt,
		},
	}
	repo.account = account

	token, err := provider.GetAccessToken(context.Background(), account)

	require.NoError(t, err)
	require.Equal(t, "new-access-token", token)
	require.Equal(t, int32(1), atomic.LoadInt32(&client.refreshCalls))
}
