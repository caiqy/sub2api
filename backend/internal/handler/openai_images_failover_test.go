//go:build unit

package handler

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type openAIImagesFailoverAccountRepo struct {
	service.AccountRepository
	accounts []service.Account
}

func (r openAIImagesFailoverAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			account := r.accounts[i]
			return &account, nil
		}
	}
	return nil, service.ErrNoAvailableAccounts
}

func (r openAIImagesFailoverAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r openAIImagesFailoverAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r openAIImagesFailoverAccountRepo) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r openAIImagesFailoverAccountRepo) accountsForPlatform(platform string) []service.Account {
	out := make([]service.Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account.Platform == platform {
			out = append(out, account)
		}
	}
	return out
}

type openAIImagesFailoverHTTPUpstream struct {
	service.HTTPUpstream
	mu         sync.Mutex
	accountIDs []int64
	sessionIDs []string
	resp       *http.Response
}

type openAIImagesSchedulerCache struct {
	service.GatewayCache
	mu       sync.Mutex
	sessions []string
}

func (c *openAIImagesSchedulerCache) GetSessionAccountID(_ context.Context, _ int64, session string) (int64, error) {
	c.mu.Lock()
	c.sessions = append(c.sessions, session)
	c.mu.Unlock()
	return 0, nil
}

func (c *openAIImagesSchedulerCache) SetSessionAccountID(context.Context, int64, string, int64, time.Duration) error {
	return nil
}

func (c *openAIImagesSchedulerCache) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}

func (c *openAIImagesSchedulerCache) DeleteSessionAccountID(context.Context, int64, string) error {
	return nil
}

func (c *openAIImagesSchedulerCache) schedulerSessions() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.sessions...)
}

func (u *openAIImagesFailoverHTTPUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.mu.Lock()
	u.accountIDs = append(u.accountIDs, accountID)
	u.sessionIDs = append(u.sessionIDs, req.Header.Get("session_id"))
	u.mu.Unlock()
	if u.resp != nil {
		return u.resp, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"X-Request-Id": []string{"req_img_failover"},
		},
		Body: io.NopCloser(bytes.NewBufferString(
			"data: {\"type\":\"error\",\"error\":{\"type\":\"server_error\",\"code\":\"server_error\",\"message\":\"image backend unavailable\"}}\n\n",
		)),
	}, nil
}

func (u *openAIImagesFailoverHTTPUpstream) calls() []int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]int64(nil), u.accountIDs...)
}

func (u *openAIImagesFailoverHTTPUpstream) sessions() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]string(nil), u.sessionIDs...)
}

func TestOpenAIGatewayHandlerImages_ServerErrorFailsOverAndReturnsClearErrorWhenExhausted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawDir := t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: rawDir, FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
	groupID := int64(3130)
	accounts := []service.Account{
		{
			ID:          1,
			Name:        "image-account-1",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Status:      service.StatusActive,
			Schedulable: true,
			Concurrency: 0,
			Priority:    0,
			Credentials: map[string]any{"access_token": "token-1"},
		},
		{
			ID:          2,
			Name:        "image-account-2",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Status:      service.StatusActive,
			Schedulable: true,
			Concurrency: 0,
			Priority:    1,
			Credentials: map[string]any{"access_token": "token-2"},
		},
	}
	accountRepo := openAIImagesFailoverAccountRepo{accounts: accounts}
	upstream := &openAIImagesFailoverHTTPUpstream{}
	cache := &openAIImagesSchedulerCache{}
	cfg := &config.Config{RunMode: config.RunModeSimple, Gateway: config.GatewayConfig{
		OpenAIWS: config.GatewayOpenAIWSConfig{SchedulerMode: "weighted"},
		Sticky:   config.GatewayStickyConfig{OpenAI: config.GatewayStickyPlatformConfig{Enabled: true}},
	}}
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		cache,
		cfg,
		nil,
		nil,
		nil,
		nil,
		nil,
		upstream,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	billingService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingService.Stop)
	concurrencyService := service.NewConcurrencyService(nil)
	handler := NewOpenAIGatewayHandler(
		gatewayService,
		concurrencyService,
		billingService,
		service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg),
		nil,
		nil,
		nil,
		nil,
		cfg,
	)
	handler.maxAccountSwitches = 10

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	require.NoError(t, form.WriteField("model", "gpt-image-2"))
	require.NoError(t, form.WriteField("prompt", "draw a cat"))
	require.NoError(t, form.Close())
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		ID:      99,
		GroupID: &groupID,
		Group: &service.Group{
			ID:                   groupID,
			AllowImageGeneration: true,
		},
		User: &service.User{ID: 100},
	})
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 100, Concurrency: 0})

	handler.Images(c)

	require.ElementsMatch(t, []int64{1, 2}, upstream.calls())
	sessions := upstream.sessions()
	require.Len(t, sessions, 2)
	require.Equal(t, sessions[0], sessions[1])
	require.NotEmpty(t, sessions[0])
	schedulerSessions := cache.schedulerSessions()
	require.NotEmpty(t, schedulerSessions)
	for _, session := range schedulerSessions {
		require.NotEmpty(t, session)
		require.Equal(t, schedulerSessions[0], session)
	}
	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Equal(t, "upstream_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	require.Equal(t, "Upstream service temporarily unavailable", gjson.GetBytes(rec.Body.Bytes(), "error.message").String())

	rawEvents, ok := c.Get(service.OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*service.OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 2)
	require.Equal(t, "failover", events[0].Kind)
	require.Equal(t, "failover", events[1].Kind)
	require.Empty(t, readTestDir(t, rawDir), "OAuth retries must clean the effective body handle")

	var otherBody bytes.Buffer
	otherForm := multipart.NewWriter(&otherBody)
	require.NoError(t, otherForm.WriteField("model", "gpt-image-2"))
	require.NoError(t, otherForm.WriteField("prompt", "draw a dog"))
	require.NoError(t, otherForm.Close())
	otherReq := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(otherBody.Bytes()))
	otherReq.Header.Set("Content-Type", otherForm.FormDataContentType())
	otherRec := httptest.NewRecorder()
	otherCtx, _ := gin.CreateTestContext(otherRec)
	otherCtx.Request = otherReq
	otherCtx.Set(string(middleware2.ContextKeyAPIKey), c.MustGet(string(middleware2.ContextKeyAPIKey)))
	otherCtx.Set(string(middleware2.ContextKeyUser), c.MustGet(string(middleware2.ContextKeyUser)))
	handler.Images(otherCtx)

	otherSchedulerSessions := cache.schedulerSessions()[len(schedulerSessions):]
	require.NotEmpty(t, otherSchedulerSessions)
	for _, session := range otherSchedulerSessions {
		require.NotEmpty(t, session)
		require.Equal(t, otherSchedulerSessions[0], session)
	}
	require.NotEqual(t, schedulerSessions[0], otherSchedulerSessions[0])
}
