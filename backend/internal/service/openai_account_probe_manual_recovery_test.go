package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type manualRecoverySnapshotCacheStub struct {
	account *Account
}

func (c *manualRecoverySnapshotCacheStub) GetSnapshot(context.Context, SchedulerBucket) ([]*Account, bool, error) {
	return nil, false, nil
}

func (c *manualRecoverySnapshotCacheStub) SetSnapshot(context.Context, SchedulerBucket, []Account) error {
	return nil
}

func (c *manualRecoverySnapshotCacheStub) GetAccount(context.Context, int64) (*Account, error) {
	if c.account != nil {
		cloned := *c.account
		return &cloned, nil
	}
	return nil, errors.New("stale snapshot missing account")
}

func (c *manualRecoverySnapshotCacheStub) SetAccount(context.Context, *Account) error { return nil }

func (c *manualRecoverySnapshotCacheStub) DeleteAccount(context.Context, int64) error { return nil }

func (c *manualRecoverySnapshotCacheStub) UpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}

func (c *manualRecoverySnapshotCacheStub) TryLockBucket(context.Context, SchedulerBucket, time.Duration) (bool, error) {
	return true, nil
}

func (c *manualRecoverySnapshotCacheStub) UnlockBucket(context.Context, SchedulerBucket) error {
	return nil
}

func (c *manualRecoverySnapshotCacheStub) ListBuckets(context.Context) ([]SchedulerBucket, error) {
	return nil, nil
}

func (c *manualRecoverySnapshotCacheStub) GetOutboxWatermark(context.Context) (int64, error) {
	return 0, nil
}

func (c *manualRecoverySnapshotCacheStub) SetOutboxWatermark(context.Context, int64) error {
	return nil
}

type manualRecoveryProbeRepo struct {
	stubOpenAIAccountRepo
	until time.Time
}

func (r *manualRecoveryProbeRepo) ClearTempUnschedulable(context.Context, int64) error {
	return nil
}

func (r *manualRecoveryProbeRepo) SetTempUnschedulable(_ context.Context, _ int64, until time.Time, _ string) error {
	r.until = until
	return nil
}

func TestProbeManualRecoveryRunsImmediateProbe(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{
		resp: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\"}\n\n"))},
	}
	repo := &manualRecoveryProbeRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{{
		ID:          91,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"expires_at":   "2999-01-01T00:00:00Z",
		},
	}}}}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{
			accountRepo:       repo,
			httpUpstream:      upstream,
			schedulerSnapshot: nil,
		},
		stats:  newOpenAIAccountRuntimeStats(),
		ctx:    context.Background(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()
	entry := &openAIAccountProbeEntry{accountID: 91}
	entry.dbFlagSet.Store(true)
	entry.ttftPenalized.Store(true)

	probe.applyManualRecovery(91, entry)

	if upstream.lastReq == nil {
		t.Fatalf("manual recovery should run an immediate probe request")
	}
}

func TestProbeManualRecoveryImmediateProbeUsesDBWhenSnapshotIsStale(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{
		resp: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\"}\n\n"))},
	}
	repo := &manualRecoveryProbeRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{{
		ID:          93,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"expires_at":   "2999-01-01T00:00:00Z",
		},
	}}}}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{
			accountRepo:  repo,
			httpUpstream: upstream,
			schedulerSnapshot: NewSchedulerSnapshotService(&manualRecoverySnapshotCacheStub{account: &Account{
				ID:          93,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: false,
			}}, nil, repo, nil, nil),
		},
		stats:  newOpenAIAccountRuntimeStats(),
		ctx:    context.Background(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()
	entry := &openAIAccountProbeEntry{accountID: 93}
	entry.dbFlagSet.Store(true)

	probe.applyManualRecovery(93, entry)

	if upstream.lastReq == nil {
		t.Fatalf("manual recovery should bypass stale scheduler snapshot and run immediate probe with DB account")
	}
}

