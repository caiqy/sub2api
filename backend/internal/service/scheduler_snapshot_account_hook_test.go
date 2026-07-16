package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type schedulerAccountHookCache struct {
	*outboxCleanupCache
	setErr error
}

func (c *schedulerAccountHookCache) SetSnapshot(context.Context, SchedulerBucket, SchedulerBucketWriteToken, []Account) error {
	return c.setErr
}

func TestSchedulerSnapshotAccountChangeHookRunsOnlyAfterSuccessfulOpenAIRebuild(t *testing.T) {
	account := Account{
		ID:          71,
		Platform:    PlatformOpenAI,
		Status:      StatusActive,
		Schedulable: true,
		GroupIDs:    []int64{1},
	}

	for name, setErr := range map[string]error{
		"successful rebuild": nil,
		"failed rebuild":     errors.New("snapshot write failed"),
	} {
		t.Run(name, func(t *testing.T) {
			cache := &schedulerAccountHookCache{
				outboxCleanupCache: &outboxCleanupCache{},
				setErr:             setErr,
			}
			repo := schedulerTestOpenAIAccountRepo{accounts: []Account{account}}
			svc := NewSchedulerSnapshotService(cache, nil, repo, nil, &config.Config{RunMode: config.RunModeSimple})
			var called []int64
			svc.SetOpenAIAccountChangeHandler(func(_ context.Context, accountID int64) {
				called = append(called, accountID)
			})

			accountID := account.ID
			err := svc.handleAccountEvent(context.Background(), &accountID, nil, map[batchSeenKey]struct{}{})
			if setErr != nil {
				require.ErrorIs(t, err, setErr)
				require.Empty(t, called)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []int64{account.ID}, called)
		})
	}
}
