//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/stretchr/testify/require"
)

type legacyVersionedUsageSubRepo struct {
	userSubRepoNoop
	version int64
	calls   int
}

type legacyUnversionedUsageSubRepo struct {
	userSubRepoNoop
	calls int
}

func (r *legacyUnversionedUsageSubRepo) IncrementUsage(context.Context, int64, float64) error {
	r.calls++
	return nil
}

type legacyFallbackInvalidatingCache struct {
	billingCacheWorkerStub
	invalidated bool
	published   bool
}

func (c *legacyFallbackInvalidatingCache) InvalidateSubscriptionCache(context.Context, int64, int64) error {
	c.invalidated = true
	return nil
}

func (c *legacyFallbackInvalidatingCache) PublishSubscriptionCacheInvalidation(context.Context, string) error {
	c.published = true
	return nil
}

func (*legacyFallbackInvalidatingCache) SubscribeSubscriptionCacheInvalidation(context.Context, func(string)) error {
	return nil
}

func (r *legacyVersionedUsageSubRepo) IncrementUsage(context.Context, int64, float64) error {
	r.calls++
	return nil
}

func (r *legacyVersionedUsageSubRepo) IncrementUsageWithVersion(context.Context, int64, float64) (int64, error) {
	r.calls++
	return r.version, nil
}

func TestApplyUsageBillingLegacySubscriptionQueuesCommittedUsageVersion(t *testing.T) {
	groupID := int64(20)
	repo := &legacyVersionedUsageSubRepo{version: 1234567890123456000}
	cache := &versionRecordingBillingCache{}

	applied, err := applyUsageBilling(context.Background(), "legacy-version", nil, &postUsageBillingParams{
		Cost:               &CostBreakdown{ActualCost: 2.5},
		User:               &User{ID: 10},
		APIKey:             &APIKey{GroupID: &groupID},
		Subscription:       &UserSubscription{ID: 30},
		IsSubscriptionBill: true,
	}, &billingDeps{
		userSubRepo:         repo,
		billingCacheService: &BillingCacheService{cache: cache},
	}, nil)

	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, 1, repo.calls)
	require.Equal(t, repo.version, cache.usageVersion.Load(), "legacy billing must queue the version returned by the DB increment")
}

func TestApplyUsageBillingLegacySubscriptionInvalidatesWhenVersionUnavailable(t *testing.T) {
	groupID := int64(20)
	repo := &legacyUnversionedUsageSubRepo{}
	cache := &legacyFallbackInvalidatingCache{}
	l1 := &trackingSubCache{}
	billingCacheService := &BillingCacheService{
		cache: cache,
	}
	subscriptionService := NewSubscriptionService(groupRepoNoop{}, repo, billingCacheService, nil, nil)
	subscriptionService.subCacheL1 = l1

	applied, err := applyUsageBilling(context.Background(), "legacy-invalidate", nil, &postUsageBillingParams{
		Cost:               &CostBreakdown{ActualCost: 2.5},
		User:               &User{ID: 10},
		APIKey:             &APIKey{GroupID: &groupID},
		Subscription:       &UserSubscription{ID: 30},
		IsSubscriptionBill: true,
	}, &billingDeps{
		userSubRepo:         repo,
		billingCacheService: billingCacheService,
	}, nil)

	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, 1, repo.calls)
	require.Equal(t, []string{subCacheKey(10, groupID)}, l1.deletedKeys, "versionless increment must synchronously clear this process's subscription L1 cache")
	require.Equal(t, 1, l1.waitCalls, "versionless increment must wait for the local L1 deletion")
	require.False(t, cache.invalidated, "versionless increment must not delete Redis L2")
	require.False(t, cache.published, "versionless increment must not publish before the durable outbox")
}

func TestApplyUsageBillingLegacySubscriptionNonpositiveVersionOnlyClearsLocalL1(t *testing.T) {
	groupID := int64(20)
	repo := &legacyVersionedUsageSubRepo{version: 0}
	cache := &legacyFallbackInvalidatingCache{}
	l1 := &trackingSubCache{}
	billingCacheService := &BillingCacheService{cache: cache}
	subscriptionService := NewSubscriptionService(groupRepoNoop{}, repo, billingCacheService, nil, nil)
	subscriptionService.subCacheL1 = l1

	applied, err := applyUsageBilling(context.Background(), "legacy-nonpositive-version", nil, &postUsageBillingParams{
		Cost:               &CostBreakdown{ActualCost: 2.5},
		User:               &User{ID: 10},
		APIKey:             &APIKey{GroupID: &groupID},
		Subscription:       &UserSubscription{ID: 30},
		IsSubscriptionBill: true,
	}, &billingDeps{
		userSubRepo:         repo,
		billingCacheService: billingCacheService,
	}, nil)

	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, []string{subCacheKey(10, groupID)}, l1.deletedKeys)
	require.Equal(t, 1, l1.waitCalls)
	require.False(t, cache.invalidated, "nonpositive versions must not delete Redis L2")
	require.False(t, cache.published, "nonpositive versions must not publish invalidations")
}

