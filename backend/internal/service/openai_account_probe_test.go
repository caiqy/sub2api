package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type failingClearTempUnschedulableRepo struct{ stubOpenAIAccountRepo }

type probeGroupAwareRepo struct{ stubOpenAIAccountRepo }

func (f failingClearTempUnschedulableRepo) ClearTempUnschedulable(ctx context.Context, id int64) error {
	return errors.New("clear failed")
}

func (r probeGroupAwareRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform && acc.IsSchedulable() && len(acc.AccountGroups) == 0 {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r probeGroupAwareRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform != platform || !acc.IsSchedulable() {
			continue
		}
		for _, ag := range acc.AccountGroups {
			if ag.GroupID == groupID {
				result = append(result, acc)
				break
			}
		}
	}
	return result, nil
}

func TestProbe_MarkPenalized_RegistersAccount(t *testing.T) {
	probe := &openAIAccountProbe{
		stats:  newOpenAIAccountRuntimeStats(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()
	probe.markPenalized(42, true, false)
	_, ok := probe.entries.Load(int64(42))
	require.True(t, ok)
}

func TestProbe_MarkPenalized_IsIdempotent(t *testing.T) {
	probe := &openAIAccountProbe{
		stats:  newOpenAIAccountRuntimeStats(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()

	probe.markPenalized(42, true, false)
	val1, ok1 := probe.entries.Load(int64(42))
	require.True(t, ok1)

	probe.markPenalized(42, true, false)
	val2, ok2 := probe.entries.Load(int64(42))
	require.True(t, ok2)

	// LoadOrStore returns the existing entry on second call, so pointers must match.
	require.Same(t, val1.(*openAIAccountProbeEntry), val2.(*openAIAccountProbeEntry))
}

func TestProbe_MarkPenalized_UpdatesReasonFlags(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{})}
	defer probe.stop()

	probe.markPenalized(42, true, false)
	v, ok := probe.entries.Load(int64(42))
	require.True(t, ok)
	entry := v.(*openAIAccountProbeEntry)
	require.True(t, entry.errorPenalized.Load())
	require.False(t, entry.ttftPenalized.Load())

	probe.markPenalized(42, false, true)
	v, _ = probe.entries.Load(int64(42))
	entry = v.(*openAIAccountProbeEntry)
	require.True(t, entry.errorPenalized.Load())
	require.True(t, entry.ttftPenalized.Load())
}

func TestProbe_ClearPenaltyReasons_RemovesEntryWhenNoReasonsRemain(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{})}
	defer probe.stop()

	probe.markPenalized(7, true, true)
	probe.clearPenaltyReasons(7)
	_, ok := probe.entries.Load(int64(7))
	require.False(t, ok)
}

func TestProbe_ClearPenaltyReasons_DoesNotRemoveEntryWhenProbing(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{})}
	defer probe.stop()

	probe.markPenalized(9, true, true)
	value, ok := probe.entries.Load(int64(9))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)
	entry.probing.Store(true)

	probe.clearPenaltyReasons(9)
	_, ok = probe.entries.Load(int64(9))
	require.True(t, ok, "entry must remain while probing is in-flight")
	require.False(t, entry.errorPenalized.Load())
	require.False(t, entry.ttftPenalized.Load())
}

func TestProbe_ClearPenaltyReasons_DoesNotRemoveEntryWhenDBFlagSet(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{})}
	defer probe.stop()

	probe.markPenalized(10, true, true)
	value, ok := probe.entries.Load(int64(10))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)
	entry.dbFlagSet.Store(true)

	probe.clearPenaltyReasons(10)
	_, ok = probe.entries.Load(int64(10))
	require.True(t, ok, "entry must remain while db flag is set")
	require.False(t, entry.errorPenalized.Load())
	require.False(t, entry.ttftPenalized.Load())
}

