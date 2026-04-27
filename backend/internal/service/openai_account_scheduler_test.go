package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type openAIWSStateStoreSpy struct {
	responseAccounts        map[string]int64
	getResponseAccountCalls map[string]int
	bindResponseCalls       map[string]int
	deleteResponseCalls     map[string]int
}

func (s *openAIWSStateStoreSpy) BindResponseAccount(ctx context.Context, groupID int64, responseID string, accountID int64, ttl time.Duration) error {
	if s.bindResponseCalls == nil {
		s.bindResponseCalls = make(map[string]int)
	}
	s.bindResponseCalls[responseID]++
	if s.responseAccounts == nil {
		s.responseAccounts = make(map[string]int64)
	}
	s.responseAccounts[responseID] = accountID
	return nil
}

func (s *openAIWSStateStoreSpy) GetResponseAccount(ctx context.Context, groupID int64, responseID string) (int64, error) {
	if s.getResponseAccountCalls == nil {
		s.getResponseAccountCalls = make(map[string]int)
	}
	s.getResponseAccountCalls[responseID]++
	if s.responseAccounts == nil {
		return 0, nil
	}
	return s.responseAccounts[responseID], nil
}

func (s *openAIWSStateStoreSpy) DeleteResponseAccount(ctx context.Context, groupID int64, responseID string) error {
	if s.deleteResponseCalls == nil {
		s.deleteResponseCalls = make(map[string]int)
	}
	s.deleteResponseCalls[responseID]++
	if s.responseAccounts != nil {
		delete(s.responseAccounts, responseID)
	}
	return nil
}

func (s *openAIWSStateStoreSpy) BindResponseConn(responseID, connID string, ttl time.Duration) {}
func (s *openAIWSStateStoreSpy) GetResponseConn(responseID string) (string, bool)              { return "", false }
func (s *openAIWSStateStoreSpy) DeleteResponseConn(responseID string)                          {}
func (s *openAIWSStateStoreSpy) BindSessionTurnState(groupID int64, sessionHash, turnState string, ttl time.Duration) {
}
func (s *openAIWSStateStoreSpy) GetSessionTurnState(groupID int64, sessionHash string) (string, bool) {
	return "", false
}
func (s *openAIWSStateStoreSpy) DeleteSessionTurnState(groupID int64, sessionHash string) {}
func (s *openAIWSStateStoreSpy) BindSessionConn(groupID int64, sessionHash, connID string, ttl time.Duration) {
}
func (s *openAIWSStateStoreSpy) GetSessionConn(groupID int64, sessionHash string) (string, bool) {
	return "", false
}
func (s *openAIWSStateStoreSpy) DeleteSessionConn(groupID int64, sessionHash string) {}

func newOpenAIStickyEnabledTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = true
	return cfg
}

type openAISnapshotCacheStub struct {
	SchedulerCache
	snapshotAccounts []*Account
	accountsByID     map[int64]*Account
}

type schedulerTestOpenAIAccountRepo struct {
	AccountRepository
	accounts  []Account
	setErrors map[int64]string
}

func (r schedulerTestOpenAIAccountRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			return &r.accounts[i], nil
		}
	}
	return nil, errors.New("account not found")
}

func (r schedulerTestOpenAIAccountRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r schedulerTestOpenAIAccountRepo) ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r schedulerTestOpenAIAccountRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return r.ListSchedulableByPlatform(ctx, platform)
}

func (r schedulerTestOpenAIAccountRepo) SetError(ctx context.Context, id int64, errorMsg string) error {
	if r.setErrors != nil {
		r.setErrors[id] = errorMsg
	}
	return nil
}

type schedulerTestGroupRepo struct {
	GroupRepository
	groups map[int64]*Group
}

func (r schedulerTestGroupRepo) GetByID(ctx context.Context, id int64) (*Group, error) {
	if r.groups == nil {
		return nil, ErrGroupNotFound
	}
	group, ok := r.groups[id]
	if !ok || group == nil {
		return nil, ErrGroupNotFound
	}
	cloned := *group
	return &cloned, nil
}

type schedulerChannelRepoStub struct {
	channels       []Channel
	groupPlatforms map[int64]string
}

func (r *schedulerChannelRepoStub) Create(ctx context.Context, channel *Channel) error { return nil }
func (r *schedulerChannelRepoStub) GetByID(ctx context.Context, id int64) (*Channel, error) {
	return nil, ErrChannelNotFound
}
func (r *schedulerChannelRepoStub) Update(ctx context.Context, channel *Channel) error { return nil }
func (r *schedulerChannelRepoStub) Delete(ctx context.Context, id int64) error         { return nil }
func (r *schedulerChannelRepoStub) List(ctx context.Context, params pagination.PaginationParams, status, search string) ([]Channel, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *schedulerChannelRepoStub) ListAll(ctx context.Context) ([]Channel, error) {
	return append([]Channel(nil), r.channels...), nil
}
func (r *schedulerChannelRepoStub) ExistsByName(ctx context.Context, name string) (bool, error) {
	return false, nil
}
func (r *schedulerChannelRepoStub) ExistsByNameExcluding(ctx context.Context, name string, excludeID int64) (bool, error) {
	return false, nil
}
func (r *schedulerChannelRepoStub) GetGroupIDs(ctx context.Context, channelID int64) ([]int64, error) {
	return nil, nil
}
func (r *schedulerChannelRepoStub) SetGroupIDs(ctx context.Context, channelID int64, groupIDs []int64) error {
	return nil
}
func (r *schedulerChannelRepoStub) GetChannelIDByGroupID(ctx context.Context, groupID int64) (int64, error) {
	for _, ch := range r.channels {
		for _, gid := range ch.GroupIDs {
			if gid == groupID {
				return ch.ID, nil
			}
		}
	}
	return 0, ErrChannelNotFound
}
func (r *schedulerChannelRepoStub) GetGroupsInOtherChannels(ctx context.Context, channelID int64, groupIDs []int64) ([]int64, error) {
	return nil, nil
}
func (r *schedulerChannelRepoStub) GetGroupPlatforms(ctx context.Context, groupIDs []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(groupIDs))
	for _, gid := range groupIDs {
		if platform, ok := r.groupPlatforms[gid]; ok {
			out[gid] = platform
		}
	}
	return out, nil
}
func (r *schedulerChannelRepoStub) ListModelPricing(ctx context.Context, channelID int64) ([]ChannelModelPricing, error) {
	for _, ch := range r.channels {
		if ch.ID == channelID {
			return append([]ChannelModelPricing(nil), ch.ModelPricing...), nil
		}
	}
	return nil, nil
}
func (r *schedulerChannelRepoStub) CreateModelPricing(ctx context.Context, pricing *ChannelModelPricing) error {
	return nil
}
func (r *schedulerChannelRepoStub) UpdateModelPricing(ctx context.Context, pricing *ChannelModelPricing) error {
	return nil
}
func (r *schedulerChannelRepoStub) DeleteModelPricing(ctx context.Context, id int64) error {
	return nil
}
func (r *schedulerChannelRepoStub) ReplaceModelPricing(ctx context.Context, channelID int64, pricingList []ChannelModelPricing) error {
	return nil
}