// composite 分组的公开别名经 BillingModelSource 来源覆盖成为计费模型后有两类错计：
// 任意别名（如 team/best）查无价静默落 $0；含家族词的别名（如 all/claude）被价格表
// 家族模糊匹配错计（Opus 流量按 Sonnet 兜底价）。compositeBillableModel 要求别名必须
// 有显式渠道定价才可参与计费，否则回退实际转发的具体模型。
func TestCompositeBillableModel(t *testing.T) {
	svc := &GatewayService{billingService: NewBillingService(&config.Config{}, nil)}
	apiKey := &APIKey{}
	ctx := context.Background()

	// 别名无渠道定价（含家族词也一样）→ 回退具体模型
	require.Equal(t, "claude-opus-4-7",
		svc.compositeBillableModel(ctx, apiKey, "all/claude", "claude-opus-4-7"))
	require.Equal(t, "claude-sonnet-4",
		svc.compositeBillableModel(ctx, apiKey, "team/best", "claude-sonnet-4"))

	// 未发生来源覆盖（计费模型已是具体模型）→ 原样返回
	require.Equal(t, "claude-sonnet-4",
		svc.compositeBillableModel(ctx, apiKey, "claude-sonnet-4", "claude-sonnet-4"))

	// 具体模型缺失 → 保持原值（走后续通用兜底/既有路径）
	require.Equal(t, "all/claude",
		svc.compositeBillableModel(ctx, apiKey, "all/claude", ""))
}

// billableModelWithFallback 是通用安全网：选定计费模型查不到任何价格时回退到
// 实际转发的具体模型；已定价流量（含家族兜底可解析的名字）不受影响。
func TestBillableModelWithFallback(t *testing.T) {
	svc := &GatewayService{billingService: NewBillingService(&config.Config{}, nil)}
	apiKey := &APIKey{}
	ctx := context.Background()

	// 完全无价的别名 → 回退到具体转发模型（claude-sonnet-4 有内置回退价格）
	require.Equal(t, "claude-sonnet-4",
		svc.billableModelWithFallback(ctx, apiKey, "team/best", "", "claude-sonnet-4"))

	// 已定价模型不回退，候选被忽略
	require.Equal(t, "claude-sonnet-4",
		svc.billableModelWithFallback(ctx, apiKey, "claude-sonnet-4", "claude-opus-4"))

	// 所有候选都无价 → 保持原值，走既有 warn + 零成本路径
	require.Equal(t, "team/best",
		svc.billableModelWithFallback(ctx, apiKey, "team/best", "another/alias", ""))

	// 空计费模型 + 有价候选 → 取候选
	require.Equal(t, "claude-sonnet-4",
		svc.billableModelWithFallback(ctx, apiKey, "", "claude-sonnet-4"))
}

func TestHasResolvableTokenPricing(t *testing.T) {
	svc := &GatewayService{billingService: NewBillingService(&config.Config{}, nil)}
	apiKey := &APIKey{}
	ctx := context.Background()

	require.True(t, svc.hasResolvableTokenPricing(ctx, "claude-sonnet-4", apiKey))
	// 注意：含家族词的名字（all/claude）会被价格表家族兜底解析为"有价"，
	// 这正是 compositeBillableModel 必须先于通用兜底拦截别名的原因。
	require.True(t, svc.hasResolvableTokenPricing(ctx, "all/claude", apiKey))
	require.False(t, svc.hasResolvableTokenPricing(ctx, "team/best", apiKey))
	require.False(t, svc.hasResolvableTokenPricing(ctx, "", apiKey))

	// billingService 缺失时 fail-closed（不误判有价）
	empty := &GatewayService{}
	require.False(t, empty.hasResolvableTokenPricing(ctx, "claude-sonnet-4", apiKey))
}