func TestProbeManualRecoveryReflagsWhenImmediateProbeFails(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{
		resp: &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("boom"))},
	}
	repo := &manualRecoveryProbeRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{{
		ID:          92,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"expires_at":   "2999-01-01T00:00:00Z",
		},
	}}}}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{
			accountRepo:  repo,
			httpUpstream: upstream,
		},
		stats:  newOpenAIAccountRuntimeStats(),
		ctx:    context.Background(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()
	entry := &openAIAccountProbeEntry{accountID: 92}
	entry.dbFlagSet.Store(true)

	probe.applyManualRecovery(92, entry)

	if repo.until.IsZero() {
		t.Fatalf("manual recovery should re-mark account temp unschedulable when immediate probe fails")
	}
}

func TestProbeManualRecoveryFailedImmediateProbeKeepsEntryForFutureProbes(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{
		resp: &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("boom"))},
	}
	repo := &manualRecoveryProbeRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{{
		ID:          94,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"expires_at":   "2999-01-01T00:00:00Z",
		},
	}}}}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{
			accountRepo:  repo,
			httpUpstream: upstream,
		},
		stats:  newOpenAIAccountRuntimeStats(),
		ctx:    context.Background(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()
	entry := &openAIAccountProbeEntry{accountID: 94}
	entry.dbFlagSet.Store(true)

	probe.applyManualRecovery(94, entry)

	stored, ok := probe.entries.Load(int64(94))
	if !ok || stored == nil {
		t.Fatalf("manual recovery should keep a probe entry after immediate probe failure")
	}
}

func TestOpenAIManualTempUnschedulableClearTriggersImmediateProbeRecovery(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{
		resp: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\"}\n\n"))},
	}
	repo := &manualRecoveryProbeRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{{
		ID:          95,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"expires_at":   "2999-01-01T00:00:00Z",
		},
	}}}}
	probe := &openAIAccountProbe{
		stats:  newOpenAIAccountRuntimeStats(),
		ctx:    context.Background(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()
	svc := &OpenAIGatewayService{
		accountRepo:     repo,
		httpUpstream:    upstream,
		openaiScheduler: &layeredOpenAIAccountScheduler{probe: probe},
	}
	probe.service = svc
	svc.openaiAccountRuntimeBlockUntil.Store(int64(95), time.Now().Add(time.Minute))

	svc.ClearAccountSchedulingBlock(95)

	if upstream.lastReq == nil {
		t.Fatalf("clearing scheduling block should trigger immediate layered probe recovery")
	}
}

func TestOpenAIManualTempUnschedulableClearWithoutLayeredProbeIsNoop(t *testing.T) {
	svc := &OpenAIGatewayService{}
	svc.openaiAccountRuntimeBlockUntil.Store(int64(96), time.Now().Add(time.Minute))

	svc.ClearAccountSchedulingBlock(96)

	if _, ok := svc.openaiAccountRuntimeBlockUntil.Load(int64(96)); ok {
		t.Fatalf("clearing scheduling block should still clear runtime block when no layered probe is configured")
	}
}

func TestOpenAIManualTempUnschedulableClearWithDefaultSchedulerDoesNotTriggerProbe(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{
		resp: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\"}\n\n"))},
	}
	svc := &OpenAIGatewayService{
		httpUpstream:    upstream,
		openaiScheduler: &defaultOpenAIAccountScheduler{},
	}
	svc.openaiAccountRuntimeBlockUntil.Store(int64(97), time.Now().Add(time.Minute))

	svc.ClearAccountSchedulingBlock(97)

	if upstream.lastReq != nil {
		t.Fatalf("clearing scheduling block should not trigger probe when scheduler is not layered")
	}
	if _, ok := svc.openaiAccountRuntimeBlockUntil.Load(int64(97)); ok {
		t.Fatalf("clearing scheduling block should still clear runtime block for non-layered scheduler")
	}
}