func newSchedulerTestChannelService(ch Channel, groupPlatforms map[int64]string) *ChannelService {
	return NewChannelService(&schedulerChannelRepoStub{
		channels:       []Channel{ch},
		groupPlatforms: groupPlatforms,
	}, nil, nil, nil)
}

type schedulerTestConcurrencyCache struct {
	ConcurrencyCache
	loadBatchErr    error
	loadMap         map[int64]*AccountLoadInfo
	acquireResults  map[int64]bool
	waitCounts      map[int64]int
	skipDefaultLoad bool
}

func (c schedulerTestConcurrencyCache) AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error) {
	if c.acquireResults != nil {
		if result, ok := c.acquireResults[accountID]; ok {
			return result, nil
		}
	}
	return true, nil
}

func (c schedulerTestConcurrencyCache) ReleaseAccountSlot(ctx context.Context, accountID int64, requestID string) error {
	return nil
}

func (c schedulerTestConcurrencyCache) GetAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	if c.loadBatchErr != nil {
		return nil, c.loadBatchErr
	}
	out := make(map[int64]*AccountLoadInfo, len(accounts))
	if c.skipDefaultLoad && c.loadMap != nil {
		for _, acc := range accounts {
			if load, ok := c.loadMap[acc.ID]; ok {
				out[acc.ID] = load
			}
		}
		return out, nil
	}
	for _, acc := range accounts {
		if c.loadMap != nil {
			if load, ok := c.loadMap[acc.ID]; ok {
				out[acc.ID] = load
				continue
			}
		}
		out[acc.ID] = &AccountLoadInfo{AccountID: acc.ID, LoadRate: 0}
	}
	return out, nil
}

func (c schedulerTestConcurrencyCache) GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error) {
	if c.waitCounts != nil {
		if count, ok := c.waitCounts[accountID]; ok {
			return count, nil
		}
	}
	return 0, nil
}

type schedulerTestGatewayCache struct {
	sessionBindings map[string]int64
	deletedSessions map[string]int
}

func (c *schedulerTestGatewayCache) GetSessionAccountID(ctx context.Context, groupID int64, sessionHash string) (int64, error) {
	if id, ok := c.sessionBindings[sessionHash]; ok {
		return id, nil
	}
	return 0, errors.New("not found")
}

func (c *schedulerTestGatewayCache) SetSessionAccountID(ctx context.Context, groupID int64, sessionHash string, accountID int64, ttl time.Duration) error {
	if c.sessionBindings == nil {
		c.sessionBindings = make(map[string]int64)
	}
	c.sessionBindings[sessionHash] = accountID
	return nil
}

func (c *schedulerTestGatewayCache) RefreshSessionTTL(ctx context.Context, groupID int64, sessionHash string, ttl time.Duration) error {
	return nil
}

func (c *schedulerTestGatewayCache) DeleteSessionAccountID(ctx context.Context, groupID int64, sessionHash string) error {
	if c.sessionBindings == nil {
		return nil
	}
	if c.deletedSessions == nil {
		c.deletedSessions = make(map[string]int)
	}
	c.deletedSessions[sessionHash]++
	delete(c.sessionBindings, sessionHash)
	return nil
}

func newSchedulerTestOpenAIWSV2Config() *config.Config {
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600
	return cfg
}

type openAIAdvancedSchedulerSettingRepoStub struct {
	values map[string]string
}

func (s *openAIAdvancedSchedulerSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	value, err := s.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *openAIAdvancedSchedulerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s == nil || s.values == nil {
		return "", ErrSettingNotFound
	}
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *openAIAdvancedSchedulerSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected call to Set")
}

func (s *openAIAdvancedSchedulerSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected call to GetMultiple")
}

func (s *openAIAdvancedSchedulerSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected call to SetMultiple")
}

func (s *openAIAdvancedSchedulerSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected call to GetAll")
}

func (s *openAIAdvancedSchedulerSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected call to Delete")
}

func newOpenAIAdvancedSchedulerRateLimitService(enabled string) *RateLimitService {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	repo := &openAIAdvancedSchedulerSettingRepoStub{
		values: map[string]string{},
	}
	if enabled != "" {
		repo.values[openAIAdvancedSchedulerSettingKey] = enabled
	}
	return &RateLimitService{
		settingService: NewSettingService(repo, &config.Config{}),
	}
}

func (s *openAISnapshotCacheStub) GetSnapshot(ctx context.Context, bucket SchedulerBucket) ([]*Account, bool, error) {
	if len(s.snapshotAccounts) == 0 {
		return nil, false, nil
	}
	out := make([]*Account, 0, len(s.snapshotAccounts))
	for _, account := range s.snapshotAccounts {
		if account == nil {
			continue
		}
		cloned := *account
		out = append(out, &cloned)
	}
	return out, true, nil
}

