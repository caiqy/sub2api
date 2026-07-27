package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type deferredBlockingAccountRepository struct {
	AccountRepository
	started sync.Once
	release chan struct{}
	onStart chan struct{}
}

func (r *deferredBlockingAccountRepository) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	r.started.Do(func() { close(r.onStart) })
	<-r.release
	return nil
}

func TestDeferredServiceStopWaitsForInFlightFlush(t *testing.T) {
	timingWheel, err := NewTimingWheelService()
	require.NoError(t, err)
	defer timingWheel.Stop()

	repo := &deferredBlockingAccountRepository{release: make(chan struct{}), onStart: make(chan struct{})}
	svc := NewDeferredService(repo, timingWheel, time.Hour)
	svc.ScheduleLastUsedUpdate(1)

	flushDone := make(chan struct{})
	go func() {
		defer close(flushDone)
		svc.flushLastUsed(context.Background())
	}()
	<-repo.onStart

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- svc.Stop(context.Background())
	}()

	select {
	case err := <-stopDone:
		t.Fatalf("Stop returned before the in-flight flush completed: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(repo.release)
	<-flushDone
	require.NoError(t, <-stopDone)
}
