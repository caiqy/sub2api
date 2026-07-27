//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/server"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

type Application struct {
	Server      *http.Server
	PromptAudit *securityaudit.PromptService
	Cleanup     func()
}

func initializeApplication(buildInfo handler.BuildInfo) (*Application, error) {
	wire.Build(
		// Infrastructure layer ProviderSets
		config.ProviderSet,

		// Business layer ProviderSets
		repository.ProviderSet,
		service.ProviderSet,
		securityaudit.ProviderSet,
		payment.ProviderSet,
		middleware.ProviderSet,
		handler.ProviderSet,

		// Server layer ProviderSet
		server.ProviderSet,

		// Privacy client factory for OpenAI training opt-out
		providePrivacyClientFactory,

		// BuildInfo provider
		provideServiceBuildInfo,

		// Cleanup function provider
		provideCleanup,

		// Application struct
		wire.Struct(new(Application), "Server", "PromptAudit", "Cleanup"),
	)
	return nil, nil
}

func providePrivacyClientFactory() service.PrivacyClientFactory {
	return repository.CreatePrivacyReqClient
}

func provideServiceBuildInfo(buildInfo handler.BuildInfo) service.BuildInfo {
	return service.BuildInfo{
		Version:   buildInfo.Version,
		BuildType: buildInfo.BuildType,
	}
}

