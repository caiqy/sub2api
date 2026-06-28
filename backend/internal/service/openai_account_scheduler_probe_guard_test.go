//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLayeredScheduler_MarkPenalizedSkipsProbeDisabledAccount(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(nil, stats)
	t.Cleanup(probe.stop)

	scheduler := &layeredOpenAIAccountScheduler{
		probe: probe,
		stats: stats,
	}

	disabled := &Account{
		ID:       77,
		Platform: PlatformOpenAI,
		Extra:    map[string]any{"openai_probe_enabled": false},
	}

	scheduler.applyProbeRegistration(disabled, true, false, nil)

	_, present := probe.entries.Load(int64(77))
	require.False(t, present, "probe-disabled account must not be registered to probe.entries")
}

func TestLayeredScheduler_MarkPenalizedRunsForProbeEnabledAccount(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(nil, stats)
	t.Cleanup(probe.stop)

	scheduler := &layeredOpenAIAccountScheduler{
		probe: probe,
		stats: stats,
	}

	enabled := &Account{
		ID:       78,
		Platform: PlatformOpenAI,
	}

	scheduler.applyProbeRegistration(enabled, true, false, nil)

	_, present := probe.entries.Load(int64(78))
	require.True(t, present, "probe-enabled account must be registered as before")
}

func TestLayeredScheduler_MarkPenalizedSkipsClearForProbeDisabledAccount(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(nil, stats)
	t.Cleanup(probe.stop)

	scheduler := &layeredOpenAIAccountScheduler{
		probe: probe,
		stats: stats,
	}

	// Pre-place an entry (simulating state left by another source)
	probe.entries.Store(int64(79), &openAIAccountProbeEntry{accountID: 79})

	disabled := &Account{
		ID:       79,
		Platform: PlatformOpenAI,
		Extra:    map[string]any{"openai_probe_enabled": false},
	}

	// errorPenalized=false, ttftPenalized=false → normally would call clearPenaltyReasons
	scheduler.applyProbeRegistration(disabled, false, false, nil)

	_, present := probe.entries.Load(int64(79))
	require.True(t, present, "probe-disabled account must not have its existing entry cleared")
}
