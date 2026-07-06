package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type adminAPIKeyBlockedUserRepo struct {
	user *User
}

type adminAPIKeyBlockedAPIKeyRepo struct {
	key     *APIKey
	updated *APIKey
}

func (r *adminAPIKeyBlockedAPIKeyRepo) GetByID(context.Context, int64) (*APIKey, error) {
	clone := *r.key
	return &clone, nil
}

func (r *adminAPIKeyBlockedAPIKeyRepo) Update(_ context.Context, key *APIKey) error {
	clone := *key
	r.updated = &clone
	return nil
}

func (r *adminAPIKeyBlockedAPIKeyRepo) Create(context.Context, *APIKey) error { panic("unexpected") }
func (r *adminAPIKeyBlockedAPIKeyRepo) GetKeyAndOwnerID(context.Context, int64) (string, int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) GetByKey(context.Context, string) (*APIKey, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) GetByKeyForAuth(context.Context, string) (*APIKey, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) Delete(context.Context, int64) error { panic("unexpected") }
func (r *adminAPIKeyBlockedAPIKeyRepo) DeleteWithAudit(context.Context, int64) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) ListByUserID(context.Context, int64, pagination.PaginationParams, APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) VerifyOwnership(context.Context, int64, []int64) ([]int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) CountByUserID(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) ExistsByKey(context.Context, string) (bool, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) SearchAPIKeys(context.Context, int64, string, int) ([]APIKey, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) ClearGroupIDByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) CountByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) ListKeysByUserID(context.Context, int64) ([]string, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) ListKeysByGroupID(context.Context, int64) ([]string, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) IncrementQuotaUsed(context.Context, int64, float64) (float64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) UpdateLastUsed(context.Context, int64, time.Time) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) IncrementRateLimitUsage(context.Context, int64, float64) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) ResetRateLimitWindows(context.Context, int64) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) GetRateLimitData(context.Context, int64) (*APIKeyRateLimitData, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedAPIKeyRepo) UpdateGroupIDByUserAndGroup(context.Context, int64, int64, int64) (int64, error) {
	panic("unexpected")
}

type adminAPIKeyBlockedGroupRepo struct {
	group *Group
}

func (r *adminAPIKeyBlockedGroupRepo) GetByID(context.Context, int64) (*Group, error) {
	clone := *r.group
	return &clone, nil
}

func (r *adminAPIKeyBlockedGroupRepo) Create(context.Context, *Group) error { panic("unexpected") }
func (r *adminAPIKeyBlockedGroupRepo) GetByIDLite(context.Context, int64) (*Group, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) Update(context.Context, *Group) error { panic("unexpected") }
func (r *adminAPIKeyBlockedGroupRepo) Delete(context.Context, int64) error  { panic("unexpected") }
func (r *adminAPIKeyBlockedGroupRepo) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) ListActive(context.Context) ([]Group, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedGroupRepo) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	panic("unexpected")
}

func (r *adminAPIKeyBlockedUserRepo) GetByID(context.Context, int64) (*User, error) {
	clone := *r.user
	return &clone, nil
}

func (r *adminAPIKeyBlockedUserRepo) Create(context.Context, *User) error { panic("unexpected") }
func (r *adminAPIKeyBlockedUserRepo) GetByEmail(context.Context, string) (*User, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) Update(context.Context, *User) error { panic("unexpected") }
func (r *adminAPIKeyBlockedUserRepo) Delete(context.Context, int64) error { panic("unexpected") }
func (r *adminAPIKeyBlockedUserRepo) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) DeleteUserAvatar(context.Context, int64) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) GetBlockedGroups(context.Context, int64) ([]int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) SetBlockedGroups(context.Context, int64, []int64) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) GetHiddenUIResources(context.Context, int64) (bool, []int64, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) SetHiddenUIResources(context.Context, int64, bool, []string) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) UnbindUserAuthProvider(context.Context, int64, string) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected")
}
func (r *adminAPIKeyBlockedUserRepo) EnableTotp(context.Context, int64) error  { panic("unexpected") }
func (r *adminAPIKeyBlockedUserRepo) DisableTotp(context.Context, int64) error { panic("unexpected") }
func (r *adminAPIKeyBlockedUserRepo) GetByIDIncludeDeleted(context.Context, int64) (*User, error) {
	panic("unexpected")
}

func TestAdminServiceAdminUpdateAPIKeyGroupIDRejectsBlockedPublicGroup(t *testing.T) {
	existing := &APIKey{ID: 1, UserID: 42, Key: "sk-test", GroupID: nil}
	apiKeyRepo := &adminAPIKeyBlockedAPIKeyRepo{key: existing}
	groupRepo := &adminAPIKeyBlockedGroupRepo{group: &Group{ID: 10, Name: "Public", Status: StatusActive, SubscriptionType: SubscriptionTypeStandard}}
	userRepo := &adminAPIKeyBlockedUserRepo{user: &User{ID: 42, BlockedGroups: []int64{10}}}
	svc := &adminServiceImpl{apiKeyRepo: apiKeyRepo, groupRepo: groupRepo, userRepo: userRepo}

	_, err := svc.AdminUpdateAPIKeyGroupID(context.Background(), 1, int64Ptr(10))
	require.Error(t, err)
	require.Equal(t, "GROUP_NOT_ALLOWED", infraerrors.Reason(err))
	require.Nil(t, apiKeyRepo.updated)
}