func provideCleanup(
	entClient *ent.Client,
	rdb *redis.Client,
	opsMetricsCollector *service.OpsMetricsCollector,
	opsAggregation *service.OpsAggregationService,
	opsAlertEvaluator *service.OpsAlertEvaluatorService,
	opsCleanup *service.OpsCleanupService,
	opsScheduledReport *service.OpsScheduledReportService,
	opsSystemLogSink *service.OpsSystemLogSink,
	opsService *service.OpsService,
	opsIngressReject *service.OpsIngressRejectAggregator,
	apiKeyService *service.APIKeyService,
	authCacheInvalidationWorker *service.AuthCacheInvalidationWorker,
	schedulerSnapshot *service.SchedulerSnapshotService,
	tokenRefresh *service.TokenRefreshService,
	accountExpiry *service.AccountExpiryService,
	proxyExpiry *service.ProxyExpiryService,
	subscriptionExpiry *service.SubscriptionExpiryService,
	usageCleanup *service.UsageCleanupService,
	idempotencyCleanup *service.IdempotencyCleanupService,
	batchImageCleanup *service.BatchImageCleanupService,
	batchImageWorker *service.BatchImageWorkerRuntime,
	pricing *service.PricingService,
	emailQueue *service.EmailQueueService,
	billingCache *service.BillingCacheService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	subscriptionService *service.SubscriptionService,
	oauth *service.OAuthService,
	openaiOAuth *service.OpenAIOAuthService,
	geminiOAuth *service.GeminiOAuthService,
	antigravityOAuth *service.AntigravityOAuthService,
	grokOAuth *service.GrokOAuthService,
	openAIGateway *service.OpenAIGatewayService,
	scheduledTestRunner *service.ScheduledTestRunnerService,
	backupSvc *service.BackupService,
	paymentOrderExpiry *service.PaymentOrderExpiryService,
	channelMonitorRunner *service.ChannelMonitorRunner,
	quotaFlusher *service.UserPlatformQuotaUsageFlusher,
	upstreamBillingProbe *service.UpstreamBillingProbeService,
	auditLog *service.AuditLogService,
	promptAudit *securityaudit.PromptService,
) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 独立 producers 可并行停止；usage、quota、billing 和基础设施必须按依赖顺序 drain。
		producerSteps := []cleanupPhase{
			{name: "OpsIngressRejectAggregator", run: func(context.Context) error {
				if opsIngressReject != nil {
					opsIngressReject.Stop()
				}
				return nil
			}},
			{name: "AuthCacheInvalidationWorker", run: func(context.Context) error {
				if authCacheInvalidationWorker != nil {
					authCacheInvalidationWorker.Stop()
				}
				return nil
			}},
			{name: "AuthCacheInvalidationSubscriber", run: func(context.Context) error {
				if apiKeyService != nil {
					apiKeyService.StopAuthCacheInvalidationSubscriber()
				}
				return nil
			}},
			{name: "OpsRuntimeSettingsRefresh", run: func(context.Context) error {
				if opsService != nil {
					opsService.StopRuntimeSettingsRefresh()
				}
				return nil
			}},
			{name: "PromptAuditService", run: func(ctx context.Context) error {
				if promptAudit != nil {
					return promptAudit.Shutdown(ctx)
				}
				return nil
			}},
			{name: "OpsScheduledReportService", run: func(context.Context) error {
				if opsScheduledReport != nil {
					opsScheduledReport.Stop()
				}
				return nil
			}},
			{name: "OpsCleanupService", run: func(context.Context) error {
				if opsCleanup != nil {
					opsCleanup.Stop()
				}
				return nil
			}},
			{name: "OpsSystemLogSink", run: func(context.Context) error {
				if opsSystemLogSink != nil {
					opsSystemLogSink.Stop()
				}
				return nil
			}},
			{name: "AuditLogService", run: func(context.Context) error {
				if auditLog != nil {
					auditLog.Stop()
				}
				return nil
			}},
			{name: "OpsAlertEvaluatorService", run: func(context.Context) error {
				if opsAlertEvaluator != nil {
					opsAlertEvaluator.Stop()
				}
				return nil
			}},
			{name: "OpsAggregationService", run: func(context.Context) error {
				if opsAggregation != nil {
					opsAggregation.Stop()
				}
				return nil
			}},
			{name: "OpsMetricsCollector", run: func(context.Context) error {
				if opsMetricsCollector != nil {
					opsMetricsCollector.Stop()
				}
				return nil
			}},
			{name: "SchedulerSnapshotService", run: func(context.Context) error {
				if schedulerSnapshot != nil {
					schedulerSnapshot.Stop()
				}
				return nil
			}},
			{name: "UsageCleanupService", run: func(context.Context) error {
				if usageCleanup != nil {
					usageCleanup.Stop()
				}
				return nil
			}},
			{name: "IdempotencyCleanupService", run: func(context.Context) error {
				if idempotencyCleanup != nil {
					idempotencyCleanup.Stop()
				}
				return nil
			}},
			{name: "BatchImageCleanupService", run: func(context.Context) error {
				if batchImageCleanup != nil {
					batchImageCleanup.Stop()
				}
				return nil
			}},
			{name: "BatchImageWorkerRuntime", run: func(context.Context) error {
				if batchImageWorker != nil {
					batchImageWorker.Stop()
				}
				return nil
			}},
			{name: "TokenRefreshService", run: func(context.Context) error {
				tokenRefresh.Stop()
				return nil
			}},
			{name: "AccountExpiryService", run: func(context.Context) error {
				accountExpiry.Stop()
				return nil
			}},
			{name: "ProxyExpiryService", run: func(context.Context) error {
				proxyExpiry.Stop()
				return nil
			}},
			{name: "SubscriptionExpiryService", run: func(context.Context) error {
				subscriptionExpiry.Stop()
				return nil
			}},
			{name: "SubscriptionService", run: func(context.Context) error {
				if subscriptionService != nil {
					subscriptionService.Stop()
				}
				return nil
			}},
			{name: "PricingService", run: func(context.Context) error {
				pricing.Stop()
				return nil
			}},
			{name: "EmailQueueService", run: func(context.Context) error {
				emailQueue.Stop()
				return nil
			}},
			{name: "OAuthService", run: func(context.Context) error {
				oauth.Stop()
				return nil
			}},
			{name: "OpenAIOAuthService", run: func(context.Context) error {
				openaiOAuth.Stop()
				return nil
			}},
			{name: "GeminiOAuthService", run: func(context.Context) error {
				geminiOAuth.Stop()
				return nil
			}},
			{name: "AntigravityOAuthService", run: func(context.Context) error {
				antigravityOAuth.Stop()
				return nil
			}},
			{name: "GrokOAuthService", run: func(context.Context) error {
				if grokOAuth != nil {
					grokOAuth.Stop()
				}
				return nil
			}},
			{name: "OpenAIWSPool", run: func(context.Context) error {
				if openAIGateway != nil {
					openAIGateway.CloseOpenAIWSPool()
				}
				return nil
			}},
			{name: "ScheduledTestRunnerService", run: func(context.Context) error {
				if scheduledTestRunner != nil {
					scheduledTestRunner.Stop()
				}
				return nil
			}},
			{name: "BackupService", run: func(context.Context) error {
				if backupSvc != nil {
					backupSvc.Stop()
				}
				return nil
			}},
			{name: "PaymentOrderExpiryService", run: func(context.Context) error {
				if paymentOrderExpiry != nil {
					paymentOrderExpiry.Stop()
				}
				return nil
			}},
			{name: "ChannelMonitorRunner", run: func(context.Context) error {
				if channelMonitorRunner != nil {
					channelMonitorRunner.Stop()
				}
				return nil
			}},
			{name: "UpstreamBillingProbeService", run: func(context.Context) error {
				if upstreamBillingProbe != nil {
					upstreamBillingProbe.Stop()
				}
				return nil
			}},
		}

		phases := []cleanupPhase{
			{name: "producers", run: func(ctx context.Context) error {
				return runCleanupParallel(ctx, producerSteps...)
			}},
			{name: "usage-record-drain", run: func(context.Context) error {
				if usageRecordWorkerPool != nil {
					usageRecordWorkerPool.Stop()
				}
				return nil
			}},
			{name: "quota-final-flush", run: func(context.Context) error {
				if quotaFlusher != nil {
					quotaFlusher.Stop()
				}
				return nil
			}},
			{name: "billing-cache-drain", run: func(context.Context) error {
				if billingCache != nil {
					billingCache.Stop()
				}
				return nil
			}},
			{name: "Redis", run: func(context.Context) error {
				if rdb == nil {
					return nil
				}
				return rdb.Close()
			}},
			{name: "Ent", run: func(context.Context) error {
				if entClient == nil {
					return nil
				}
				return entClient.Close()
			}},
		}
		if runCleanupPhases(ctx, phases...) {
			log.Printf("[Cleanup] All cleanup phases completed")
		}
	}
}
