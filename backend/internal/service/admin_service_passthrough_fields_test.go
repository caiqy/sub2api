package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type passthroughAdminAccountRepo struct {
	accountsByID map[int64]*Account
	created      *Account
	updated      *Account
	updatedIDs   []int64
	bulkUpdated  *AccountBulkUpdate
	bulkIDs      []int64
}

func (m *passthroughAdminAccountRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	if account, ok := m.accountsByID[id]; ok {
		return cloneAccountForPassthroughTest(account), nil
	}
	return nil, errors.New("account not found")
}

func (m *passthroughAdminAccountRepo) GetByIDs(ctx context.Context, ids []int64) ([]*Account, error) {
	result := make([]*Account, 0, len(ids))
	for _, id := range ids {
		if account, ok := m.accountsByID[id]; ok {
			result = append(result, cloneAccountForPassthroughTest(account))
		}
	}
	return result, nil
}

func (m *passthroughAdminAccountRepo) ExistsByID(ctx context.Context, id int64) (bool, error) {
	_, ok := m.accountsByID[id]
	return ok, nil
}

func (m *passthroughAdminAccountRepo) GetByCRSAccountID(ctx context.Context, crsAccountID string) (*Account, error) {
	return nil, nil
}

func (m *passthroughAdminAccountRepo) FindByExtraField(ctx context.Context, key string, value any) ([]Account, error) {
	return nil, nil
}

func (m *passthroughAdminAccountRepo) ListCRSAccountIDs(ctx context.Context) (map[string]int64, error) {
	return nil, nil
}

func (m *passthroughAdminAccountRepo) Create(ctx context.Context, account *Account) error {
	clone := cloneAccountForPassthroughTest(account)
	if clone.ID == 0 {
		clone.ID = 1
	}
	account.ID = clone.ID
	m.created = clone
	if m.accountsByID == nil {
		m.accountsByID = map[int64]*Account{}
	}
	m.accountsByID[clone.ID] = clone
	return nil
}

func (m *passthroughAdminAccountRepo) Update(ctx context.Context, account *Account) error {
	clone := cloneAccountForPassthroughTest(account)
	m.updated = clone
	m.updatedIDs = append(m.updatedIDs, clone.ID)
	if m.accountsByID == nil {
		m.accountsByID = map[int64]*Account{}
	}
	m.accountsByID[clone.ID] = clone
	return nil
}

func (m *passthroughAdminAccountRepo) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	return nil
}

func (m *passthroughAdminAccountRepo) Delete(ctx context.Context, id int64) error { return nil }
func (m *passthroughAdminAccountRepo) List(ctx context.Context, params pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *passthroughAdminAccountRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *passthroughAdminAccountRepo) ListByGroup(ctx context.Context, groupID int64) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) ListActive(ctx context.Context) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) UpdateLastUsed(ctx context.Context, id int64) error { return nil }
func (m *passthroughAdminAccountRepo) BatchUpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	return nil
}
func (m *passthroughAdminAccountRepo) SetError(ctx context.Context, id int64, errorMsg string) error {
	return nil
}
func (m *passthroughAdminAccountRepo) ClearError(ctx context.Context, id int64) error {
	return nil
}
func (m *passthroughAdminAccountRepo) SetSchedulable(ctx context.Context, id int64, schedulable bool) error {
	return nil
}
func (m *passthroughAdminAccountRepo) AutoPauseExpiredAccounts(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}
func (m *passthroughAdminAccountRepo) ListSchedulable(ctx context.Context) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) ListSchedulableByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) ListSchedulableUngroupedByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	return nil, nil
}
func (m *passthroughAdminAccountRepo) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	return nil
}
func (m *passthroughAdminAccountRepo) SetModelRateLimit(ctx context.Context, id int64, scope string, resetAt time.Time) error {
	return nil
}
func (m *passthroughAdminAccountRepo) SetOverloaded(ctx context.Context, id int64, until time.Time) error {
	return nil
}
func (m *passthroughAdminAccountRepo) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	return nil
}
func (m *passthroughAdminAccountRepo) ClearTempUnschedulable(ctx context.Context, id int64) error {
	return nil
}
func (m *passthroughAdminAccountRepo) ClearRateLimit(ctx context.Context, id int64) error { return nil }
func (m *passthroughAdminAccountRepo) ClearAntigravityQuotaScopes(ctx context.Context, id int64) error {
	return nil
}
func (m *passthroughAdminAccountRepo) ClearModelRateLimits(ctx context.Context, id int64) error {
	return nil
}
func (m *passthroughAdminAccountRepo) UpdateSessionWindow(ctx context.Context, id int64, start, end *time.Time, status string) error {
	return nil
}
func (m *passthroughAdminAccountRepo) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	return nil
}
func (m *passthroughAdminAccountRepo) BulkUpdate(ctx context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	clone := updates
	if updates.Credentials != nil {
		clone.Credentials = cloneMapForPassthroughTest(updates.Credentials)
	}
	if updates.Extra != nil {
		clone.Extra = cloneMapForPassthroughTest(updates.Extra)
	}
	m.bulkUpdated = &clone
	m.bulkIDs = append([]int64{}, ids...)
	return 0, nil
}
func (m *passthroughAdminAccountRepo) IncrementQuotaUsed(ctx context.Context, id int64, amount float64) error {
	return nil
}
func (m *passthroughAdminAccountRepo) ResetQuotaUsed(ctx context.Context, id int64) error { return nil }

