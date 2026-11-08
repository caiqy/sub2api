package service

import "github.com/Wei-Shaw/sub2api/internal/config"

// Shared eligibility regressions use the retained upstream scheduler.
func newDefaultSchedulerTestService(accounts []Account) *OpenAIGatewayService {
	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = true
	cfg.Gateway.OpenAIWS.LBTopK = 7
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 3600
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600
	return &OpenAIGatewayService{
		cfg:                cfg,
		accountRepo:        stubOpenAIAccountRepo{accounts: accounts},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}
}
