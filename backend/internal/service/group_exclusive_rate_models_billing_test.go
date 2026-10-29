//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestExclusiveRateModelsBillingQuoteAndCacheStorage(t *testing.T) {
	rate := 0.12
	group := &Group{ID: 2, RateMultiplier: 0.225, ExclusiveRateModels: GroupExclusiveRateModels{Enabled: true, Models: []string{"client-A"}}}
	user := &User{ID: 1}
	key := &APIKey{ID: 3, GroupID: &group.ID, Group: group, User: user}
	for _, client := range []string{"client-A", "client-B", "CLIENT-A", "client-A"} {
		t.Run(client, func(t *testing.T) {
			want := group.RateMultiplier
			if client == "client-A" {
				want = rate
			}
			log := &openAIRecordUsageLogRepoStub{inserted: true}
			userRepo := &openAIRecordUsageUserRepoStub{}
			svc := newGatewayRecordUsageServiceForTest(log, userRepo, &openAIRecordUsageSubRepoStub{})
			svc.userGroupRateResolver = newUserGroupRateResolver(&openAIUserGroupRateRepoStub{rate: &rate}, nil, time.Minute, nil, "test")
			svc.resolver = NewModelPricingResolver(nil, svc.billingService)
			fields := ChannelUsageFields{OriginalModel: client, ChannelMappedModel: "claude-sonnet-4-5", BillingModelSource: BillingModelSourceChannelMapped}
			input := &RecordUsageInput{APIKey: key, User: user, Account: &Account{ID: 4}, ChannelUsageFields: fields,
				Result: &ForwardResult{RequestID: "rate-gw", Model: "claude-sonnet-4-5", UpstreamModel: "claude-sonnet-4-5", Usage: ClaudeUsage{InputTokens: 1000, OutputTokens: 100, CacheReadInputTokens: 500}}}
			quote := svc.QuoteUsageCost(context.Background(), input)
			require.Greater(t, quote, 0.0)
			require.NoError(t, svc.RecordUsage(context.Background(), input))
			require.Equal(t, want, log.lastLog.RateMultiplier)
			require.InDelta(t, quote, log.lastLog.ActualCost, 1e-12)
			require.InDelta(t, quote, userRepo.lastAmount, 1e-12)

			openaiLog := &openAIRecordUsageLogRepoStub{inserted: true}
			openaiUser := &openAIRecordUsageUserRepoStub{}
			openai := newOpenAIRecordUsageServiceForTest(t, openaiLog, openaiUser, &openAIRecordUsageSubRepoStub{}, &openAIUserGroupRateRepoStub{rate: &rate})
			require.NoError(t, openai.RecordUsage(context.Background(), &OpenAIRecordUsageInput{APIKey: key, User: user, Account: &Account{ID: 4}, ChannelUsageFields: ChannelUsageFields{OriginalModel: client, ChannelMappedModel: "gpt-5.1", BillingModelSource: BillingModelSourceChannelMapped}, Result: &OpenAIForwardResult{RequestID: "rate-oa", Model: "gpt-5.1", UpstreamModel: "gpt-5.1", Usage: OpenAIUsage{InputTokens: 1000, OutputTokens: 100, CacheReadInputTokens: 500}}}))
			require.Equal(t, want, openaiLog.lastLog.RateMultiplier)
			require.Greater(t, openaiUser.lastAmount, 0.0)
			require.InDelta(t, openaiLog.lastLog.ActualCost, openaiUser.lastAmount, 1e-12)
		})
	}
	// Dedicated Gemini cache storage charges use the same client model gate as token cost.
	svc, log := newGeminiCacheBillingGatewayForTest(t)
	svc.userGroupRateResolver = newUserGroupRateResolver(&openAIUserGroupRateRepoStub{rate: &rate}, nil, time.Minute, nil, "test")
	for _, client := range []string{"client-A", "client-B"} {
		input := &RecordUsageInput{APIKey: key, User: user, Account: &Account{ID: 4}, ChannelUsageFields: ChannelUsageFields{OriginalModel: client, ChannelMappedModel: "gemini-2.5-pro", BillingModelSource: BillingModelSourceChannelMapped}, Result: &ForwardResult{RequestID: "cache-" + client, Model: "gemini-2.5-pro", CacheStorageTokenHours: 1000000}}
		quote := svc.QuoteUsageCost(context.Background(), input)
		require.NoError(t, svc.RecordUsage(context.Background(), input))
		want := 0.225
		if client == "client-A" {
			want = rate
		}
		require.InDelta(t, 4.5*want, quote, 1e-12)
		require.InDelta(t, quote, log.lastLog.ActualCost, 1e-12)
	}
}