var _ AccountRepository = (*passthroughAdminAccountRepo)(nil)

type passthroughAdminGroupRepo struct{}

func (m *passthroughAdminGroupRepo) Create(ctx context.Context, group *Group) error { return nil }
func (m *passthroughAdminGroupRepo) GetByID(ctx context.Context, id int64) (*Group, error) {
	return nil, errors.New("group not found")
}
func (m *passthroughAdminGroupRepo) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	return nil, errors.New("group not found")
}
func (m *passthroughAdminGroupRepo) Update(ctx context.Context, group *Group) error { return nil }
func (m *passthroughAdminGroupRepo) Delete(ctx context.Context, id int64) error     { return nil }
func (m *passthroughAdminGroupRepo) DeleteCascade(ctx context.Context, id int64) ([]int64, error) {
	return nil, nil
}
func (m *passthroughAdminGroupRepo) List(ctx context.Context, params pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *passthroughAdminGroupRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, status, search string, isExclusive *bool) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *passthroughAdminGroupRepo) ListActive(ctx context.Context) ([]Group, error) { return nil, nil }
func (m *passthroughAdminGroupRepo) ListActiveByPlatform(ctx context.Context, platform string) ([]Group, error) {
	return nil, nil
}
func (m *passthroughAdminGroupRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	return false, nil
}
func (m *passthroughAdminGroupRepo) GetAccountCount(ctx context.Context, groupID int64) (int64, int64, error) {
	return 0, 0, nil
}
func (m *passthroughAdminGroupRepo) DeleteAccountGroupsByGroupID(ctx context.Context, groupID int64) (int64, error) {
	return 0, nil
}
func (m *passthroughAdminGroupRepo) GetAccountIDsByGroupIDs(ctx context.Context, groupIDs []int64) ([]int64, error) {
	return nil, nil
}
func (m *passthroughAdminGroupRepo) BindAccountsToGroup(ctx context.Context, groupID int64, accountIDs []int64) error {
	return nil
}
func (m *passthroughAdminGroupRepo) UpdateSortOrders(ctx context.Context, updates []GroupSortOrderUpdate) error {
	return nil
}

var _ GroupRepository = (*passthroughAdminGroupRepo)(nil)

func cloneAccountForPassthroughTest(account *Account) *Account {
	if account == nil {
		return nil
	}
	clone := *account
	clone.Extra = cloneMapForPassthroughTest(account.Extra)
	clone.Credentials = cloneMapForPassthroughTest(account.Credentials)
	return &clone
}

