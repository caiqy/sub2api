package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

func (c *manualRecoverySnapshotCacheStub) GetAccount(_ context.Context, _ int64) (*Account, error) {
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
	clearCalls int
	setCalls   int
}

func (r *manualRecoveryProbeRepo) ClearTempUnschedulable(context.Context, int64) error {
	r.clearCalls++
	return nil
}

func (r *manualRecoveryProbeRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	r.setCalls++
	return nil
}

func TestProbeManualRecoveryDoesNotSendImmediateProbe(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{}
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
			accountRepo:  repo,
			httpUpstream: upstream,
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

	require.Nil(t, upstream.lastReq, "manual recovery must not send any probe request")
	require.Equal(t, 1, repo.clearCalls, "manual recovery still clears DB temp unschedulable")
	require.Equal(t, 0, repo.setCalls, "manual recovery must not re-mark temp unschedulable")
}

func TestProbeManualRecoveryRemovesEntry(t *testing.T) {
	repo := &manualRecoveryProbeRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{{
		ID:          92,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
	}}}}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{accountRepo: repo},
		stats:   newOpenAIAccountRuntimeStats(),
		ctx:     context.Background(),
		stopCh:  make(chan struct{}),
	}
	defer probe.stop()
	entry := &openAIAccountProbeEntry{accountID: 92}
	entry.dbFlagSet.Store(true)
	probe.entries.Store(int64(92), entry)

	probe.applyManualRecovery(92, entry)

	_, present := probe.entries.Load(int64(92))
	require.False(t, present, "manual recovery removes the probe entry")
}

func TestOpenAIManualTempUnschedulableClearDoesNotTriggerProbe(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{}
	repo := &manualRecoveryProbeRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{{
		ID:          95,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
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

	require.Nil(t, upstream.lastReq, "ClearAccountSchedulingBlock must not trigger any probe request")
	_, blockPresent := svc.openaiAccountRuntimeBlockUntil.Load(int64(95))
	require.False(t, blockPresent, "ClearAccountSchedulingBlock still clears the runtime block")
}

func TestOpenAIManualTempUnschedulableClearWithoutLayeredProbeIsNoop(t *testing.T) {
	svc := &OpenAIGatewayService{}
	svc.openaiAccountRuntimeBlockUntil.Store(int64(96), time.Now().Add(time.Minute))

	svc.ClearAccountSchedulingBlock(96)

	_, ok := svc.openaiAccountRuntimeBlockUntil.Load(int64(96))
	require.False(t, ok, "ClearAccountSchedulingBlock clears runtime block even when no probe is configured")
}
