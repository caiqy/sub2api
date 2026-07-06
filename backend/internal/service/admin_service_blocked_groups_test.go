//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type blockedGroupRepoStub struct {
	groupRepoStub
	groups map[int64]*Group
}

func (s *blockedGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if group, ok := s.groups[id]; ok {
		return group, nil
	}
	return nil, ErrGroupNotFound
}

func TestAdminServiceUpdateUserBlockedGroups(t *testing.T) {
	ctx := context.Background()
	repo := &userRepoStub{user: &User{ID: 42, Email: "u@example.com"}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		groupRepo:            &blockedGroupRepoStub{groups: map[int64]*Group{10: {ID: 10, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard}}},
		redeemCodeRepo:       &redeemRepoStub{},
		authCacheInvalidator: invalidator,
	}

	blocked := []int64{10}
	updated, err := svc.UpdateUser(ctx, 42, &UpdateUserInput{BlockedGroups: &blocked})
	require.NoError(t, err)
	require.Equal(t, []int64{10}, repo.blockedGroups[42])
	require.Equal(t, []int64{10}, updated.BlockedGroups)
	require.Equal(t, []int64{42}, invalidator.userIDs)
}

func TestAdminServiceUpdateUserBlockedGroupsRejectsNonPublicStandardGroup(t *testing.T) {
	ctx := context.Background()
	svc := &adminServiceImpl{
		userRepo: &userRepoStub{user: &User{ID: 42, Email: "u@example.com"}},
		groupRepo: &blockedGroupRepoStub{groups: map[int64]*Group{
			10: {ID: 10, Status: StatusActive, IsExclusive: true, SubscriptionType: SubscriptionTypeStandard},
		}},
		redeemCodeRepo: &redeemRepoStub{},
	}

	blocked := []int64{10}
	_, err := svc.UpdateUser(ctx, 42, &UpdateUserInput{BlockedGroups: &blocked})
	require.ErrorContains(t, err, "blocked_groups")
}