func cloneMapForPassthroughTest(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for k, v := range input {
		switch typed := v.(type) {
		case []any:
			copied := make([]any, len(typed))
			for i, item := range typed {
				if m, ok := item.(map[string]any); ok {
					copied[i] = cloneMapForPassthroughTest(m)
				} else {
					copied[i] = item
				}
			}
			cloned[k] = copied
		case map[string]any:
			cloned[k] = cloneMapForPassthroughTest(typed)
		default:
			cloned[k] = v
		}
	}
	return cloned
}

func TestAdminServiceCreateAccountPassthrough_SavesAPIKeyRules(t *testing.T) {
	repo := &passthroughAdminAccountRepo{}
	service := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &passthroughAdminGroupRepo{},
	}

	account, err := service.CreateAccount(context.Background(), &CreateAccountInput{
		Name:        "api key",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []any{
				map[string]any{"target": "header", "mode": "inject", "key": "X-Env", "value": "prod"},
			},
		},
		Concurrency:          1,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.NotNil(t, repo.created)
	require.Equal(t, true, repo.created.Extra["passthrough_fields_enabled"])
	require.Equal(t, []PassthroughFieldRule{{Target: "header", Mode: "inject", Key: "X-Env", Value: "prod"}}, repo.created.Extra["passthrough_field_rules"])
}

func TestAdminServiceUpdateAccountPassthrough_RemovesRulesWhenTypeChanges(t *testing.T) {
	repo := &passthroughAdminAccountRepo{
		accountsByID: map[int64]*Account{
			1: {
				ID:       1,
				Name:     "api key",
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Status:   StatusActive,
				Extra: map[string]any{
					"passthrough_fields_enabled": true,
					"passthrough_field_rules": []any{
						map[string]any{"target": "header", "mode": "forward", "key": "X-Test"},
					},
					"other": "keep",
				},
			},
		},
	}
	service := &adminServiceImpl{accountRepo: repo, groupRepo: &passthroughAdminGroupRepo{}}

	updated, err := service.UpdateAccount(context.Background(), 1, &UpdateAccountInput{
		Type: AccountTypeOAuth,
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	require.NotNil(t, repo.updated)
	require.Equal(t, AccountTypeOAuth, repo.updated.Type)
	require.Equal(t, "keep", repo.updated.Extra["other"])
	require.NotContains(t, repo.updated.Extra, "passthrough_fields_enabled")
	require.NotContains(t, repo.updated.Extra, "passthrough_field_rules")
	_, enabled := updated.Extra["passthrough_fields_enabled"]
	require.False(t, enabled)
}

func TestAdminServiceUpdateAccountPassthrough_RemovesRulesForNonAPIKeyWithoutExplicitExtra(t *testing.T) {
	repo := &passthroughAdminAccountRepo{
		accountsByID: map[int64]*Account{
			1: {
				ID:       1,
				Name:     "oauth",
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Status:   StatusActive,
				Extra: map[string]any{
					"passthrough_fields_enabled": true,
					"passthrough_field_rules": []any{
						map[string]any{"target": "body", "mode": "forward", "key": "metadata.user_id"},
					},
					"other": "keep",
				},
			},
		},
	}
	service := &adminServiceImpl{accountRepo: repo, groupRepo: &passthroughAdminGroupRepo{}}

	updated, err := service.UpdateAccount(context.Background(), 1, &UpdateAccountInput{
		Name: "oauth-updated",
	})

	require.NoError(t, err)
	require.NotNil(t, repo.updated)
	require.Equal(t, "keep", repo.updated.Extra["other"])
	require.NotContains(t, repo.updated.Extra, "passthrough_fields_enabled")
	require.NotContains(t, repo.updated.Extra, "passthrough_field_rules")
	require.NotContains(t, updated.Extra, "passthrough_fields_enabled")
	require.NotContains(t, updated.Extra, "passthrough_field_rules")
}

func TestAdminServiceCreateAccountPassthrough_RejectsNonAPIKeySubmission(t *testing.T) {
	repo := &passthroughAdminAccountRepo{}
	service := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &passthroughAdminGroupRepo{},
	}

	_, err := service.CreateAccount(context.Background(), &CreateAccountInput{
		Name:        "oauth",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{"access_token": "token"},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
		},
		Concurrency:          1,
		SkipDefaultGroupBind: true,
	})

	require.EqualError(t, err, "passthrough field rules are only supported for apikey accounts")
	require.Nil(t, repo.created)
}

