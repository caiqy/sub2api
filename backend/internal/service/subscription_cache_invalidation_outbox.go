package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	subscriptionInvalidationBatchSize    = 100
	subscriptionInvalidationPollInterval = 500 * time.Millisecond
	subscriptionInvalidationLease        = 30 * time.Second
	subscriptionInvalidationRedisTimeout = 2 * time.Second
	subscriptionInvalidationSafetyDelay  = 30 * time.Second
	subscriptionInvalidationConcurrency  = 16
)

type SubscriptionCacheInvalidationEvent struct {
	ID       int64
	UserID   int64
	GroupID  int64
	Version  int64
	Attempts int
	Stage    int
}

type SubscriptionCacheInvalidationOutboxRepository interface {
	Claim(ctx context.Context, workerID string, limit int, lease time.Duration) ([]SubscriptionCacheInvalidationEvent, error)
	ScheduleSecondPass(ctx context.Context, id int64, workerID string, availableAt time.Time) error
	DeleteClaimed(ctx context.Context, id int64, workerID string) error
	RetryClaimed(ctx context.Context, id int64, workerID string, availableAt time.Time, lastError string) error
}

type subscriptionCacheInvalidationStore interface {
	InvalidateSubscriptionVersioned(ctx context.Context, userID, groupID, version int64) error
	PublishSubscriptionCacheInvalidation(ctx context.Context, cacheKey string) error
}

type SubscriptionCacheInvalidationWorker struct {
	repo     SubscriptionCacheInvalidationOutboxRepository
	cache    subscriptionCacheInvalidationStore
	workerID string
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	start    sync.Once
	stop     sync.Once
}

func NewSubscriptionCacheInvalidationWorker(repo SubscriptionCacheInvalidationOutboxRepository, cache subscriptionCacheInvalidationStore) *SubscriptionCacheInvalidationWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &SubscriptionCacheInvalidationWorker{repo: repo, cache: cache, workerID: uuid.NewString(), ctx: ctx, cancel: cancel}
}

func ProvideSubscriptionCacheInvalidationWorker(repo SubscriptionCacheInvalidationOutboxRepository, cache *BillingCacheService) *SubscriptionCacheInvalidationWorker {
	worker := NewSubscriptionCacheInvalidationWorker(repo, cache)
	worker.Start()
	return worker
}

func (w *SubscriptionCacheInvalidationWorker) Start() {
	if w == nil || w.repo == nil || w.cache == nil {
		return
	}
	w.start.Do(func() {
		w.wg.Add(1)
		go w.run()
	})
}

func (w *SubscriptionCacheInvalidationWorker) Stop() {
	if w == nil {
		return
	}
	w.stop.Do(func() {
		w.cancel()
		w.wg.Wait()
	})
}

func (w *SubscriptionCacheInvalidationWorker) run() {
	defer w.wg.Done()
	ticker := time.NewTicker(subscriptionInvalidationPollInterval)
	defer ticker.Stop()
	for {
		if err := w.processBatch(w.ctx); err != nil && w.ctx.Err() == nil {
			slog.Warn("subscription cache invalidation outbox processing failed", "error", err)
		}
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *SubscriptionCacheInvalidationWorker) processBatch(ctx context.Context) error {
	events, err := w.repo.Claim(ctx, w.workerID, subscriptionInvalidationBatchSize, subscriptionInvalidationLease)
	if err != nil {
		return fmt.Errorf("claim subscription cache invalidations: %w", err)
	}
	semaphore := make(chan struct{}, subscriptionInvalidationConcurrency)
	var wg sync.WaitGroup
	for i := range events {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case semaphore <- struct{}{}:
		}
		wg.Add(1)
		go func(event SubscriptionCacheInvalidationEvent) {
			defer wg.Done()
			defer func() { <-semaphore }()
			w.processEvent(ctx, event)
		}(events[i])
	}
	wg.Wait()
	return nil
}

func (w *SubscriptionCacheInvalidationWorker) processEvent(parent context.Context, event SubscriptionCacheInvalidationEvent) {
	tombstoneCtx, cancelTombstone := context.WithTimeout(parent, subscriptionInvalidationRedisTimeout)
	tombstoneErr := w.cache.InvalidateSubscriptionVersioned(tombstoneCtx, event.UserID, event.GroupID, event.Version)
	cancelTombstone()
	publishCtx, cancelPublish := context.WithTimeout(parent, subscriptionInvalidationRedisTimeout)
	publishErr := w.cache.PublishSubscriptionCacheInvalidation(publishCtx, subCacheKey(event.UserID, event.GroupID))
	cancelPublish()
	if err := errors.Join(tombstoneErr, publishErr); err != nil {
		w.retry(event, err)
		return
	}
	if event.Stage == 0 {
		nextCtx, nextCancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := w.repo.ScheduleSecondPass(nextCtx, event.ID, w.workerID, time.Now().UTC().Add(subscriptionInvalidationSafetyDelay))
		nextCancel()
		if err != nil {
			slog.Warn("schedule second subscription cache invalidation pass", "event_id", event.ID, "error", err)
		}
		return
	}
	ackCtx, ackCancel := context.WithTimeout(context.Background(), 2*time.Second)
	err := w.repo.DeleteClaimed(ackCtx, event.ID, w.workerID)
	ackCancel()
	if err != nil {
		slog.Warn("ack subscription cache invalidation", "event_id", event.ID, "error", err)
	}
}

func (w *SubscriptionCacheInvalidationWorker) retry(event SubscriptionCacheInvalidationEvent, err error) {
	retryCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	retryErr := w.repo.RetryClaimed(retryCtx, event.ID, w.workerID, time.Now().UTC().Add(authInvalidationRetryDelay(event.Attempts+1)), boundedAuthInvalidationError(err))
	cancel()
	if retryErr != nil {
		slog.Warn("release failed subscription cache invalidation", "event_id", event.ID, "error", retryErr)
	}
}
