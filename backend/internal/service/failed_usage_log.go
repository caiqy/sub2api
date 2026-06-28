package service

import (
	"context"
	"time"
)

// FailedUsageLogInput 描述失败请求的零成本 usage log 写入参数。
// 该路径只负责保留审计/诊断信息，不参与扣费。
type FailedUsageLogInput struct {
	APIKey           *APIKey
	User             *User
	Account          *Account
	Model            string
	UpstreamModel    string
	ReasoningEffort  *string
	Stream           bool
	OpenAIWSMode     bool
	InboundEndpoint  string
	UpstreamEndpoint string
	UserAgent        string
	IPAddress        string
	DetailSnapshot   *UsageLogDetailSnapshot
	Duration         time.Duration
}

// WriteFailedUsageLogBestEffort 为失败请求写入零 token / 零成本 usage log。
// 该函数为 best-effort：依赖缺失时静默返回，写库失败由底层统一记录日志。
func WriteFailedUsageLogBestEffort(ctx context.Context, repo UsageLogRepository, input *FailedUsageLogInput, logKey string) {
	if repo == nil || input == nil || input.APIKey == nil || input.User == nil || input.Account == nil {
		return
	}

	durationMs := int(input.Duration.Milliseconds())
	accountRateMultiplier := input.Account.BillingRateMultiplier()
	usageLog := &UsageLog{
		UserID:                input.User.ID,
		APIKeyID:              input.APIKey.ID,
		AccountID:             input.Account.ID,
		RequestID:             resolveUsageBillingRequestID(ctx, ""),
		Model:                 input.Model,
		UpstreamModel:         optionalNonEqualStringPtr(input.UpstreamModel, input.Model),
		ReasoningEffort:       input.ReasoningEffort,
		InboundEndpoint:       optionalTrimmedStringPtr(input.InboundEndpoint),
		UpstreamEndpoint:      optionalTrimmedStringPtr(input.UpstreamEndpoint),
		TotalCost:             0,
		ActualCost:            0,
		RateMultiplier:        1,
		AccountRateMultiplier: &accountRateMultiplier,
		BillingType:           BillingTypeBalance,
		DetailSnapshot:        input.DetailSnapshot.Normalize(),
		Stream:                input.Stream,
		OpenAIWSMode:          input.OpenAIWSMode,
		DurationMs:            &durationMs,
		CreatedAt:             time.Now(),
	}
	usageLog.SyncRequestTypeAndLegacyFields()

	if input.APIKey.GroupID != nil {
		usageLog.GroupID = input.APIKey.GroupID
	}
	if input.UserAgent != "" {
		usageLog.UserAgent = &input.UserAgent
	}
	if input.IPAddress != "" {
		usageLog.IPAddress = &input.IPAddress
	}

	writeUsageLogBestEffort(ctx, repo, usageLog, logKey)
}
