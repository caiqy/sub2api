package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
