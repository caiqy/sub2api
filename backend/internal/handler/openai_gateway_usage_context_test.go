package handler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type usageRecordTrackingContext struct {
	context.Context
	valueReads atomic.Int32
}

func (c *usageRecordTrackingContext) Value(key any) any {
	c.valueReads.Add(1)
	return c.Context.Value(key)
}

func TestSubmitUsageRecordTaskCopiesRequestContext(t *testing.T) {
	parent := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-request-123")
	parent = context.WithValue(parent, ctxkey.RequestID, "request-456")

	var gotClientRequestID string
	var gotRequestID string
	h := &GatewayHandler{}
	h.submitUsageRecordTask(parent, func(ctx context.Context) {
		gotClientRequestID, _ = ctx.Value(ctxkey.ClientRequestID).(string)
		gotRequestID, _ = ctx.Value(ctxkey.RequestID).(string)
	})

	require.Equal(t, "client-request-123", gotClientRequestID)
	require.Equal(t, "request-456", gotRequestID)
}

func TestOpenAISubmitUsageRecordTaskCopiesRequestContext(t *testing.T) {
	parent := context.WithValue(context.Background(), ctxkey.ClientRequestID, "openai-client-request-123")
	parent = context.WithValue(parent, ctxkey.RequestID, "openai-request-456")

	var gotClientRequestID string
	var gotRequestID string
	h := &OpenAIGatewayHandler{}
	h.submitUsageRecordTask(parent, func(ctx context.Context) {
		gotClientRequestID, _ = ctx.Value(ctxkey.ClientRequestID).(string)
		gotRequestID, _ = ctx.Value(ctxkey.RequestID).(string)
	})

	require.Equal(t, "openai-client-request-123", gotClientRequestID)
	require.Equal(t, "openai-request-456", gotRequestID)
}

func TestOpenAIUsageRecordTaskCopiesCompositeBillingContextAfterQueueDelay(t *testing.T) {
	pool := newUsageRecordTestPool(t)
	block := make(chan struct{})
	started := make(chan struct{})
	pool.Submit(func(context.Context) {
		close(started)
		<-block
	})
	<-started

	groupID := int64(7)
	parent := context.WithValue(context.Background(), ctxkey.ForcePlatform, service.PlatformOpenAI)
	parent = service.WithCompositeRouteDecision(parent, service.CompositeRouteDecision{
		Matched:        true,
		GroupID:        groupID,
		PublicModel:    "public-alias",
		TargetPlatform: service.PlatformOpenAI,
		UpstreamModel:  "gpt-5",
	})
	h := &OpenAIGatewayHandler{usageRecordWorkerPool: pool}
	got := make(chan struct {
		decision service.CompositeRouteDecision
		quota    string
	}, 1)
	h.submitUsageRecordTask(parent, func(ctx context.Context) {
		decision, _ := service.CompositeRouteDecisionFromContext(ctx)
		quota := service.QuotaPlatform(ctx, &service.APIKey{Group: &service.Group{Platform: service.PlatformComposite}})
		got <- struct {
			decision service.CompositeRouteDecision
			quota    string
		}{decision: decision, quota: quota}
	})

	close(block)
	select {
	case result := <-got:
		require.Equal(t, service.PlatformOpenAI, result.quota)
		require.Equal(t, groupID, result.decision.GroupID)
		require.Equal(t, "gpt-5", result.decision.UpstreamModel)
	case <-time.After(time.Second):
		t.Fatal("queued usage task did not execute")
	}
}

func TestUsageRecordTaskSnapshotsParentContextBeforeQueueDelay(t *testing.T) {
	pool := newUsageRecordTestPool(t)
	block := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(block) })
	}
	t.Cleanup(release)
	started := make(chan struct{})
	pool.Submit(func(context.Context) {
		close(started)
		<-block
	})
	<-started

	parent := &usageRecordTrackingContext{Context: context.WithValue(context.Background(), ctxkey.ForcePlatform, service.PlatformOpenAI)}
	done := make(chan struct{})
	h := &GatewayHandler{usageRecordWorkerPool: pool}
	h.submitUsageRecordTask(parent, func(context.Context) {
		close(done)
	})

	readsBeforeWorker := parent.valueReads.Load()
	require.NotZero(t, readsBeforeWorker)
	release()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("queued usage task did not execute")
	}
	require.Equal(t, readsBeforeWorker, parent.valueReads.Load())
}
