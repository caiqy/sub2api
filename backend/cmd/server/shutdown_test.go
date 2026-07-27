package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunCleanupPhasesOrdersDependentDrains(t *testing.T) {
	var mu sync.Mutex
	var order []string
	add := func(name string) func(context.Context) error {
		return func(context.Context) error {
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			return nil
		}
	}

	completed := runCleanupPhases(context.Background(),
		cleanupPhase{name: "producers", run: add("producers")},
		cleanupPhase{name: "usage", run: add("usage")},
		cleanupPhase{name: "quota", run: add("quota")},
		cleanupPhase{name: "billing", run: add("billing")},
		cleanupPhase{name: "infra", run: add("infra")},
	)

	require.True(t, completed)
	require.Equal(t, []string{"producers", "usage", "quota", "billing", "infra"}, order)
}

func TestActiveHandlerTrackerRejectsLateAdmissionAfterClose(t *testing.T) {
	tracker := newActiveHandlerTracker()
	called := false
	wrapped := tracker.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	tracker.CloseAdmission()
	response := httptest.NewRecorder()
	wrapped.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.False(t, called)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.True(t, tracker.Wait(ctx))
}

func TestRunCleanupParallelReturnsChildErrors(t *testing.T) {
	want := errors.New("producer failed")
	err := runCleanupParallel(context.Background(),
		cleanupPhase{name: "failed-producer", run: func(context.Context) error { return want }},
		cleanupPhase{name: "healthy-producer", run: func(context.Context) error { return nil }},
	)

	require.ErrorIs(t, err, want)
}

func TestRunCleanupPhasesStopsAfterProducerFailure(t *testing.T) {
	want := errors.New("producer failed")
	var order []string

	completed := runCleanupPhases(context.Background(),
		cleanupPhase{name: "producers", run: func(ctx context.Context) error {
			order = append(order, "producers")
			return runCleanupParallel(ctx, cleanupPhase{name: "failed", run: func(context.Context) error { return want }})
		}},
		cleanupPhase{name: "usage-record-drain", run: func(context.Context) error {
			order = append(order, "usage")
			return nil
		}},
		cleanupPhase{name: "deferred-last-used-flush", run: func(context.Context) error {
			order = append(order, "deferred")
			return nil
		}},
	)

	require.False(t, completed)
	require.Equal(t, []string{"producers"}, order)
}

func TestProvideCleanupOrdersProducerUsageAndDeferredDrains(t *testing.T) {
	source, err := os.ReadFile("wire.go")
	require.NoError(t, err)

	producers := bytes.Index(source, []byte(`name: "producers"`))
	imageTasks := bytes.Index(source, []byte(`name: "ImageTaskService"`))
	usage := bytes.Index(source, []byte(`name: "usage-record-drain"`))
	deferred := bytes.Index(source, []byte(`name: "deferred-last-used-flush"`))
	require.GreaterOrEqual(t, producers, 0)
	require.GreaterOrEqual(t, imageTasks, 0)
	require.GreaterOrEqual(t, usage, 0)
	require.GreaterOrEqual(t, deferred, 0)
	require.Less(t, producers, usage)
	require.Less(t, imageTasks, usage)
	require.Less(t, usage, deferred)
}

func TestRunCleanupPhasesStopsAfterDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	var order []string
	add := func(name string) {
		mu.Lock()
		order = append(order, name)
		mu.Unlock()
	}

	completed := runCleanupPhases(ctx,
		cleanupPhase{name: "usage", run: func(context.Context) error {
			add("usage")
			close(started)
			<-release
			return nil
		}},
		cleanupPhase{name: "quota", run: func(context.Context) error {
			add("quota")
			return nil
		}},
	)
	<-started
	require.False(t, completed)
	mu.Lock()
	require.Equal(t, []string{"usage"}, order)
	mu.Unlock()
	close(release)
}

func TestShutdownServerWithDrainForceClosesBeforeCleanupAfterActiveHandlerFinishes(t *testing.T) {
	tracker := newActiveHandlerTracker()
	handlerStarted := make(chan struct{})
	allowSideEffect := make(chan struct{})
	sideEffectCommitted := make(chan struct{}, 1)
	handlerExited := make(chan struct{})
	wrapped := tracker.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(handlerStarted)
		<-allowSideEffect
		sideEffectCommitted <- struct{}{}
		close(handlerExited)
	}))
	go wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	<-handlerStarted

	var mu sync.Mutex
	var order []string
	appendOrder := func(name string) {
		mu.Lock()
		order = append(order, name)
		mu.Unlock()
	}
	closed := make(chan struct{})
	hardCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- shutdownServerWithDrain(
			hardCtx,
			10*time.Millisecond,
			func(ctx context.Context) error {
				appendOrder("shutdown")
				<-ctx.Done()
				return ctx.Err()
			},
			func() error {
				appendOrder("close")
				close(closed)
				return nil
			},
			func(ctx context.Context) bool {
				appendOrder("drain")
				return tracker.Wait(ctx)
			},
		)
	}()

	<-closed
	select {
	case err := <-done:
		t.Fatalf("shutdown returned before active handler committed side effect: %v", err)
	default:
	}
	close(allowSideEffect)
	<-sideEffectCommitted
	<-handlerExited
	require.ErrorIs(t, <-done, context.DeadlineExceeded)
	mu.Lock()
	require.Equal(t, []string{"shutdown", "close", "drain"}, order)
	mu.Unlock()
}

func TestShutdownServerWithDrainReturnsAtHardDeadline(t *testing.T) {
	tracker := newActiveHandlerTracker()
	handlerStarted := make(chan struct{})
	release := make(chan struct{})
	handlerExited := make(chan struct{})
	wrapped := tracker.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(handlerStarted)
		<-release
		close(handlerExited)
	}))
	go wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	<-handlerStarted

	hardCtx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()
	err := shutdownServerWithDrain(
		hardCtx,
		5*time.Millisecond,
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		func() error { return nil },
		tracker.Wait,
	)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(started), 250*time.Millisecond)

	close(release)
	<-handlerExited
}
