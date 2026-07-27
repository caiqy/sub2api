package service

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// DeferredService provides deferred batch update functionality
type DeferredService struct {
	accountRepo AccountRepository
	timingWheel *TimingWheelService
	interval    time.Duration

	lastUsedUpdates sync.Map
	flushMu         sync.Mutex
	stopped         atomic.Bool
}

// NewDeferredService creates a new DeferredService instance
func NewDeferredService(accountRepo AccountRepository, timingWheel *TimingWheelService, interval time.Duration) *DeferredService {
	return &DeferredService{
		accountRepo: accountRepo,
		timingWheel: timingWheel,
		interval:    interval,
	}
}

// Start starts the deferred service
func (s *DeferredService) Start() {
	if s == nil || s.timingWheel == nil || s.stopped.Load() {
		return
	}
	s.timingWheel.ScheduleRecurring("deferred:last_used", s.interval, s.tick)
	log.Printf("[DeferredService] Started (interval: %v)", s.interval)
}

// Stop stops the deferred service
func (s *DeferredService) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.stopped.Store(true)
	if s.timingWheel != nil {
		s.timingWheel.Cancel("deferred:last_used")
	}
	if err := s.flushLastUsed(ctx); err != nil {
		return err
	}
	log.Printf("[DeferredService] Service stopped")
	return nil
}

func (s *DeferredService) ScheduleLastUsedUpdate(accountID int64) {
	if s == nil || s.stopped.Load() {
		return
	}
	s.lastUsedUpdates.Store(accountID, time.Now())
}

func (s *DeferredService) tick() {
	if s == nil || s.stopped.Load() {
		return
	}
	if err := s.flushLastUsed(context.Background()); err != nil {
		log.Printf("[DeferredService] BatchUpdateLastUsed failed: %v", err)
	}
}

func (s *DeferredService) flushLastUsed(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	updates := make(map[int64]time.Time)
	s.lastUsedUpdates.Range(func(key, value any) bool {
		id, ok := key.(int64)
		if !ok {
			return true
		}
		ts, ok := value.(time.Time)
		if !ok {
			return true
		}
		updates[id] = ts
		s.lastUsedUpdates.Delete(key)
		return true
	})

	if len(updates) == 0 {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.accountRepo.BatchUpdateLastUsed(ctx, updates); err != nil {
		for id, ts := range updates {
			s.lastUsedUpdates.Store(id, ts)
		}
		return err
	} else {
		log.Printf("[DeferredService] BatchUpdateLastUsed flushed %d accounts", len(updates))
	}
	return nil
}
