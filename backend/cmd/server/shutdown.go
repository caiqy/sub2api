package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

type activeHandlerTracker struct {
	mu       sync.Mutex
	closed   bool
	handlers sync.WaitGroup
}

func newActiveHandlerTracker() *activeHandlerTracker {
	return &activeHandlerTracker{}
}

func (t *activeHandlerTracker) Wrap(next http.Handler) http.Handler {
	if next == nil {
		next = http.DefaultServeMux
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.mu.Lock()
		if t.closed {
			t.mu.Unlock()
			http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
			return
		}
		t.handlers.Add(1)
		t.mu.Unlock()
		defer t.handlers.Done()
		next.ServeHTTP(w, r)
	})
}

func (t *activeHandlerTracker) CloseAdmission() {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()
}

func (t *activeHandlerTracker) Wait(ctx context.Context) bool {
	done := make(chan struct{})
	go func() {
		t.handlers.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

type cleanupPhase struct {
	name string
	run  func(context.Context) error
}

func runCleanupPhases(ctx context.Context, phases ...cleanupPhase) bool {
	for _, phase := range phases {
		if phase.run == nil {
			continue
		}
		if ctx.Err() != nil {
			log.Printf("[Cleanup] %s skipped: %v", phase.name, ctx.Err())
			return false
		}
		done := make(chan error, 1)
		go func(phase cleanupPhase) {
			done <- phase.run(ctx)
		}(phase)
		select {
		case err := <-done:
			if err != nil {
				log.Printf("[Cleanup] %s failed: %v", phase.name, err)
				return false
			}
			log.Printf("[Cleanup] %s succeeded", phase.name)
			if ctx.Err() != nil {
				log.Printf("[Cleanup] stopped before downstream teardown: %v", ctx.Err())
				return false
			}
		case <-ctx.Done():
			log.Printf("[Cleanup] %s timed out; downstream teardown skipped: %v", phase.name, ctx.Err())
			return false
		}
	}
	return true
}

func runCleanupParallel(ctx context.Context, steps ...cleanupPhase) error {
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var errs []error
	for _, step := range steps {
		if step.run == nil {
			continue
		}
		wg.Add(1)
		go func(step cleanupPhase) {
			defer wg.Done()
			if err := step.run(ctx); err != nil {
				log.Printf("[Cleanup] %s failed: %v", step.name, err)
				errMu.Lock()
				errs = append(errs, err)
				errMu.Unlock()
			}
		}(step)
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return errors.Join(errs...)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func shutdownServerWithDrain(
	ctx context.Context,
	gracefulTimeout time.Duration,
	shutdown func(context.Context) error,
	closeServer func() error,
	waitHandlers func(context.Context) bool,
) error {
	gracefulCtx, cancel := context.WithTimeout(ctx, gracefulTimeout)
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- shutdown(gracefulCtx)
	}()

	var shutdownErr error
	select {
	case shutdownErr = <-shutdownDone:
	case <-gracefulCtx.Done():
		shutdownErr = gracefulCtx.Err()
	}
	cancel()

	if shutdownErr != nil && closeServer != nil {
		if err := closeServer(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server force-close failed: %v", err)
		}
	}
	if waitHandlers != nil && !waitHandlers(ctx) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return shutdownErr
}