func TestAdminServiceCreateAccountPassthrough_ErrorIncludesConflictingField(t *testing.T) {
	repo := &passthroughAdminAccountRepo{}
	service := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &passthroughAdminGroupRepo{},
	}

	_, err := service.CreateAccount(context.Background(), &CreateAccountInput{
		Name:        "api key",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []any{
				map[string]any{"target": "header", "mode": "forward", "key": "X-Test"},
				map[string]any{"target": "header", "mode": "inject", "key": "x-test", "value": "prod"},
			},
		},
		Concurrency:          1,
		SkipDefaultGroupBind: true,
	})

	require.ErrorContains(t, err, "x-test")
	require.Nil(t, repo.created)
}

func TestAdminServiceBulkUpdatePassthrough_RejectsNonAPIKeyAccounts(t *testing.T) {
	repo := &passthroughAdminAccountRepo{
		accountsByID: map[int64]*Account{
			1: {
				ID:       1,
				Name:     "oauth",
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Status:   StatusActive,
			},
		},
	}
	service := &adminServiceImpl{accountRepo: repo, groupRepo: &passthroughAdminGroupRepo{}}

	_, err := service.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []any{
				map[string]any{"target": "header", "mode": "inject", "key": "X-Env", "value": "prod"},
			},
		},
	})

	require.EqualError(t, err, "passthrough field rules are only supported for apikey accounts")
	require.Nil(t, repo.bulkUpdated)
	require.Nil(t, repo.updated)
}

func TestAdminServiceBulkUpdatePassthrough_NormalizesAPIKeyAccounts(t *testing.T) {
	repo := &passthroughAdminAccountRepo{
		accountsByID: map[int64]*Account{
			1: {
				ID:       1,
				Name:     "apikey",
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Status:   StatusActive,
				Extra: map[string]any{
					"other": "keep",
				},
			},
		},
	}
	service := &adminServiceImpl{accountRepo: repo, groupRepo: &passthroughAdminGroupRepo{}}

	result, err := service.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []any{
				map[string]any{"target": "header", "mode": "inject", "key": "X-Env", "value": "prod"},
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, repo.updated)
	require.Equal(t, "keep", repo.updated.Extra["other"])
	require.Equal(t, true, repo.updated.Extra["passthrough_fields_enabled"])
	require.Equal(t, []PassthroughFieldRule{{Target: "header", Mode: "inject", Key: "X-Env", Value: "prod"}}, repo.updated.Extra["passthrough_field_rules"])
	require.Nil(t, repo.bulkUpdated)
}

func TestAdminServiceBulkUpdatePassthrough_PrevalidatesAllAccountsBeforeWriting(t *testing.T) {
	repo := &passthroughAdminAccountRepo{
		accountsByID: map[int64]*Account{
			1: {
				ID:       1,
				Name:     "apikey",
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Status:   StatusActive,
			},
			2: {
				ID:       2,
				Name:     "oauth",
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Status:   StatusActive,
			},
		},
	}
	service := &adminServiceImpl{accountRepo: repo, groupRepo: &passthroughAdminGroupRepo{}}

	_, err := service.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{1, 2},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []any{
				map[string]any{"target": "header", "mode": "inject", "key": "X-Env", "value": "prod"},
			},
		},
	})

	require.EqualError(t, err, "passthrough field rules are only supported for apikey accounts")
	require.Empty(t, repo.updatedIDs)
	require.Equal(t, "apikey", repo.accountsByID[1].Name)
	require.NotContains(t, repo.accountsByID[1].Extra, "passthrough_fields_enabled")
}