func TestExclusiveRateModelsInflightProfitAndWSTurn(t *testing.T) {
	rate := 0.12
	group := profitControlTestGroup(2, 0, 0)
	group.RateMultiplier = 0.225
	group.ExclusiveRateModels = GroupExclusiveRateModels{Enabled: true, Models: []string{"client-A"}}
	key := &APIKey{ID: 3, GroupID: &group.ID, Group: group, User: &User{ID: 1}}
	resolver := newUserGroupRateResolver(&openAIUserGroupRateRepoStub{rate: &rate}, nil, time.Minute, nil, "test")
	gw := &GatewayService{userGroupRateResolver: resolver}
	oa := &OpenAIGatewayService{userGroupRateResolver: resolver}
	deps := inflightEstimateDeps{userGroupRate: gw.getUserGroupRateMultiplier}
	base := context.WithValue(profitControlTestCtx(group), ctxkey.UserID, int64(1))
	var previous context.Context
	for _, client := range []string{"client-A", "client-B", "client-A"} {
		want := group.RateMultiplier
		if client == "client-A" {
			want = rate
		}
		text, image := deps.rates(base, key, time.Now(), client)
		require.Equal(t, want, text)
		require.Equal(t, want, image)
		ctx, _ := WithGatewayTokenRequestPricing(base, client)
		ctx = gw.withGatewayProfitControlGate(ctx, &group.ID)
		gate, _ := ctx.Value(openAIProfitControlGateCtxKey{}).(*openAIProfitControlGate)
		require.NotNil(t, gate)
		require.Equal(t, want, gate.threshold)
		ctx, _ = oa.WithOpenAIRequestPricingContext(base, &group.ID, client)
		gate, _ = ctx.Value(openAIProfitControlGateCtxKey{}).(*openAIProfitControlGate)
		require.Equal(t, want, gate.threshold)
		if previous == nil {
			previous = ctx
		}
		previous, _ = oa.WithOpenAITurnPricingContext(previous, &group.ID, client)
		gate, _ = previous.Value(openAIProfitControlGateCtxKey{}).(*openAIProfitControlGate)
		require.Equal(t, want, gate.threshold, "a WS turn must use its own original client model")
	}
}

func TestExclusiveRateModelsBatchImageSnapshot(t *testing.T) {
	svc, _, _, _, _ := newTestBatchImagePublicService(true)
	rate := 0.12
	group := &Group{ID: 2, RateMultiplier: 0.225, AllowBatchImageGeneration: true, ImagePrice1K: ptrFloat(1), BatchImageDiscountMultiplier: 0.5, BatchImageHoldMultiplier: 0.6, ExclusiveRateModels: GroupExclusiveRateModels{Enabled: true, Models: []string{"client-A"}}}
	svc.GroupRepo = &publicBatchImageGroupRepo{groups: map[int64]*Group{2: group}}
	svc.UserGroupRateRepo = &publicBatchImageUserGroupRateRepo{rates: map[int64]*float64{2: &rate}}
	for _, model := range []string{"client-A", "client-B"} {
		req := validBatchImageSubmitRequest()
		req.Model = model
		snapshot, err := svc.resolvePricingSnapshot(context.Background(), BatchImageOwner{UserID: 1, GroupID: &group.ID}, req, "gemini_api", &Account{ID: 4})
		require.NoError(t, err)
		want := group.RateMultiplier
		if model == "client-A" {
			want = rate
		}
		require.Equal(t, want, snapshot.GroupRateMultiplier)
		require.InDelta(t, want*0.5, snapshot.BillableUnitPrice, 1e-12)
		require.InDelta(t, want*0.6, snapshot.HoldUnitPrice, 1e-12)
	}
}