func TestProbe_FinalizePenaltyState_KeepsEntryWhenTTFTReasonRemains(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{}), ctx: context.Background()}
	defer probe.stop()

	probe.markPenalized(1, true, true)
	value, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)

	entry.errorPenalized.Store(false)
	entry.ttftPenalized.Store(true)
	probe.finalizePenaltyState(1, entry)

	_, ok = probe.entries.Load(int64(1))
	require.True(t, ok, "entry must remain while TTFT reason is still active")
}

func TestProbe_FinalizePenaltyState_RemovesEntryWhenBothReasonsClear(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{}), ctx: context.Background()}
	defer probe.stop()

	probe.markPenalized(1, true, true)
	value, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)

	entry.errorPenalized.Store(false)
	entry.ttftPenalized.Store(false)
	probe.finalizePenaltyState(1, entry)

	_, ok = probe.entries.Load(int64(1))
	require.False(t, ok, "entry should be removed only after both reasons clear")
}

func TestProbe_ReevaluatePenaltyReasons_UsesSharedGroupBaseline(t *testing.T) {
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 10},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50},
	}
	svc := newLayeredTestService(accounts)
	defer svc.StopOpenAIAccountScheduler()

	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(svc, stats)
	defer probe.stop()

	for i := 0; i < 5; i++ {
		slow := 9000
		fast := 1000
		stats.report(1, true, &slow)
		stats.report(2, true, &fast)
	}

	eval, err := probe.reevaluatePenaltyReasons(context.Background(), 1, nil)
	require.NoError(t, err)
	require.True(t, eval.TTFTPenalized)
	require.False(t, eval.ErrorPenalized)
	require.Greater(t, eval.GroupMinTTFT, 0.0)
}

func TestProbe_ReevaluatePenaltyReasons_UsesAccountGroupID(t *testing.T) {
	groupID := int64(100)
	accounts := []Account{
		{
			ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Schedulable: true, Concurrency: 3, Priority: 10,
			AccountGroups: []AccountGroup{{GroupID: groupID}},
		},
		{
			ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Schedulable: true, Concurrency: 3, Priority: 50,
			AccountGroups: []AccountGroup{{GroupID: groupID}},
		},
		{
			ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Schedulable: true, Concurrency: 3, Priority: 50,
			AccountGroups: []AccountGroup{{GroupID: 200}},
		},
	}
	repo := probeGroupAwareRepo{stubOpenAIAccountRepo{accounts: accounts}}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold = 0.3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue = 100
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier = 3.0
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue = 50
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 60
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 30
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures = 3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds = 15

	svc := &OpenAIGatewayService{accountRepo: repo, cfg: cfg}
	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(svc, stats)
	defer probe.stop()

	for i := 0; i < 5; i++ {
		slow := 9000
		fastSameGroup := 1000
		fastOtherGroup := 500
		stats.report(1, true, &slow)
		stats.report(2, true, &fastSameGroup)
		stats.report(3, true, &fastOtherGroup)
	}

	eval, err := probe.reevaluatePenaltyReasons(context.Background(), 1, probeAccountGroupID(&accounts[0]))
	require.NoError(t, err)
	require.True(t, eval.TTFTPenalized, "account 1 should compare against same-group min TTFT")
	require.InDelta(t, 1000.0, eval.GroupMinTTFT, 0.01, "group min TTFT must come from same group, not other groups")
}

func TestProbe_RecoverAccount_ResetsEWMA(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	probe := &openAIAccountProbe{
		stats:  stats,
		stopCh: make(chan struct{}),
	}
	defer probe.stop()

	// Report 5 failures → errorRate > 0.3
	for i := 0; i < 5; i++ {
		stats.report(1, false, nil)
	}
	errorRate, _, _ := stats.snapshot(1)
	require.Greater(t, errorRate, 0.3, "errorRate should exceed 0.3 after 5 failures")

	// Register entry in probe list
	probe.markPenalized(1, true, false)
	val, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
	entry := val.(*openAIAccountProbeEntry)

	// Recover
	probe.recoverAccount(1, entry)

	// After recovery: errorRate == 0, TTFT unchanged (not reset)
	errorRate, _, _ = stats.snapshot(1)
	require.Equal(t, 0.0, errorRate, "errorRate should be reset to 0")

	// Entry should be removed from probe list
	_, ok = probe.entries.Load(int64(1))
	require.False(t, ok, "entry should be removed from probe list after recovery")
}

