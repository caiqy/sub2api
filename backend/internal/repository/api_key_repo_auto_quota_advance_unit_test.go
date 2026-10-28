package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRepository_GetByKeyForAuth_PreservesQuotaAdvanceSwitches(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "quota-advance-auth@test.com")

	for _, tt := range []struct {
		name  string
		allow bool
		auto  bool
	}{
		{"both enabled", true, true},
		{"manual only", true, false},
		{"auto only", false, true},
		{"both disabled", false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			group, err := client.Group.Create().
				SetName(tt.name).
				SetPlatform(service.PlatformOpenAI).
				SetStatus(service.StatusActive).
				SetSubscriptionType(service.SubscriptionTypeSubscription).
				SetAllowQuotaAdvance(tt.allow).
				SetAutoQuotaAdvance(tt.auto).
				Save(ctx)
			require.NoError(t, err)
			key, err := client.APIKey.Create().
				SetUserID(user.ID).
				SetKey("sk-quota-advance-" + tt.name).
				SetName(tt.name).
				SetGroupID(group.ID).
				SetStatus(service.StatusActive).
				Save(ctx)
			require.NoError(t, err)

			got, err := repo.GetByKeyForAuth(ctx, key.Key)

			require.NoError(t, err)
			require.NotNil(t, got.Group)
			require.Equal(t, tt.allow, got.Group.AllowQuotaAdvance)
			require.Equal(t, tt.auto, got.Group.AutoQuotaAdvance)
		})
	}
}
