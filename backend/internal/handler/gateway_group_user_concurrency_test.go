package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type gatewayUserGroupConcurrencyCacheMock struct {
	*concurrencyCacheMock
	acquireUserGroupSlotFn func(ctx context.Context, userID, groupID int64, maxConcurrency int, requestID string) (bool, error)
	releaseUserGroupCalled int32
}

func (m *gatewayUserGroupConcurrencyCacheMock) AcquireUserGroupSlot(ctx context.Context, userID, groupID int64, maxConcurrency int, requestID string) (bool, error) {
	if m.acquireUserGroupSlotFn != nil {
		return m.acquireUserGroupSlotFn(ctx, userID, groupID, maxConcurrency, requestID)
	}
	return m.concurrencyCacheMock.AcquireUserGroupSlot(ctx, userID, groupID, maxConcurrency, requestID)
}

func (m *gatewayUserGroupConcurrencyCacheMock) ReleaseUserGroupSlot(ctx context.Context, userID, groupID int64, requestID string) error {
	atomic.AddInt32(&m.releaseUserGroupCalled, 1)
	return nil
}

func newGatewayHandlerForUserGroupConcurrencyTest(cache service.ConcurrencyCache) *GatewayHandler {
	return &GatewayHandler{
		concurrencyHelper: NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatNone, time.Second),
	}
}

func newUserGroupConcurrencyHelperForTest(cache service.ConcurrencyCache) *ConcurrencyHelper {
	return NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatNone, time.Second)
}

func newUserGroupConcurrencyTestContext(t *testing.T) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
	return c
}

func TestGatewayHandlerAcquireUserGroupSlot_SkipsWhenGroupDisabled(t *testing.T) {
	cache := &gatewayUserGroupConcurrencyCacheMock{concurrencyCacheMock: &concurrencyCacheMock{}}
	h := newGatewayHandlerForUserGroupConcurrencyTest(cache)
	c := newUserGroupConcurrencyTestContext(t)
	streamStarted := false
	groupID := int64(11)
	group := &service.Group{UserConcurrencyEnabled: false, UserConcurrencyLimit: 2}

	releaseFunc, ok := h.acquireUserGroupSlot(c, h.concurrencyHelper, 101, &groupID, group, false, &streamStarted, zap.NewNop())

	require.True(t, ok)
	require.Nil(t, releaseFunc)
	require.Equal(t, int32(0), atomic.LoadInt32(&cache.releaseUserGroupCalled))
}

func TestGatewayHandlerAcquireUserGroupSlot_AcquiresWhenGroupEnabled(t *testing.T) {
	var acquireCalls int32
	cache := &gatewayUserGroupConcurrencyCacheMock{
		concurrencyCacheMock: &concurrencyCacheMock{},
		acquireUserGroupSlotFn: func(ctx context.Context, userID, groupID int64, maxConcurrency int, requestID string) (bool, error) {
			atomic.AddInt32(&acquireCalls, 1)
			return true, nil
		},
	}
	h := newGatewayHandlerForUserGroupConcurrencyTest(cache)
	c := newUserGroupConcurrencyTestContext(t)
	streamStarted := false
	groupID := int64(12)
	group := &service.Group{UserConcurrencyEnabled: true, UserConcurrencyLimit: 2}

	releaseFunc, ok := h.acquireUserGroupSlot(c, h.concurrencyHelper, 202, &groupID, group, false, &streamStarted, zap.NewNop())

	require.True(t, ok)
	require.NotNil(t, releaseFunc)
	require.Equal(t, int32(1), atomic.LoadInt32(&acquireCalls))
	releaseFunc()
	require.Equal(t, int32(1), atomic.LoadInt32(&cache.releaseUserGroupCalled))
}

func TestGatewayHandlerAcquireUserGroupSlot_SkipsWhenGroupIDNil(t *testing.T) {
	cache := &gatewayUserGroupConcurrencyCacheMock{concurrencyCacheMock: &concurrencyCacheMock{}}
	h := newGatewayHandlerForUserGroupConcurrencyTest(cache)
	c := newUserGroupConcurrencyTestContext(t)
	streamStarted := false
	group := &service.Group{UserConcurrencyEnabled: true, UserConcurrencyLimit: 2}

	releaseFunc, ok := h.acquireUserGroupSlot(c, h.concurrencyHelper, 303, nil, group, false, &streamStarted, zap.NewNop())

	require.True(t, ok)
	require.Nil(t, releaseFunc)
	require.Equal(t, int32(0), atomic.LoadInt32(&cache.releaseUserGroupCalled))
}

func TestGatewayHandlerAcquireUserGroupSlot_SkipsWhenGroupNil(t *testing.T) {
	cache := &gatewayUserGroupConcurrencyCacheMock{concurrencyCacheMock: &concurrencyCacheMock{}}
	h := newGatewayHandlerForUserGroupConcurrencyTest(cache)
	c := newUserGroupConcurrencyTestContext(t)
	streamStarted := false
	groupID := int64(13)

	releaseFunc, ok := h.acquireUserGroupSlot(c, h.concurrencyHelper, 404, &groupID, nil, false, &streamStarted, zap.NewNop())

	require.True(t, ok)
	require.Nil(t, releaseFunc)
	require.Equal(t, int32(0), atomic.LoadInt32(&cache.releaseUserGroupCalled))
}

