//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGrokTeamModelRateLimit_MarksAndFiltersSiblings(t *testing.T) {
	// Isolate from other tests by using unique team ids.
	team := "team-test-" + time.Now().Format("150405.000")
	a1 := &Account{
		ID: 101, Platform: PlatformGrok, Type: AccountTypeOAuth,
		Credentials: map[string]any{"team_id": team},
	}
	a2 := &Account{
		ID: 102, Platform: PlatformGrok, Type: AccountTypeOAuth,
		Credentials: map[string]any{"team_id": team},
	}
	other := &Account{
		ID: 103, Platform: PlatformGrok, Type: AccountTypeOAuth,
		Credentials: map[string]any{"team_id": team + "-other"},
	}
	noTeam := &Account{
		ID: 104, Platform: PlatformGrok, Type: AccountTypeOAuth,
		Credentials: map[string]any{},
	}

	now := time.Now()
	markGrokTeamModelRateLimit(a1, "grok-4.5", now.Add(5*time.Minute))

	require.True(t, isGrokTeamModelRateLimited(a1, "grok-4.5", now))
	require.True(t, isGrokTeamModelRateLimited(a2, "grok-4.5", now), "sibling with same team must cool")
	require.False(t, isGrokTeamModelRateLimited(a2, "grok-4.3", now), "other model stays pickable")
	require.False(t, isGrokTeamModelRateLimited(other, "grok-4.5", now))
	require.False(t, isGrokTeamModelRateLimited(noTeam, "grok-4.5", now))

	filtered := filterGrokTeamModelRateLimitedAccounts([]Account{*a1, *a2, *other, *noTeam}, "grok-4.5", now)
	require.Len(t, filtered, 2)
	ids := []int64{filtered[0].ID, filtered[1].ID}
	require.Contains(t, ids, int64(103))
	require.Contains(t, ids, int64(104))
}

func TestGrokTeamModelRateLimit_Expires(t *testing.T) {
	team := "team-expire-" + time.Now().Format("150405.000")
	a := &Account{
		ID: 201, Platform: PlatformGrok, Type: AccountTypeOAuth,
		Credentials: map[string]any{"team_id": team},
	}
	now := time.Now()
	key := grokTeamModelRateLimitKey(grokTeamFingerprint(team), "grok-4.5")
	globalGrokTeamModelRateLimits.mu.Lock()
	globalGrokTeamModelRateLimits.items[key] = grokTeamModelRateLimit{Until: now.Add(-time.Minute)}
	globalGrokTeamModelRateLimits.mu.Unlock()

	require.False(t, isGrokTeamModelRateLimited(a, "grok-4.5", now))
	globalGrokTeamModelRateLimits.mu.Lock()
	_, found := globalGrokTeamModelRateLimits.items[key]
	globalGrokTeamModelRateLimits.mu.Unlock()
	require.False(t, found)
}

func TestGrokTeamModelRateLimitFilterUsesMappedUpstreamModel(t *testing.T) {
	now := time.Now()
	account := &Account{
		ID:       301,
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"team_id":       "team-mapped-301",
			"model_mapping": map[string]any{"gpt-*": "grok-4.5"},
		},
	}
	markGrokTeamModelRateLimit(account, "grok-4.5", now.Add(time.Hour))

	require.Empty(t, filterGrokTeamModelRateLimitedAccounts([]Account{*account}, "gpt-5", now))
}

func TestGrokSchedulingEligibilityUsesFinalResponsesModelID(t *testing.T) {
	now := time.Now()
	account := &Account{
		ID:       302,
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"team_id":       "team-final-model-302",
			"model_mapping": map[string]any{"gpt-*": "grok-4.20-multi-agent"},
		},
	}
	markGrokTeamModelRateLimit(account, "grok-4.20-multi-agent-0309", now.Add(time.Hour))

	require.False(t, isGrokSchedulingEligible(account, "gpt-5", now))
	require.Equal(t, "", canonicalOpenAIAccountSchedulingModel(account, ""))
}
