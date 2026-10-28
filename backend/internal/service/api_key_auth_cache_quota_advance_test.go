//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// The normal authentication path must preserve group consent through both a
// cold lookup and the JSON snapshot used by the shared cache.
func TestAPIKeyService_GetByKey_PreservesAutoQuotaAdvance(t *testing.T) {
	for _, flags := range []struct {
		name        string
		manual      bool
		automatic   bool
		wantAdvance bool
	}{
		{"disabled", false, false, false},
		{"manual only", true, false, false},
		{"automatic only", false, true, false},
		{"enabled", true, true, true},
	} {
		for _, cached := range []bool{false, true} {
			name := flags.name + "/cold"
			if cached {
				name = flags.name + "/L2 JSON"
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
				sub := exhaustedQuotaSubscription(now)
				sub.AutoQuotaAdvance = true
				sub.WeeklyUsageUSD, sub.MonthlyUsageUSD = 0, 0
				sub.Group.AllowQuotaAdvance = flags.manual
				sub.Group.AutoQuotaAdvance = flags.automatic
				key := &APIKey{
					ID: 3, Key: "sk-auto-quota-auth", UserID: sub.UserID, GroupID: &sub.GroupID,
					Status: StatusActive, User: &User{ID: sub.UserID, Status: StatusActive}, Group: sub.Group,
				}
				repo := &authRepoStub{getByKeyForAuth: func(context.Context, string) (*APIKey, error) {
					require.False(t, cached, "a valid L2 snapshot must not fall back to the repository")
					return key, nil
				}}
				cache := &authCacheStub{}
				keySvc := NewAPIKeyService(repo, nil, nil, nil, nil, cache, &config.Config{
					APIKeyAuth: config.APIKeyAuthCacheConfig{L2TTLSeconds: 60},
				})
				if cached {
					payload, err := json.Marshal(&APIKeyAuthCacheEntry{Snapshot: keySvc.snapshotFromAPIKey(ctx, key)})
					require.NoError(t, err)
					cache.getAuthCache = func(context.Context, string) (*APIKeyAuthCacheEntry, error) {
						var entry APIKeyAuthCacheEntry
						require.NoError(t, json.Unmarshal(payload, &entry))
						return &entry, nil
					}
				}

				authenticated, err := keySvc.GetByKey(ctx, key.Key)
				require.NoError(t, err)
				require.NotNil(t, authenticated.Group)
				require.Equal(t, flags.manual, authenticated.Group.AllowQuotaAdvance)
				require.Equal(t, flags.automatic, authenticated.Group.AutoQuotaAdvance)

				subRepo := newTermLockingUserSubRepo()
				subRepo.seed(sub)
				subSvc := &SubscriptionService{userSubRepo: subRepo, entClient: newPaymentOrderLifecycleTestClient(t), now: func() time.Time { return now }}
				advanced, err := subSvc.TryAutoAdvanceQuotaCycle(ctx, sub, authenticated.Group)
				require.NoError(t, err)
				if flags.wantAdvance {
					require.NotNil(t, advanced, "a normal authenticated request must reach automatic quota advance")
					require.Equal(t, sub.ExpiresAt.Add(-20*time.Hour), advanced.ExpiresAt)
					require.Zero(t, advanced.DailyUsageUSD)
				} else {
					require.Nil(t, advanced)
					stored, ok := subRepo.lookupByID(sub.ID)
					require.True(t, ok)
					require.Equal(t, sub.ExpiresAt, stored.ExpiresAt)
				}
			})
		}
	}
}

func TestAPIKeyService_RejectsV25AuthSnapshotWithoutQuotaAdvanceConsent(t *testing.T) {
	svc := &APIKeyService{}
	key, used, err := svc.applyAuthCacheEntry("sk-before-quota-advance", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{Version: 25},
	})
	require.NoError(t, err)
	require.False(t, used, "snapshots without quota advance fields must be rebuilt from the repository")
	require.Nil(t, key)
}