func (s *openAISnapshotCacheStub) GetAccount(ctx context.Context, accountID int64) (*Account, error) {
	if s.accountsByID == nil {
		return nil, nil
	}
	account := s.accountsByID[accountID]
	if account == nil {
		return nil, nil
	}
	cloned := *account
	return &cloned, nil
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_DefaultDisabledUsesLegacyLoadAwareness(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10106)
	accounts := []Account{
		{
			ID:          36001,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
		},
		{
			ID:          36002,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	cache := &schedulerTestGatewayCache{}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	store := svc.getOpenAIWSStateStore()
	require.NoError(t, store.BindResponseAccount(ctx, groupID, "resp_disabled_001", 36001, time.Hour))
	require.False(t, svc.isOpenAIAdvancedSchedulerEnabled(ctx))

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"resp_disabled_001",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(36002), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickyPreviousHit)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_DefaultDisabled_RequiredWSV2_SkipsHTTPOnlyAccount(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10108)
	accounts := []Account{
		{
			ID:          36011,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          36012,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
			},
		},
	}
	cfg := newSchedulerTestOpenAIWSV2Config()
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportResponsesWebsocketV2,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(36012), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_DefaultDisabled_RequiredWSV2_NoAvailableAccount(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10109)
	accounts := []Account{
		{
			ID:          36021,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
	}
	cfg := newSchedulerTestOpenAIWSV2Config()
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportResponsesWebsocketV2,
		false,
	)
	require.ErrorContains(t, err, "no available OpenAI accounts")
	require.Nil(t, selection)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_EnabledUsesAdvancedPreviousResponseRouting(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10107)
	accounts := []Account{
		{
			ID:          37001,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
			},
		},
		{
			ID:          37002,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
	}
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	store := svc.getOpenAIWSStateStore()
	require.NoError(t, store.BindResponseAccount(ctx, groupID, "resp_enabled_001", 37001, time.Hour))
	require.True(t, svc.isOpenAIAdvancedSchedulerEnabled(ctx))

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"resp_enabled_001",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(37001), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerPreviousResponse, decision.Layer)
	require.True(t, decision.StickyPreviousHit)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_AdvancedHonorsChannelPricingRestriction(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(41001)
	accounts := []Account{{
		ID:          410011,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
	}}
	channelSvc := newSchedulerTestChannelService(Channel{
		ID:                 1,
		Status:             StatusActive,
		GroupIDs:           []int64{groupID},
		RestrictModels:     true,
		BillingModelSource: BillingModelSourceRequested,
		ModelPricing: []ChannelModelPricing{{
			Platform: "openai",
			Models:   []string{"gpt-4.1"},
		}},
	}, map[int64]string{groupID: "openai"})
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.SchedulerMode = "weighted"

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		channelService:     channelSvc,
	}

	selection, _, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.ErrorContains(t, err, "channel pricing restriction")
	require.Nil(t, selection)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_AdvancedUpstreamRestrictionClearsPreviousResponseSticky(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(41003)
	stickyAccount := Account{
		ID:          410031,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    9,
		Extra:       map[string]any{"openai_apikey_responses_websockets_v2_enabled": true},
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.1": "restricted-upstream"}},
	}
	backupAccount := Account{
		ID:          410032,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra:       map[string]any{"openai_apikey_responses_websockets_v2_enabled": true},
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.1": "allowed-upstream"}},
	}
	channelSvc := newSchedulerTestChannelService(Channel{
		ID:                 2,
		Status:             StatusActive,
		GroupIDs:           []int64{groupID},
		RestrictModels:     true,
		BillingModelSource: BillingModelSourceUpstream,
		ModelPricing: []ChannelModelPricing{{
			Platform: "openai",
			Models:   []string{"allowed-upstream"},
		}},
	}, map[int64]string{groupID: "openai"})
	stateStore := &openAIWSStateStoreSpy{responseAccounts: map[string]int64{"resp_prev_restricted": stickyAccount.ID}}
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600
	cfg.Gateway.OpenAIWS.SchedulerMode = "weighted"

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{stickyAccount, backupAccount}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		channelService:     channelSvc,
		openaiWSStateStore: stateStore,
	}

	selection, _, err := svc.SelectAccountWithScheduler(ctx, &groupID, "resp_prev_restricted", "session_hash_prev_restricted", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, backupAccount.ID, selection.Account.ID)
	_, exists := stateStore.responseAccounts["resp_prev_restricted"]
	require.False(t, exists, "restricted previous_response sticky binding should be cleared")
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_StickyDisabledDoesNotDeletePreviousResponseOnRestriction(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(41004)
	restrictedAccount := Account{
		ID:          410041,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra:       map[string]any{"openai_apikey_responses_websockets_v2_enabled": true},
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.1": "restricted-upstream"}},
	}
	channelSvc := newSchedulerTestChannelService(Channel{
		ID:                 4,
		Status:             StatusActive,
		GroupIDs:           []int64{groupID},
		RestrictModels:     true,
		BillingModelSource: BillingModelSourceUpstream,
		ModelPricing: []ChannelModelPricing{{
			Platform: "openai",
			Models:   []string{"allowed-upstream"},
		}},
	}, map[int64]string{groupID: "openai"})
	stateStore := &openAIWSStateStoreSpy{responseAccounts: map[string]int64{"resp_prev_disabled_restricted": restrictedAccount.ID}}
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.Sticky.OpenAI.Enabled = false
	cfg.Gateway.Sticky.Gemini.Enabled = true
	cfg.Gateway.Sticky.Anthropic.Enabled = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600
	cfg.Gateway.OpenAIWS.SchedulerMode = "weighted"

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{restrictedAccount}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		channelService:     channelSvc,
		openaiWSStateStore: stateStore,
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "resp_prev_disabled_restricted", "session_hash_prev_disabled_restricted", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.Error(t, err)
	require.Nil(t, selection)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Zero(t, stateStore.getResponseAccountCalls["resp_prev_disabled_restricted"])
	require.Zero(t, stateStore.deleteResponseCalls["resp_prev_disabled_restricted"])
	require.Equal(t, restrictedAccount.ID, stateStore.responseAccounts["resp_prev_disabled_restricted"])
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_LayeredRequirePrivacySet(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(41002)
	repo := schedulerTestOpenAIAccountRepo{
		accounts: []Account{
			{
				ID:          410021,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    0,
				Extra:       map[string]any{},
			},
			{
				ID:          410022,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    5,
				Extra: map[string]any{
					"privacy_mode": PrivacyModeTrainingOff,
				},
			},
		},
		setErrors: map[int64]string{},
	}
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	snapshotCfg := &config.Config{}
	snapshotCfg.Gateway.Scheduling.DbFallbackEnabled = true

	svc := &OpenAIGatewayService{
		accountRepo:        repo,
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		schedulerSnapshot: NewSchedulerSnapshotService(nil, nil, repo, schedulerTestGroupRepo{groups: map[int64]*Group{
			groupID: {
				ID:                groupID,
				Name:              "privacy-required",
				RequirePrivacySet: true,
			},
		}}, snapshotCfg),
	}

	selection, _, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(410022), selection.Account.ID)
	require.Contains(t, repo.setErrors[410021], "Privacy not set")
}

