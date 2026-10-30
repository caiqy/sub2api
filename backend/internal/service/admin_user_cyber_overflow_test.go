//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCyberOverflowAuthColdHotAndAdminInvalidation(t *testing.T) {
	ctx := context.Background()
	users := &rpmUserRepoStub{userRepoStub: &userRepoStub{user: &User{ID: 42, Email: "u@example.com"}}}
	calls := map[string]int{}
	keys := &authRepoStub{
		getByKeyForAuth: func(_ context.Context, key string) (*APIKey, error) {
			calls[key]++
			user := *users.user
			if key == "other" {
				user = User{ID: 43, CyberOverflowExempt: true}
			}
			return &APIKey{ID: 1, UserID: user.ID, User: &user}, nil
		},
		listKeysByUserID: func(_ context.Context, userID int64) ([]string, error) {
			require.Equal(t, int64(42), userID)
			return []string{"first", "second"}, nil
		},
	}
	auth := NewAPIKeyService(keys, nil, nil, nil, nil, &authCacheStub{}, &config.Config{APIKeyAuth: config.APIKeyAuthCacheConfig{L1Size: 1000, L1TTLSeconds: 60}})
	t.Cleanup(auth.authCacheL1.Close)
	admin := &adminServiceImpl{userRepo: users, redeemCodeRepo: &redeemRepoStub{}, authCacheInvalidator: auth}
	for round, exempt := range []bool{false, true, false} {
		if round > 0 {
			_, err := admin.UpdateUser(ctx, 42, &UpdateUserInput{CyberOverflowExempt: &exempt})
			require.NoError(t, err)
		}
		for _, key := range []string{"first", "second", "other"} {
			for range 2 {
				loaded, err := auth.GetByKey(ctx, key)
				require.NoError(t, err)
				require.Equal(t, exempt || key == "other", loaded.User.CyberOverflowExempt)
				auth.authCacheL1.Wait()
			}
			if key == "other" {
				require.Equal(t, 1, calls[key])
			} else {
				require.Equal(t, round+1, calls[key])
			}
		}
	}
}

func TestAdminUserCyberOverflowPolicy(t *testing.T) {
	repo := &rpmUserRepoStub{userRepoStub: &userRepoStub{user: &User{ID: 42, Email: "u@example.com"}}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}, authCacheInvalidator: invalidator}
	for _, exempt := range []bool{true, false} {
		updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{CyberOverflowExempt: &exempt})
		require.NoError(t, err)
		require.Equal(t, exempt, updated.CyberOverflowExempt)
	}
	require.Equal(t, []int64{42, 42}, invalidator.userIDs, "both directions invalidate all keys for this user")
	exempt := true
	_, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{CyberOverflowExempt: &exempt})
	require.NoError(t, err)
	name := "renamed"
	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Username: &name})
	require.NoError(t, err)
	require.True(t, updated.CyberOverflowExempt, "omitting policy preserves the stored value")
	require.Equal(t, []int64{42, 42, 42}, invalidator.userIDs, "omitted policy does not invalidate")
}
