package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// IdempotencyCleanupService 定期清理已过期的幂等记录，避免表无限增长。
type IdempotencyCleanupService struct {
	repo        IdempotencyRepository
	receiptRepo QuotaAdvanceReceiptRepository
	interval    time.Duration
	batch       int

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
}

func NewIdempotencyCleanupService(repo IdempotencyRepository, cfg *config.Config, receiptRepos ...QuotaAdvanceReceiptRepository) *IdempotencyCleanupService {
	interval := 60 * time.Second
	batch := 500
	if cfg != nil {
		if cfg.Idempotency.CleanupIntervalSeconds > 0 {
			interval = time.Duration(cfg.Idempotency.CleanupIntervalSeconds) * time.Second
		}
		if cfg.Idempotency.CleanupBatchSize > 0 {
			batch = cfg.Idempotency.CleanupBatchSize
		}
	}
	svc := &IdempotencyCleanupService{
		repo:     repo,
		interval: interval,
		batch:    batch,
		stopCh:   make(chan struct{}),
	}
	if len(receiptRepos) > 0 {
		svc.receiptRepo = receiptRepos[0]
	}
	return svc
}

func (s *IdempotencyCleanupService) Start() {
	if s == nil || s.repo == nil {
		return
	}
	s.startOnce.Do(func() {
		logger.LegacyPrintf("service.idempotency_cleanup", "[IdempotencyCleanup] started interval=%s batch=%d", s.interval, s.batch)
		go s.runLoop()
	})
}

func (s *IdempotencyCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		logger.LegacyPrintf("service.idempotency_cleanup", "[IdempotencyCleanup] stopped")
	})
}

func (s *IdempotencyCleanupService) runLoop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// 启动后先清理一轮，防止重启后积压。
	s.cleanupOnce()

	for {
		select {
		case <-ticker.C:
			s.cleanupOnce()
		case <-s.stopCh:
			return
		}
	}
}

func (s *IdempotencyCleanupService) cleanupOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()
	deleted, err := s.repo.DeleteExpired(ctx, now, s.batch)
	if err != nil {
		logger.LegacyPrintf("service.idempotency_cleanup", "[IdempotencyCleanup] cleanup failed err=%v", err)
	} else if deleted > 0 {
		logger.LegacyPrintf("service.idempotency_cleanup", "[IdempotencyCleanup] cleaned expired records count=%d", deleted)
	}
	if s.receiptRepo == nil {
		return
	}
	deleted, err = s.receiptRepo.DeleteExpired(ctx, now, s.batch)
	if err != nil {
		logger.LegacyPrintf("service.idempotency_cleanup", "[IdempotencyCleanup] receipt cleanup failed err=%v", err)
	} else if deleted > 0 {
		logger.LegacyPrintf("service.idempotency_cleanup", "[IdempotencyCleanup] cleaned expired receipt count=%d", deleted)
	}
}