func TestOpenAIGatewayService_GetOpenAIAccountScheduler_RuntimeRefreshRecreatesAndIgnoresLegacyDisableWhenModeConfigured(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.SchedulerMode = "weighted"
	settingSvc := NewSettingService(&openAIAdvancedSchedulerSettingRepoStub{values: map[string]string{
		openAIAdvancedSchedulerSettingKey: "true",
	}}, cfg)
	rateLimitSvc := &RateLimitService{settingService: settingSvc}
	svc := &OpenAIGatewayService{cfg: cfg, rateLimitService: rateLimitSvc}

	scheduler := svc.getOpenAIAccountSchedulerWithContext(ctx)
	require.IsType(t, &defaultOpenAIAccountScheduler{}, scheduler)

	settingSvc.refreshCachedSettings(&SystemSettings{
		OpenAIAdvancedSchedulerEnabled:                       true,
		GatewayStickyOpenAIEnabled:                           true,
		GatewayStickyGeminiEnabled:                           true,
		GatewayStickyAnthropicEnabled:                        true,
		GatewayOpenAIWSSchedulerMode:                         "layered",
		GatewayOpenAIWSSchedulerLayeredErrorPenaltyThreshold: 0.2,
		GatewayOpenAIWSSchedulerLayeredErrorPenaltyValue:     1,
		GatewayOpenAIWSSchedulerLayeredTTFTPenaltyMultiplier: 2,
		GatewayOpenAIWSSchedulerLayeredTTFTPenaltyValue:      1,
		GatewayOpenAIWSSchedulerLayeredProbeCooldownSeconds:  1,
		GatewayOpenAIWSSchedulerLayeredProbeIntervalSeconds:  1,
		GatewayOpenAIWSSchedulerLayeredProbeMaxFailures:      1,
		GatewayOpenAIWSSchedulerLayeredProbeTimeoutSeconds:   1,
	})

	scheduler = svc.getOpenAIAccountSchedulerWithContext(ctx)
	require.IsType(t, &layeredOpenAIAccountScheduler{}, scheduler)

	settingSvc.refreshCachedSettings(&SystemSettings{
		OpenAIAdvancedSchedulerEnabled: false,
	})

	scheduler = svc.getOpenAIAccountSchedulerWithContext(ctx)
	require.IsType(t, &layeredOpenAIAccountScheduler{}, scheduler)
	require.NotNil(t, svc.openaiScheduler)
}

func TestOpenAIGatewayService_SchedulerModeLayeredIgnoresLegacyAdvancedSchedulerDisabledFlag(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold = 0.2
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue = 1
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier = 2
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue = 1
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 1
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 1
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures = 1
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds = 1
	settingSvc := NewSettingService(&openAIAdvancedSchedulerSettingRepoStub{values: map[string]string{
		openAIAdvancedSchedulerSettingKey: "false",
	}}, cfg)
	svc := &OpenAIGatewayService{cfg: cfg, rateLimitService: &RateLimitService{settingService: settingSvc}}

	scheduler := svc.getOpenAIAccountSchedulerWithContext(ctx)

	require.IsType(t, &layeredOpenAIAccountScheduler{}, scheduler)
}

func TestOpenAIGatewayService_SchedulerModeWeightedUsesDefaultSchedulerWithoutLegacyAdvancedFlag(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.SchedulerMode = "weighted"
	settingSvc := NewSettingService(&openAIAdvancedSchedulerSettingRepoStub{}, cfg)
	svc := &OpenAIGatewayService{cfg: cfg, rateLimitService: &RateLimitService{settingService: settingSvc}}

	scheduler := svc.getOpenAIAccountSchedulerWithContext(ctx)

	require.IsType(t, &defaultOpenAIAccountScheduler{}, scheduler)
}

func TestOpenAIGatewayService_SchedulerModeCacheDoesNotLeakEffectiveEnabledAcrossServices(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	repo := &openAIAdvancedSchedulerSettingRepoStub{values: map[string]string{
		openAIAdvancedSchedulerSettingKey: "false",
	}}
	modeCfg := newOpenAIStickyEnabledTestConfig()
	modeCfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	modeCfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold = 0.2
	modeCfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue = 1
	modeCfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier = 2
	modeCfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue = 1
	modeCfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 1
	modeCfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 1
	modeCfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures = 1
	modeCfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds = 1
	modeSettingSvc := NewSettingService(repo, modeCfg)
	modeSvc := &OpenAIGatewayService{cfg: modeCfg, rateLimitService: &RateLimitService{settingService: modeSettingSvc}}
	require.IsType(t, &layeredOpenAIAccountScheduler{}, modeSvc.getOpenAIAccountSchedulerWithContext(ctx))

	legacyCfg := newOpenAIStickyEnabledTestConfig()
	legacySettingSvc := NewSettingService(repo, legacyCfg)
	legacySvc := &OpenAIGatewayService{cfg: legacyCfg, rateLimitService: &RateLimitService{settingService: legacySettingSvc}}

	require.Nil(t, legacySvc.getOpenAIAccountSchedulerWithContext(ctx))
}

func TestOpenAIGatewayService_SettingsRefreshRecreatesLayeredSchedulerOnProbeIntervalChange(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold = 0.2
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue = 1
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier = 2
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue = 1
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 60
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 30
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures = 1
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds = 1
	settingSvc := NewSettingService(&openAIAdvancedSchedulerSettingRepoStub{values: map[string]string{
		openAIAdvancedSchedulerSettingKey: "true",
	}}, cfg)
	rateLimitSvc := &RateLimitService{settingService: settingSvc}
	svc := NewOpenAIGatewayService(nil, nil, nil, nil, nil, nil, nil, cfg, nil, nil, nil, rateLimitSvc, nil, nil, nil, nil, nil, nil, nil)
	first := svc.getOpenAIAccountSchedulerWithContext(ctx)
	firstLayered, ok := first.(*layeredOpenAIAccountScheduler)
	require.True(t, ok)
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	settingSvc.refreshCachedSettings(&SystemSettings{
		OpenAIAdvancedSchedulerEnabled:                       true,
		GatewayStickyOpenAIEnabled:                           true,
		GatewayStickyGeminiEnabled:                           true,
		GatewayStickyAnthropicEnabled:                        true,
		GatewayOpenAIWSSchedulerMode:                         "layered",
		GatewayOpenAIWSSchedulerLayeredErrorPenaltyThreshold: 0.2,
		GatewayOpenAIWSSchedulerLayeredErrorPenaltyValue:     1,
		GatewayOpenAIWSSchedulerLayeredTTFTPenaltyMultiplier: 2,
		GatewayOpenAIWSSchedulerLayeredTTFTPenaltyValue:      1,
		GatewayOpenAIWSSchedulerLayeredProbeCooldownSeconds:  60,
		GatewayOpenAIWSSchedulerLayeredProbeIntervalSeconds:  1,
		GatewayOpenAIWSSchedulerLayeredProbeMaxFailures:      1,
		GatewayOpenAIWSSchedulerLayeredProbeTimeoutSeconds:   1,
	})

	second, ok := svc.openaiScheduler.(*layeredOpenAIAccountScheduler)
	require.True(t, ok)
	require.NotSame(t, firstLayered, second)
	require.NotSame(t, firstLayered.probe, second.probe)
}