func TestGatewayHandlerAcquireUserGroupSlot_SkipsWhenLimitNonPositive(t *testing.T) {
	cache := &gatewayUserGroupConcurrencyCacheMock{concurrencyCacheMock: &concurrencyCacheMock{}}
	h := newGatewayHandlerForUserGroupConcurrencyTest(cache)
	c := newUserGroupConcurrencyTestContext(t)
	streamStarted := false
	groupID := int64(14)
	group := &service.Group{UserConcurrencyEnabled: true, UserConcurrencyLimit: 0}

	releaseFunc, ok := h.acquireUserGroupSlot(c, h.concurrencyHelper, 505, &groupID, group, false, &streamStarted, zap.NewNop())

	require.True(t, ok)
	require.Nil(t, releaseFunc)
	require.Equal(t, int32(0), atomic.LoadInt32(&cache.releaseUserGroupCalled))
}

func TestGatewayHandlerAcquireUserGroupSlot_ReturnsFalseOnTryAcquireError(t *testing.T) {
	wantErr := errors.New("boom")
	cache := &gatewayUserGroupConcurrencyCacheMock{
		concurrencyCacheMock: &concurrencyCacheMock{},
		acquireUserGroupSlotFn: func(ctx context.Context, userID, groupID int64, maxConcurrency int, requestID string) (bool, error) {
			return false, wantErr
		},
	}
	h := newGatewayHandlerForUserGroupConcurrencyTest(cache)
	c := newUserGroupConcurrencyTestContext(t)
	streamStarted := false
	groupID := int64(15)
	group := &service.Group{UserConcurrencyEnabled: true, UserConcurrencyLimit: 2}

	releaseFunc, ok := h.acquireUserGroupSlot(c, h.concurrencyHelper, 606, &groupID, group, false, &streamStarted, zap.NewNop())

	require.False(t, ok)
	require.Nil(t, releaseFunc)
	require.Equal(t, http.StatusTooManyRequests, c.Writer.Status())
}

func TestGatewayHandlerAcquireUserGroupSlot_WaitsAfterTryAcquireMiss(t *testing.T) {
	var acquireCalls int32
	cache := &gatewayUserGroupConcurrencyCacheMock{
		concurrencyCacheMock: &concurrencyCacheMock{},
		acquireUserGroupSlotFn: func(ctx context.Context, userID, groupID int64, maxConcurrency int, requestID string) (bool, error) {
			call := atomic.AddInt32(&acquireCalls, 1)
			if call == 1 {
				return false, nil
			}
			return true, nil
		},
	}
	h := newGatewayHandlerForUserGroupConcurrencyTest(cache)
	c := newUserGroupConcurrencyTestContext(t)
	streamStarted := false
	groupID := int64(16)
	group := &service.Group{UserConcurrencyEnabled: true, UserConcurrencyLimit: 2}

	releaseFunc, ok := h.acquireUserGroupSlot(c, newUserGroupConcurrencyHelperForTest(cache), 707, &groupID, group, false, &streamStarted, zap.NewNop())

	require.True(t, ok)
	require.NotNil(t, releaseFunc)
	require.GreaterOrEqual(t, atomic.LoadInt32(&acquireCalls), int32(2))
	releaseFunc()
	require.Equal(t, int32(1), atomic.LoadInt32(&cache.releaseUserGroupCalled))
}

func TestGatewayHandlerAcquireUserGroupSlot_AllowsNilStreamStarted(t *testing.T) {
	cache := &gatewayUserGroupConcurrencyCacheMock{
		concurrencyCacheMock: &concurrencyCacheMock{},
		acquireUserGroupSlotFn: func(ctx context.Context, userID, groupID int64, maxConcurrency int, requestID string) (bool, error) {
			return true, nil
		},
	}
	h := newGatewayHandlerForUserGroupConcurrencyTest(cache)
	c := newUserGroupConcurrencyTestContext(t)
	groupID := int64(17)
	group := &service.Group{UserConcurrencyEnabled: true, UserConcurrencyLimit: 2}

	var (
		releaseFunc func()
		ok          bool
	)
	require.NotPanics(t, func() {
		releaseFunc, ok = h.acquireUserGroupSlot(c, h.concurrencyHelper, 808, &groupID, group, false, nil, zap.NewNop())
	})

	require.True(t, ok)
	require.NotNil(t, releaseFunc)
	releaseFunc()
	require.Equal(t, int32(1), atomic.LoadInt32(&cache.releaseUserGroupCalled))
}

func TestGatewayHandlerAcquireUserGroupSlot_ReturnsFalseWhenWaitFailsAfterTryAcquireMiss(t *testing.T) {
	var acquireCalls int32
	wantErr := errors.New("wait failed")
	cache := &gatewayUserGroupConcurrencyCacheMock{
		concurrencyCacheMock: &concurrencyCacheMock{},
		acquireUserGroupSlotFn: func(ctx context.Context, userID, groupID int64, maxConcurrency int, requestID string) (bool, error) {
			call := atomic.AddInt32(&acquireCalls, 1)
			switch call {
			case 1, 2:
				return false, nil
			default:
				return false, wantErr
			}
		},
	}
	h := newGatewayHandlerForUserGroupConcurrencyTest(cache)
	c := newUserGroupConcurrencyTestContext(t)
	streamStarted := false
	groupID := int64(18)
	group := &service.Group{UserConcurrencyEnabled: true, UserConcurrencyLimit: 2}

	releaseFunc, ok := h.acquireUserGroupSlot(c, newUserGroupConcurrencyHelperForTest(cache), 909, &groupID, group, false, &streamStarted, zap.NewNop())

	require.False(t, ok)
	require.Nil(t, releaseFunc)
	require.GreaterOrEqual(t, atomic.LoadInt32(&acquireCalls), int32(3))
	require.Equal(t, http.StatusTooManyRequests, c.Writer.Status())
}