func TestProbe_RecoverAccount_KeepsEntryWhenClearTempUnschedulableFails(t *testing.T) {
	probe := &openAIAccountProbe{
		stats:  newOpenAIAccountRuntimeStats(),
		stopCh: make(chan struct{}),
		ctx:    context.Background(),
		service: &OpenAIGatewayService{
			accountRepo: failingClearTempUnschedulableRepo{},
		},
	}
	defer probe.stop()

	probe.markPenalized(123, true, false)
	value, ok := probe.entries.Load(int64(123))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)
	entry.dbFlagSet.Store(true)

	probe.recoverAccount(123, entry)

	_, ok = probe.entries.Load(int64(123))
	require.True(t, ok, "entry must remain so future probes can retry DB cleanup")
	require.True(t, entry.dbFlagSet.Load(), "dbFlagSet should remain true after failed clear")
}

func TestProbe_SuccessPath_LeavesEntryWhenTTFTStillPenalized(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{}), ctx: context.Background()}
	defer probe.stop()
	probe.markPenalized(1, true, true)
	value, _ := probe.entries.Load(int64(1))
	entry := value.(*openAIAccountProbeEntry)

	entry.errorPenalized.Store(false)
	entry.ttftPenalized.Store(true)
	probe.finalizePenaltyState(1, entry)

	_, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
}

func TestProbe_ResetAccount_PreservesTTFT(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()

	// Report a TTFT so hasTTFT becomes true
	ttftVal := 500
	stats.report(1, true, &ttftVal)
	_, ttft, hasTTFT := stats.snapshot(1)
	require.True(t, hasTTFT)
	require.Greater(t, ttft, 0.0)

	// Reset — should only clear errorRate, not TTFT
	stats.resetAccount(1)

	errRate, ttftAfter, hasTTFTAfter := stats.snapshot(1)
	require.Equal(t, 0.0, errRate, "errorRate should be 0 after reset")
	require.True(t, hasTTFTAfter, "TTFT should be preserved after reset")
	require.InDelta(t, ttft, ttftAfter, 0.01, "TTFT value should be unchanged")
}

func TestProbe_Stop_PreventsNewRegistrations(t *testing.T) {
	probe := &openAIAccountProbe{
		stats:  newOpenAIAccountRuntimeStats(),
		stopCh: make(chan struct{}),
	}
	probe.stop()

	probe.markPenalized(99, true, false)

	_, ok := probe.entries.Load(int64(99))
	require.False(t, ok, "markPenalized should be no-op after stop()")
}

func TestProbe_StopCancelsRootContextAndPreventsNewWork(t *testing.T) {
	probe := newOpenAIAccountProbe(nil, newOpenAIAccountRuntimeStats())
	require.NotNil(t, probe)

	// stop() 的生命周期契约：
	// 1) 会取消 probe 根 context；
	// 2) 会等待 loop 与所有已注册 worker 退出；
	// 3) 一旦 stop 开始，就不应再接受新的已注册工作。
	// 这里用一个已注册的 in-flight worker 来验证 stop 会等待其观察到取消。
	probe.wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer probe.wg.Done()
		<-probe.ctx.Done()
	}()

	probe.stop()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("probe.stop() did not wait for in-flight work to observe cancellation")
	}

	require.True(t, probe.stopped.Load())
	select {
	case <-probe.ctx.Done():
	default:
		t.Fatal("probe root context should be canceled after stop")
	}

	select {
	case <-probe.stopCh:
	default:
		t.Fatal("probe stopCh should be closed after stop")
	}
}
