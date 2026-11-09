package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestReadRemainingQuotaIncludesDisabledAndExcludesDeleted(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()
	now := time.Now()
	active, err := client.User.Create().SetEmail("a@example.com").SetPasswordHash("hash").SetBalance(12).Save(ctx)
	require.NoError(t, err)
	disabled, err := client.User.Create().SetEmail("b@example.com").SetPasswordHash("hash").SetStatus(service.StatusDisabled).SetBalance(-2).Save(ctx)
	require.NoError(t, err)
	deleted, err := client.User.Create().SetEmail("c@example.com").SetPasswordHash("hash").SetBalance(100).SetDeletedAt(now).Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().SetName("quota").SetDailyLimitUsd(10).Save(ctx)
	require.NoError(t, err)
	for _, id := range []int64{active.ID, disabled.ID, deleted.ID} {
		_, err = client.UserSubscription.Create().SetStartsAt(now.Add(-24 * time.Hour)).SetUserID(id).SetGroupID(group.ID).SetExpiresAt(now.Add(time.Hour)).Save(ctx)
		require.NoError(t, err)
	}
	for _, status := range []string{service.SubscriptionStatusExpired, service.SubscriptionStatusSuspended, service.SubscriptionStatusActive} {
		g, createErr := client.Group.Create().SetName(status).Save(ctx)
		require.NoError(t, createErr)
		expires := now.Add(time.Hour)
		if status == service.SubscriptionStatusActive {
			expires = now
		}
		_, createErr = client.UserSubscription.Create().SetStartsAt(now.Add(-24 * time.Hour)).SetUserID(active.ID).SetGroupID(g.ID).SetStatus(status).SetExpiresAt(expires).Save(ctx)
		require.NoError(t, createErr)
	}
	balance, subs, err := repo.ReadRemainingQuota(ctx, nil, now)
	require.NoError(t, err)
	require.Equal(t, 10.0, balance)
	require.Len(t, subs, 2)
	require.NotNil(t, subs[0].Group)
	balance, subs, err = repo.ReadRemainingQuota(ctx, []int64{active.ID}, now)
	require.NoError(t, err)
	require.Equal(t, -2.0, balance)
	require.Len(t, subs, 1)
	require.Equal(t, disabled.ID, subs[0].UserID)
	balance, subs, err = repo.ReadRemainingQuota(ctx, []int64{active.ID, disabled.ID}, now)
	require.NoError(t, err)
	require.Zero(t, balance)
	require.Empty(t, subs)
}
