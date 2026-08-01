package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

type opsDrainBeforeEntCloseDriver struct {
	dialect.Driver
	beforeClose func() error
}

func (d *opsDrainBeforeEntCloseDriver) Close() error {
	if err := d.beforeClose(); err != nil {
		return err
	}
	return d.Driver.Close()
}

type blockingCleanupOpsRepo struct {
	service.OpsRepository
	started   chan struct{}
	release   chan struct{}
	persisted atomic.Bool
}

func (r *blockingCleanupOpsRepo) InsertErrorLog(ctx context.Context, _ *service.OpsInsertErrorLogInput) (int64, error) {
	select {
	case r.started <- struct{}{}:
	default:
	}
	select {
	case <-r.release:
		r.persisted.Store(true)
		return 1, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func TestProvideCleanupDrainsOpsErrorsBeforeEntTeardown(t *testing.T) {
	repo := &blockingCleanupOpsRepo{started: make(chan struct{}, 1), release: make(chan struct{})}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(repo.release) }) }
	t.Cleanup(release)
	ops := service.NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(handler.OpsErrorLoggerMiddleware(ops))
	router.GET("/v1/messages", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "upstream failed")
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/messages", nil))
	select {
	case <-repo.started:
	case <-time.After(time.Second):
		t.Fatal("ops error worker did not start")
	}

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectClose()
	entClient := dbent.NewClient(dbent.Driver(&opsDrainBeforeEntCloseDriver{
		Driver: entsql.OpenDB(dialect.Postgres, db),
		beforeClose: func() error {
			if !repo.persisted.Load() {
				return errors.New("ops error worker was not drained before Ent teardown")
			}
			return nil
		},
	}))

	cfg := &config.Config{}
	oauthSvc := service.NewOAuthService(nil, nil)
	openAIOAuthSvc := service.NewOpenAIOAuthService(nil, nil)
	geminiOAuthSvc := service.NewGeminiOAuthService(nil, nil, nil, nil, cfg)
	antigravityOAuthSvc := service.NewAntigravityOAuthService(nil)
	cleanup := provideCleanup(
		entClient, nil, &service.OpsMetricsCollector{}, &service.OpsAggregationService{}, &service.OpsAlertEvaluatorService{},
		&service.OpsCleanupService{}, &service.OpsScheduledReportService{}, service.NewOpsSystemLogSink(nil), ops, nil, nil, nil, nil,
		service.NewSchedulerSnapshotService(nil, nil, nil, nil, cfg),
		service.NewTokenRefreshService(nil, oauthSvc, openAIOAuthSvc, geminiOAuthSvc, antigravityOAuthSvc, nil, nil, cfg, nil),
		service.NewAccountExpiryService(nil, time.Second), service.NewProxyExpiryService(nil, time.Second), service.NewSubscriptionExpiryService(nil, time.Second),
		&service.UsageCleanupService{}, service.NewIdempotencyCleanupService(nil, cfg), &service.BatchImageCleanupService{}, nil, nil,
		service.NewPricingService(cfg, nil), service.NewEmailQueueService(nil, 1), service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil),
		&service.UsageRecordWorkerPool{}, nil, &service.SubscriptionService{}, oauthSvc, openAIOAuthSvc, geminiOAuthSvc, antigravityOAuthSvc,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)

	cleanupDone := make(chan struct{})
	go func() {
		cleanup()
		close(cleanupDone)
	}()
	select {
	case <-cleanupDone:
		t.Fatal("cleanup reached Ent teardown before the queued ops error drained")
	case <-time.After(75 * time.Millisecond):
	}

	release()
	select {
	case <-cleanupDone:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not finish after the queued ops error drained")
	}
	require.True(t, repo.persisted.Load())
	require.NoError(t, mock.ExpectationsWereMet())
}
