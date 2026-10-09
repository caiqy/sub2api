//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestPlatformWriteValidationAccounts(t *testing.T) {
	for _, platform := range append(domain.ConcretePlatformIDs(), "bogus", PlatformComposite, "", "openai ") {
		t.Run(platform, func(t *testing.T) {
			valid := domain.IsConcretePlatform(platform)
			account, err := buildAccountForCreate(&CreateAccountInput{Platform: platform, Type: AccountTypeAPIKey}, nil)
			if valid {
				require.NoError(t, err)
				require.Equal(t, platform, account.Platform)
			} else {
				require.Error(t, err)
			}
			for _, admin := range []bool{true, false} {
				repo := &upstreamBillingProbeAccountRepo{}
				if admin {
					account, err = (&adminServiceImpl{accountRepo: repo}).CreateAccount(context.Background(), &CreateAccountInput{
						Name: "a", Platform: platform, Type: AccountTypeAPIKey, SkipDefaultGroupBind: true,
					})
				} else {
					account, err = NewAccountService(repo, nil).Create(context.Background(), CreateAccountRequest{
						Name: "a", Platform: platform, Type: AccountTypeAPIKey,
					})
				}
				if valid {
					require.NoError(t, err)
					require.Equal(t, platform, account.Platform)
					require.Len(t, repo.accounts, 1)
				} else {
					require.Error(t, err)
					require.Equal(t, "INVALID_ACCOUNT_PLATFORM", infraerrors.Reason(err))
					require.Empty(t, repo.accounts)
				}
			}
		})
	}
}

func TestPlatformWriteValidationGroups(t *testing.T) {
	for _, platform := range []string{"", PlatformAnthropic, PlatformComposite, PlatformCline, PlatformCommandCode, "bogus", "openai "} {
		t.Run(platform, func(t *testing.T) {
			repo := &groupRepoStubForAdmin{getByID: &Group{ID: 1, Name: "before", Platform: PlatformCline}}
			svc := &adminServiceImpl{groupRepo: repo}
			group, err := svc.CreateGroup(context.Background(), &CreateGroupInput{Name: "g", Platform: platform, RateMultiplier: 1})
			valid := domain.IsGroupPlatform(NormalizeGroupPlatform(platform))
			if valid {
				require.NoError(t, err)
				require.Equal(t, NormalizeGroupPlatform(platform), group.Platform)
			} else {
				require.Error(t, err)
				require.Equal(t, "INVALID_GROUP_PLATFORM", infraerrors.Reason(err))
				require.Nil(t, repo.created)
			}
			group, err = svc.UpdateGroup(context.Background(), 1, &UpdateGroupInput{Name: "after", Platform: platform})
			if valid {
				require.NoError(t, err)
				expected := platform
				if expected == "" {
					expected = PlatformCline
				}
				require.Equal(t, expected, group.Platform)
			} else {
				require.Error(t, err)
				require.Equal(t, "INVALID_GROUP_PLATFORM", infraerrors.Reason(err))
				require.Nil(t, repo.updated)
				require.Equal(t, "before", repo.getByID.Name)
				require.Equal(t, PlatformCline, repo.getByID.Platform)
			}
		})
	}
}