func TestOpenAIGatewayService_OpenAIAccountSchedulerMetrics_DisabledNoOp(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	svc := &OpenAIGatewayService{}
	ttft := 120
	svc.ReportOpenAIAccountScheduleResult(10, true, &ttft)
	svc.RecordOpenAIAccountSwitch()

	snapshot := svc.SnapshotOpenAIAccountSchedulerMetrics()
	require.Equal(t, OpenAIAccountSchedulerMetricsSnapshot{}, snapshot)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyRateLimitedAccountFallsBackToFreshCandidate(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10101)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	staleSticky := &Account{ID: 31001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	staleBackup := &Account{ID: 31002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	freshSticky := &Account{ID: 31001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, RateLimitResetAt: &rateLimitedUntil}
	freshBackup := &Account{ID: 31002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_rate_limited": 31001}}
	snapshotCache := &openAISnapshotCacheStub{snapshotAccounts: []*Account{staleSticky, staleBackup}, accountsByID: map[int64]*Account{31001: freshSticky, 31002: freshBackup}}
	snapshotService := &SchedulerSnapshotService{cache: snapshotCache}
	cfg := newOpenAIStickyEnabledTestConfig()
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{*freshSticky, *freshBackup}},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		schedulerSnapshot:  snapshotService,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_rate_limited", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(31002), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_SkipsFreshlyRateLimitedSnapshotCandidate(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10102)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	stalePrimary := &Account{ID: 32001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	staleSecondary := &Account{ID: 32002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	freshPrimary := &Account{ID: 32001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, RateLimitResetAt: &rateLimitedUntil}
	freshSecondary := &Account{ID: 32002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	snapshotCache := &openAISnapshotCacheStub{snapshotAccounts: []*Account{stalePrimary, staleSecondary}, accountsByID: map[int64]*Account{32001: freshPrimary, 32002: freshSecondary}}
	snapshotService := &SchedulerSnapshotService{cache: snapshotCache}
	cfg := newOpenAIStickyEnabledTestConfig()
	svc := &OpenAIGatewayService{
		accountRepo:       schedulerTestOpenAIAccountRepo{accounts: []Account{*freshPrimary, *freshSecondary}},
		cfg:               cfg,
		rateLimitService:  newOpenAIAdvancedSchedulerRateLimitService("true"),
		schedulerSnapshot: snapshotService,
	}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, &groupID, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(32002), account.ID)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDBRuntimeRecheckSkipsStaleCachedAccount(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10103)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	staleSticky := &Account{ID: 33001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	staleBackup := &Account{ID: 33002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	dbSticky := Account{ID: 33001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, RateLimitResetAt: &rateLimitedUntil}
	dbBackup := Account{ID: 33002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_db_runtime_recheck": 33001}}
	snapshotCache := &openAISnapshotCacheStub{
		snapshotAccounts: []*Account{staleSticky, staleBackup},
		accountsByID:     map[int64]*Account{33001: staleSticky, 33002: staleBackup},
	}
	snapshotService := &SchedulerSnapshotService{cache: snapshotCache}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{dbSticky, dbBackup}},
		cache:              cache,
		cfg:                newOpenAIStickyEnabledTestConfig(),
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		schedulerSnapshot:  snapshotService,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_db_runtime_recheck", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(33002), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_DBRuntimeRecheckSkipsStaleCachedCandidate(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10104)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	stalePrimary := &Account{ID: 34001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	staleSecondary := &Account{ID: 34002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	dbPrimary := Account{ID: 34001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, RateLimitResetAt: &rateLimitedUntil}
	dbSecondary := Account{ID: 34002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	snapshotCache := &openAISnapshotCacheStub{
		snapshotAccounts: []*Account{stalePrimary, staleSecondary},
		accountsByID:     map[int64]*Account{34001: stalePrimary, 34002: staleSecondary},
	}
	snapshotService := &SchedulerSnapshotService{cache: snapshotCache}
	svc := &OpenAIGatewayService{
		accountRepo:       schedulerTestOpenAIAccountRepo{accounts: []Account{dbPrimary, dbSecondary}},
		cfg:               newOpenAIStickyEnabledTestConfig(),
		rateLimitService:  newOpenAIAdvancedSchedulerRateLimitService("true"),
		schedulerSnapshot: snapshotService,
	}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, &groupID, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(34002), account.ID)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseSticky(t *testing.T) {
	ctx := context.Background()
	groupID := int64(9)
	account := Account{
		ID:          1001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 2,
		Extra: map[string]any{
			"openai_apikey_responses_websockets_v2_enabled": true,
		},
	}
	cache := &schedulerTestGatewayCache{}
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 1800
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	store := svc.getOpenAIWSStateStore()
	require.NoError(t, store.BindResponseAccount(ctx, groupID, "resp_prev_001", account.ID, time.Hour))

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"resp_prev_001",
		"session_hash_001",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerPreviousResponse, decision.Layer)
	require.True(t, decision.StickyPreviousHit)
	require.Equal(t, account.ID, cache.sessionBindings["openai:session_hash_001"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionSticky(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10)
	account := Account{
		ID:          2001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_abc": account.ID,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              cache,
		cfg:                newOpenAIStickyEnabledTestConfig(),
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"session_hash_abc",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseStickyDisabledBypassesStickyLookupAndBind(t *testing.T) {
	ctx := context.Background()
	groupID := int64(9101)
	stickyAccount := Account{ID: 91011, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 9, Extra: map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}}
	bestAccount := Account{ID: 91012, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, Extra: map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}}
	cache := &stubGatewayCache{}
	stateStore := &openAIWSStateStoreSpy{responseAccounts: map[string]int64{"resp_prev_disabled": stickyAccount.ID}}
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.Sticky.OpenAI.Enabled = false
	cfg.Gateway.Sticky.Gemini.Enabled = true
	cfg.Gateway.Sticky.Anthropic.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "weighted"

	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{stickyAccount, bestAccount}},
		cache:              cache,
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
		openaiWSStateStore: stateStore,
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "resp_prev_disabled", "session_hash_prev_disabled", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickyPreviousHit)
	require.Zero(t, stateStore.getResponseAccountCalls["resp_prev_disabled"])
	require.Zero(t, stateStore.bindResponseCalls["resp_prev_disabled"])
	require.Zero(t, cache.getCalls["openai:session_hash_prev_disabled"])
	require.Zero(t, cache.setCalls["openai:session_hash_prev_disabled"])
	require.Zero(t, cache.refreshCalls["openai:session_hash_prev_disabled"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDisabledBypassesLookupBindAndRefresh(t *testing.T) {
	ctx := context.Background()
	groupID := int64(9102)
	stickyAccount := Account{ID: 91021, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 9}
	bestAccount := Account{ID: 91022, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	cache := &stubGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_disabled": stickyAccount.ID}}
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.Sticky.OpenAI.Enabled = false
	cfg.Gateway.Sticky.Gemini.Enabled = true
	cfg.Gateway.Sticky.Anthropic.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "weighted"

	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{stickyAccount, bestAccount}},
		cache:              cache,
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_disabled", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	require.Zero(t, cache.getCalls["openai:session_hash_disabled"])
	require.Zero(t, cache.setCalls["openai:session_hash_disabled"])
	require.Zero(t, cache.refreshCalls["openai:session_hash_disabled"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_BindOpenAIResponseAccount_DisabledSkipsStateStoreBind(t *testing.T) {
	store := &openAIWSStateStoreSpy{}
	svc := &OpenAIGatewayService{
		cfg:                &config.Config{Gateway: config.GatewayConfig{Sticky: config.GatewayStickyConfig{}}},
		openaiWSStateStore: store,
	}

	err := svc.bindOpenAIResponseAccount(context.Background(), 1, "resp_bind_disabled", 7, time.Hour)
	require.NoError(t, err)
	require.Zero(t, store.bindResponseCalls["resp_bind_disabled"])
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyBusyKeepsSticky(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10100)
	accounts := []Account{
		{
			ID:          21001,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          21002,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    9,
		},
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_sticky_busy": 21001,
		},
	}
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.Scheduling.StickySessionMaxWaiting = 2
	cfg.Gateway.Scheduling.StickySessionWaitTimeout = 45 * time.Second
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true

	concurrencyCache := schedulerTestConcurrencyCache{
		acquireResults: map[int64]bool{
			21001: false, // sticky 账号已满
			21002: true,  // 若回退负载均衡会命中该账号（本测试要求不能切换）
		},
		waitCounts: map[int64]int{
			21001: 999,
		},
		loadMap: map[int64]*AccountLoadInfo{
			21001: {AccountID: 21001, LoadRate: 90, WaitingCount: 9},
			21002: {AccountID: 21002, LoadRate: 1, WaitingCount: 0},
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"session_hash_sticky_busy",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(21001), selection.Account.ID, "busy sticky account should remain selected")
	require.False(t, selection.Acquired)
	require.NotNil(t, selection.WaitPlan)
	require.Equal(t, int64(21001), selection.WaitPlan.AccountID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionSticky_ForceHTTP(t *testing.T) {
	ctx := context.Background()
	groupID := int64(1010)
	account := Account{
		ID:          2101,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Extra: map[string]any{
			"openai_ws_force_http": true,
		},
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_force_http": account.ID,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              cache,
		cfg:                newOpenAIStickyEnabledTestConfig(),
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"session_hash_force_http",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_RequiredWSV2_SkipsStickyHTTPAccount(t *testing.T) {
	ctx := context.Background()
	groupID := int64(1011)
	accounts := []Account{
		{
			ID:          2201,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          2202,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
			},
		},
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_ws_only": 2201,
		},
	}
	cfg := newSchedulerTestOpenAIWSV2Config()

	// 构造“HTTP-only 账号负载更低”的场景，验证 required transport 会强制过滤。
	concurrencyCache := schedulerTestConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			2201: {AccountID: 2201, LoadRate: 0, WaitingCount: 0},
			2202: {AccountID: 2202, LoadRate: 90, WaitingCount: 5},
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"session_hash_ws_only",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportResponsesWebsocketV2,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(2202), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	require.Equal(t, 1, decision.CandidateCount)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_RequiredWSV2_NoAvailableAccount(t *testing.T) {
	ctx := context.Background()
	groupID := int64(1012)
	accounts := []Account{
		{
			ID:          2301,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                newSchedulerTestOpenAIWSV2Config(),
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportResponsesWebsocketV2,
		false,
	)
	require.Error(t, err)
	require.Nil(t, selection)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, 0, decision.CandidateCount)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_LoadBalanceTopKFallback(t *testing.T) {
	ctx := context.Background()
	groupID := int64(11)
	accounts := []Account{
		{
			ID:          3001,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          3002,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          3003,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
	}

	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.LBTopK = 2
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Priority = 0.4
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Load = 1.0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Queue = 1.0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.ErrorRate = 0.2
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.TTFT = 0.1

	concurrencyCache := schedulerTestConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			3001: {AccountID: 3001, LoadRate: 95, WaitingCount: 8},
			3002: {AccountID: 3002, LoadRate: 20, WaitingCount: 1},
			3003: {AccountID: 3003, LoadRate: 10, WaitingCount: 0},
		},
		acquireResults: map[int64]bool{
			3003: false, // top1 失败，必须回退到 top-K 的下一候选
			3002: true,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(3002), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, 3, decision.CandidateCount)
	require.Equal(t, 2, decision.TopK)
	require.Greater(t, decision.LoadSkew, 0.0)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_OpenAIAccountSchedulerMetrics(t *testing.T) {
	ctx := context.Background()
	groupID := int64(12)
	account := Account{
		ID:          4001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_metrics": account.ID,
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              cache,
		cfg:                newOpenAIStickyEnabledTestConfig(),
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, _, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_metrics", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	svc.ReportOpenAIAccountScheduleResult(account.ID, true, intPtrForTest(120))
	svc.RecordOpenAIAccountSwitch()

	snapshot := svc.SnapshotOpenAIAccountSchedulerMetrics()
	require.GreaterOrEqual(t, snapshot.SelectTotal, int64(1))
	require.GreaterOrEqual(t, snapshot.StickySessionHitTotal, int64(1))
	require.GreaterOrEqual(t, snapshot.AccountSwitchTotal, int64(1))
	require.GreaterOrEqual(t, snapshot.SchedulerLatencyMsAvg, float64(0))
	require.GreaterOrEqual(t, snapshot.StickyHitRatio, 0.0)
	require.GreaterOrEqual(t, snapshot.RuntimeStatsAccountCount, 1)
}

func intPtrForTest(v int) *int {
	return &v
}

func TestOpenAIAccountRuntimeStats_ReportAndSnapshot(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	stats.report(1001, true, nil)
	firstTTFT := 100
	stats.report(1001, false, &firstTTFT)
	secondTTFT := 200
	stats.report(1001, false, &secondTTFT)

	errorRate, ttft, hasTTFT := stats.snapshot(1001)
	require.True(t, hasTTFT)
	require.InDelta(t, 0.36, errorRate, 1e-9)
	require.InDelta(t, 120.0, ttft, 1e-9)
	require.Equal(t, 1, stats.size())
}

func TestOpenAIAccountRuntimeStats_ReportConcurrent(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()

	const (
		accountCount = 4
		workers      = 16
		iterations   = 800
	)
	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := 0; worker < workers; worker++ {
		worker := worker
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				accountID := int64(i%accountCount + 1)
				success := (i+worker)%3 != 0
				ttft := 80 + (i+worker)%40
				stats.report(accountID, success, &ttft)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, accountCount, stats.size())
	for accountID := int64(1); accountID <= accountCount; accountID++ {
		errorRate, ttft, hasTTFT := stats.snapshot(accountID)
		require.GreaterOrEqual(t, errorRate, 0.0)
		require.LessOrEqual(t, errorRate, 1.0)
		require.True(t, hasTTFT)
		require.Greater(t, ttft, 0.0)
	}
}

func TestSelectTopKOpenAICandidates(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		{
			account:  &Account{ID: 11, Priority: 2},
			loadInfo: &AccountLoadInfo{LoadRate: 10, WaitingCount: 1},
			score:    10.0,
		},
		{
			account:  &Account{ID: 12, Priority: 1},
			loadInfo: &AccountLoadInfo{LoadRate: 20, WaitingCount: 1},
			score:    9.5,
		},
		{
			account:  &Account{ID: 13, Priority: 1},
			loadInfo: &AccountLoadInfo{LoadRate: 30, WaitingCount: 0},
			score:    10.0,
		},
		{
			account:  &Account{ID: 14, Priority: 0},
			loadInfo: &AccountLoadInfo{LoadRate: 40, WaitingCount: 0},
			score:    8.0,
		},
	}

	top2 := selectTopKOpenAICandidates(candidates, 2)
	require.Len(t, top2, 2)
	require.Equal(t, int64(13), top2[0].account.ID)
	require.Equal(t, int64(11), top2[1].account.ID)

	topAll := selectTopKOpenAICandidates(candidates, 8)
	require.Len(t, topAll, len(candidates))
	require.Equal(t, int64(13), topAll[0].account.ID)
	require.Equal(t, int64(11), topAll[1].account.ID)
	require.Equal(t, int64(12), topAll[2].account.ID)
	require.Equal(t, int64(14), topAll[3].account.ID)
}

func TestBuildOpenAIWeightedSelectionOrder_DeterministicBySessionSeed(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		{
			account:  &Account{ID: 101},
			loadInfo: &AccountLoadInfo{LoadRate: 10, WaitingCount: 0},
			score:    4.2,
		},
		{
			account:  &Account{ID: 102},
			loadInfo: &AccountLoadInfo{LoadRate: 30, WaitingCount: 1},
			score:    3.5,
		},
		{
			account:  &Account{ID: 103},
			loadInfo: &AccountLoadInfo{LoadRate: 50, WaitingCount: 2},
			score:    2.1,
		},
	}
	req := OpenAIAccountScheduleRequest{
		GroupID:        int64PtrForTest(99),
		SessionHash:    "session_seed_fixed",
		RequestedModel: "gpt-5.1",
	}

	first := buildOpenAIWeightedSelectionOrder(candidates, req)
	second := buildOpenAIWeightedSelectionOrder(candidates, req)
	require.Len(t, first, len(candidates))
	require.Len(t, second, len(candidates))
	for i := range first {
		require.Equal(t, first[i].account.ID, second[i].account.ID)
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_LoadBalanceDistributesAcrossSessions(t *testing.T) {
	ctx := context.Background()
	groupID := int64(15)
	accounts := []Account{
		{
			ID:          5101,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 3,
			Priority:    0,
		},
		{
			ID:          5102,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 3,
			Priority:    0,
		},
		{
			ID:          5103,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 3,
			Priority:    0,
		},
	}
	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.LBTopK = 3
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Priority = 1
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Load = 1
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Queue = 1
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.ErrorRate = 1
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.TTFT = 1

	concurrencyCache := schedulerTestConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			5101: {AccountID: 5101, LoadRate: 20, WaitingCount: 1},
			5102: {AccountID: 5102, LoadRate: 20, WaitingCount: 1},
			5103: {AccountID: 5103, LoadRate: 20, WaitingCount: 1},
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{sessionBindings: map[string]int64{}},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	selected := make(map[int64]int, len(accounts))
	for i := 0; i < 60; i++ {
		sessionHash := fmt.Sprintf("session_hash_lb_%d", i)
		selection, decision, err := svc.SelectAccountWithScheduler(
			ctx,
			&groupID,
			"",
			sessionHash,
			"gpt-5.1",
			nil,
			OpenAIUpstreamTransportAny,
			false,
		)
		require.NoError(t, err)
		require.NotNil(t, selection)
		require.NotNil(t, selection.Account)
		require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
		selected[selection.Account.ID]++
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
	}

	// 多 session 应该能打散到多个账号，避免“恒定单账号命中”。
	require.GreaterOrEqual(t, len(selected), 2)
}

func TestDeriveOpenAISelectionSeed_NoAffinityAddsEntropy(t *testing.T) {
	req := OpenAIAccountScheduleRequest{
		RequestedModel: "gpt-5.1",
	}
	seed1 := deriveOpenAISelectionSeed(req)
	time.Sleep(1 * time.Millisecond)
	seed2 := deriveOpenAISelectionSeed(req)
	require.NotZero(t, seed1)
	require.NotZero(t, seed2)
	require.NotEqual(t, seed1, seed2)
}

func TestBuildOpenAIWeightedSelectionOrder_HandlesInvalidScores(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		{
			account:  &Account{ID: 901},
			loadInfo: &AccountLoadInfo{LoadRate: 5, WaitingCount: 0},
			score:    math.NaN(),
		},
		{
			account:  &Account{ID: 902},
			loadInfo: &AccountLoadInfo{LoadRate: 5, WaitingCount: 0},
			score:    math.Inf(1),
		},
		{
			account:  &Account{ID: 903},
			loadInfo: &AccountLoadInfo{LoadRate: 5, WaitingCount: 0},
			score:    -1,
		},
	}
	req := OpenAIAccountScheduleRequest{
		SessionHash: "seed_invalid_scores",
	}

	order := buildOpenAIWeightedSelectionOrder(candidates, req)
	require.Len(t, order, len(candidates))
	seen := map[int64]struct{}{}
	for _, item := range order {
		seen[item.account.ID] = struct{}{}
	}
	require.Len(t, seen, len(candidates))
}

func TestOpenAISelectionRNG_SeedZeroStillWorks(t *testing.T) {
	rng := newOpenAISelectionRNG(0)
	v1 := rng.nextUint64()
	v2 := rng.nextUint64()
	require.NotEqual(t, v1, v2)
	require.GreaterOrEqual(t, rng.nextFloat64(), 0.0)
	require.Less(t, rng.nextFloat64(), 1.0)
}

func TestOpenAIAccountCandidateHeap_PushPopAndInvalidType(t *testing.T) {
	h := openAIAccountCandidateHeap{}
	h.Push(openAIAccountCandidateScore{
		account:  &Account{ID: 7001},
		loadInfo: &AccountLoadInfo{LoadRate: 0, WaitingCount: 0},
		score:    1.0,
	})
	require.Equal(t, 1, h.Len())
	popped, ok := h.Pop().(openAIAccountCandidateScore)
	require.True(t, ok)
	require.Equal(t, int64(7001), popped.account.ID)
	require.Equal(t, 0, h.Len())

	require.Panics(t, func() {
		h.Push("bad_element_type")
	})
}

func TestClamp01_AllBranches(t *testing.T) {
	require.Equal(t, 0.0, clamp01(-0.2))
	require.Equal(t, 1.0, clamp01(1.3))
	require.Equal(t, 0.5, clamp01(0.5))
}

func TestCalcLoadSkewByMoments_Branches(t *testing.T) {
	require.Equal(t, 0.0, calcLoadSkewByMoments(1, 1, 1))
	// variance < 0 分支：sumSquares/count - mean^2 为负值时应钳制为 0。
	require.Equal(t, 0.0, calcLoadSkewByMoments(1, 0, 2))
	require.GreaterOrEqual(t, calcLoadSkewByMoments(6, 20, 3), 0.0)
}

func TestDefaultOpenAIAccountScheduler_ReportSwitchAndSnapshot(t *testing.T) {
	schedulerAny := newDefaultOpenAIAccountScheduler(&OpenAIGatewayService{}, nil)
	scheduler, ok := schedulerAny.(*defaultOpenAIAccountScheduler)
	require.True(t, ok)

	ttft := 100
	scheduler.ReportResult(1001, true, &ttft)
	scheduler.ReportSwitch()
	scheduler.metrics.recordSelect(OpenAIAccountScheduleDecision{
		Layer:             openAIAccountScheduleLayerLoadBalance,
		LatencyMs:         8,
		LoadSkew:          0.5,
		StickyPreviousHit: true,
	})
	scheduler.metrics.recordSelect(OpenAIAccountScheduleDecision{
		Layer:            openAIAccountScheduleLayerSessionSticky,
		LatencyMs:        6,
		LoadSkew:         0.2,
		StickySessionHit: true,
	})

	snapshot := scheduler.SnapshotMetrics()
	require.Equal(t, int64(2), snapshot.SelectTotal)
	require.Equal(t, int64(1), snapshot.StickyPreviousHitTotal)
	require.Equal(t, int64(1), snapshot.StickySessionHitTotal)
	require.Equal(t, int64(1), snapshot.LoadBalanceSelectTotal)
	require.Equal(t, int64(1), snapshot.AccountSwitchTotal)
	require.Greater(t, snapshot.SchedulerLatencyMsAvg, 0.0)
	require.Greater(t, snapshot.StickyHitRatio, 0.0)
	require.Greater(t, snapshot.LoadSkewAvg, 0.0)
}

func TestOpenAIGatewayService_SchedulerWrappersAndDefaults(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	svc := &OpenAIGatewayService{}
	ttft := 120
	svc.ReportOpenAIAccountScheduleResult(10, true, &ttft)
	svc.RecordOpenAIAccountSwitch()
	snapshot := svc.SnapshotOpenAIAccountSchedulerMetrics()
	require.Equal(t, OpenAIAccountSchedulerMetricsSnapshot{}, snapshot)
	require.Equal(t, 7, svc.openAIWSLBTopK())
	require.Equal(t, openaiStickySessionTTL, svc.openAIWSSessionStickyTTL())

	defaultWeights := svc.openAIWSSchedulerWeights()
	require.Equal(t, 1.0, defaultWeights.Priority)
	require.Equal(t, 1.0, defaultWeights.Load)
	require.Equal(t, 0.7, defaultWeights.Queue)
	require.Equal(t, 0.8, defaultWeights.ErrorRate)
	require.Equal(t, 0.5, defaultWeights.TTFT)

	cfg := newOpenAIStickyEnabledTestConfig()
	cfg.Gateway.OpenAIWS.LBTopK = 9
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 180
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Priority = 0.2
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Load = 0.3
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Queue = 0.4
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.ErrorRate = 0.5
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.TTFT = 0.6
	svcWithCfg := &OpenAIGatewayService{cfg: cfg}

	require.Equal(t, 9, svcWithCfg.openAIWSLBTopK())
	require.Equal(t, 180*time.Second, svcWithCfg.openAIWSSessionStickyTTL())
	customWeights := svcWithCfg.openAIWSSchedulerWeights()
	require.Equal(t, 0.2, customWeights.Priority)
	require.Equal(t, 0.3, customWeights.Load)
	require.Equal(t, 0.4, customWeights.Queue)
	require.Equal(t, 0.5, customWeights.ErrorRate)
	require.Equal(t, 0.6, customWeights.TTFT)
}

func TestDefaultOpenAIAccountScheduler_IsAccountTransportCompatible_Branches(t *testing.T) {
	scheduler := &defaultOpenAIAccountScheduler{}
	require.True(t, scheduler.isAccountTransportCompatible(nil, OpenAIUpstreamTransportAny))
	require.True(t, scheduler.isAccountTransportCompatible(nil, OpenAIUpstreamTransportHTTPSSE))
	require.False(t, scheduler.isAccountTransportCompatible(nil, OpenAIUpstreamTransportResponsesWebsocketV2))

	cfg := newSchedulerTestOpenAIWSV2Config()
	scheduler.service = &OpenAIGatewayService{cfg: cfg}
	account := &Account{
		ID:          8801,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Extra: map[string]any{
			"openai_apikey_responses_websockets_v2_enabled": true,
		},
	}
	require.True(t, scheduler.isAccountTransportCompatible(account, OpenAIUpstreamTransportResponsesWebsocketV2))
}

func int64PtrForTest(v int64) *int64 {
	return &v
}
