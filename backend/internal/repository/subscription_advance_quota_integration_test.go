//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type refreshReadForbiddenSubscriptionRepo struct {
	service.UserSubscriptionRepository
	locking interface {
		GetByIDForUpdate(context.Context, int64) (*service.UserSubscription, error)
		GetByUserIDAndGroupIDForUpdate(context.Context, int64, int64) (*service.UserSubscription, error)
	}
}

func (r *refreshReadForbiddenSubscriptionRepo) GetByIDForUpdate(ctx context.Context, id int64) (*service.UserSubscription, error) {
	return r.locking.GetByIDForUpdate(ctx, id)
}

func (r *refreshReadForbiddenSubscriptionRepo) GetByUserIDAndGroupIDForUpdate(ctx context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	return r.locking.GetByUserIDAndGroupIDForUpdate(ctx, userID, groupID)
}

func (r *refreshReadForbiddenSubscriptionRepo) GetByID(context.Context, int64) (*service.UserSubscription, error) {
	return nil, errors.New("unexpected refresh GetByID after Update")
}

// firstLockReadBarrierRepo wraps the real repository and parks the FIRST
// GetByIDForUpdate call until the test releases it. The barrier makes the
// interleaving deterministic while the locked read is held.
type firstLockReadBarrierRepo struct {
	service.UserSubscriptionRepository
	base          *userSubscriptionRepository
	blocked       chan struct{}
	release       chan struct{}
	blockOnce     sync.Once
	plainGetByID  atomic.Int32
	lockReadCalls atomic.Int32
}

func newFirstLockReadBarrierRepo(base *userSubscriptionRepository) *firstLockReadBarrierRepo {
	return &firstLockReadBarrierRepo{
		UserSubscriptionRepository: base,
		base:                       base,
		blocked:                    make(chan struct{}),
		release:                    make(chan struct{}),
	}
}

func (r *firstLockReadBarrierRepo) GetByID(ctx context.Context, id int64) (*service.UserSubscription, error) {
	r.plainGetByID.Add(1)
	return r.UserSubscriptionRepository.GetByID(ctx, id)
}

func (r *firstLockReadBarrierRepo) GetByIDForUpdate(ctx context.Context, id int64) (*service.UserSubscription, error) {
	if r.lockReadCalls.Add(1) == 1 {
		r.blockOnce.Do(func() { close(r.blocked) })
		<-r.release
	}
	return r.base.GetByIDForUpdate(ctx, id)
}

// waitBlocked waits until the first GetByIDForUpdate is parked at the barrier.
func (r *firstLockReadBarrierRepo) waitBlocked(t *testing.T) {
	t.Helper()
	select {
	case <-r.blocked:
	case <-time.After(10 * time.Second):
		t.Fatalf("first GetByIDForUpdate never blocked: unlocked GetByID calls=%d lockRead calls=%d (current code bypasses the row lock)",
			r.plainGetByID.Load(), r.lockReadCalls.Load())
	}
}

// TestExtendSubscription_SerializedWithQuotaAdvance proves that ExtendSubscription
// parks on the row lock taken inside its transaction: after the quota advance
// commits its deduction, the extend computes from the advanced expiry, so the
// final expires_at contains BOTH the deduction and the extension. Without the
// lock discipline the extend reads a stale absolute expiry and overwrites the
// deduction (RED).
func TestExtendSubscription_SerializedWithQuotaAdvance(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	suffix := time.Now().UnixNano()
	user, err := client.User.Create().
		SetEmail(fmt.Sprintf("extend-advance-%d@test.com", suffix)).
		SetPasswordHash("test-password-hash").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName(fmt.Sprintf("extend-advance-%d", suffix)).
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetDailyLimitUsd(10).
		Save(ctx)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	dailyStart := now.Add(-4 * time.Hour)
	originalExpiry := now.Add(30 * 24 * time.Hour)
	sub, err := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(now.Add(-10 * 24 * time.Hour)).
		SetExpiresAt(originalExpiry).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetDailyWindowStart(dailyStart).
		SetDailyUsageUsd(10).
		SetNotes("").
		Save(ctx)
	require.NoError(t, err)

	baseRepo := NewUserSubscriptionRepository(client)
	barrier := newFirstLockReadBarrierRepo(baseRepo.(*userSubscriptionRepository))
	svc := service.NewSubscriptionService(
		NewGroupRepository(client, integrationDB),
		barrier,
		nil,
		client,
		nil,
	)
	defer svc.Stop()

	extendDone := make(chan error, 1)
	go func() {
		_, callErr := svc.ExtendSubscription(ctx, sub.ID, 10)
		extendDone <- callErr
	}()

	// The extend must park on its transaction row lock BEFORE reading.
	barrier.waitBlocked(t)
	close(barrier.release)

	// The quota advance runs after the extend's locked read has completed its
	// own serialized write; both effects must persist.
	_, err = svc.AdvanceQuotaCycle(ctx, user.ID, sub.ID, service.QuotaWindowSelection{Daily: true})
	require.NoError(t, err)
	require.NoError(t, <-extendDone)

	require.Zero(t, barrier.plainGetByID.Load(), "ExtendSubscription must not use the unlocked GetByID read")

	stored, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	expected := originalExpiry.AddDate(0, 0, 10).Add(-20 * time.Hour)
	require.WithinDuration(t, expected, stored.ExpiresAt, 2*time.Second,
		"final expiry must include both the 10-day extension and the ~20h cycle deduction")
}

// TestExtendSubscription_TwoConcurrentExtendsBothApply proves that two
// concurrent extensions serialize on the row lock and both accumulate. The
// first locks and parks at the barrier; the second passes and completes; after
// release the first re-reads the extended expiry and adds its own days.
func TestExtendSubscription_TwoConcurrentExtendsBothApply(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	suffix := time.Now().UnixNano()
	user, err := client.User.Create().
		SetEmail(fmt.Sprintf("extend-extend-%d@test.com", suffix)).
		SetPasswordHash("test-password-hash").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName(fmt.Sprintf("extend-extend-%d", suffix)).
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		Save(ctx)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	originalExpiry := now.Add(30 * 24 * time.Hour)
	sub, err := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(now.Add(-10 * 24 * time.Hour)).
		SetExpiresAt(originalExpiry).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetDailyWindowStart(now.Add(-time.Hour)).
		SetNotes("").
		Save(ctx)
	require.NoError(t, err)

	baseRepo := NewUserSubscriptionRepository(client)
	barrier := newFirstLockReadBarrierRepo(baseRepo.(*userSubscriptionRepository))
	svc := service.NewSubscriptionService(
		NewGroupRepository(client, integrationDB),
		barrier,
		nil,
		client,
		nil,
	)
	defer svc.Stop()

	done := make(chan error, 2)
	for range 2 {
		go func() {
			_, callErr := svc.ExtendSubscription(ctx, sub.ID, 10)
			done <- callErr
		}()
	}

	barrier.waitBlocked(t)
	close(barrier.release)

	for range 2 {
		require.NoError(t, <-done)
	}

	stored, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	expected := originalExpiry.AddDate(0, 0, 20)
	require.WithinDuration(t, expected, stored.ExpiresAt, 2*time.Second,
		"both concurrent 10-day extensions must accumulate")
}

func TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	suffix := time.Now().UnixNano()
	user, err := client.User.Create().
		SetEmail(fmt.Sprintf("quota-advance-%d@test.com", suffix)).
		SetPasswordHash("test-password-hash").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName(fmt.Sprintf("quota-advance-%d", suffix)).
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetDailyLimitUsd(10).
		Save(ctx)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	dailyStart := now.Add(-4 * time.Hour)
	originalExpiry := now.Add(30 * 24 * time.Hour)
	sub, err := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(now.Add(-10 * 24 * time.Hour)).
		SetExpiresAt(originalExpiry).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetDailyWindowStart(dailyStart).
		SetDailyUsageUsd(10).
		SetNotes("").
		Save(ctx)
	require.NoError(t, err)

	svc := service.NewSubscriptionService(
		NewGroupRepository(client, integrationDB),
		NewUserSubscriptionRepository(client),
		nil,
		client,
		nil,
	)
	defer svc.Stop()

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, callErr := svc.AdvanceQuotaCycle(ctx, user.ID, sub.ID, service.QuotaWindowSelection{Daily: true})
			errs <- callErr
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	var success, rejected int
	for callErr := range errs {
		if callErr == nil {
			success++
		} else if service.ErrQuotaAdvanceStateChanged.Is(callErr) {
			rejected++
		} else {
			t.Fatalf("unexpected concurrent result: %v", callErr)
		}
	}
	require.Equal(t, 1, success)
	require.Equal(t, 1, rejected)

	stored, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	require.Zero(t, stored.DailyUsageUsd)
	require.NotNil(t, stored.DailyWindowStart)
	require.WithinDuration(t, time.Now().UTC(), *stored.DailyWindowStart, 5*time.Second)
	require.True(t, stored.ExpiresAt.Before(originalExpiry.Add(-19*time.Hour)))
	require.True(t, stored.ExpiresAt.After(originalExpiry.Add(-21*time.Hour)))
}

func TestAdvanceQuotaCycle_HidesSubscriptionOwnedByAnotherUser(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	suffix := time.Now().UnixNano()
	owner, err := client.User.Create().
		SetEmail(fmt.Sprintf("quota-owner-%d@test.com", suffix)).
		SetPasswordHash("test-password-hash").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName(fmt.Sprintf("quota-owner-%d", suffix)).
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetDailyLimitUsd(10).
		Save(ctx)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	windowStart := now.Add(-time.Hour)
	sub, err := client.UserSubscription.Create().
		SetUserID(owner.ID).
		SetGroupID(group.ID).
		SetStartsAt(now.Add(-24 * time.Hour)).
		SetExpiresAt(now.Add(10 * 24 * time.Hour)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetDailyWindowStart(windowStart).
		SetDailyUsageUsd(10).
		SetNotes("").
		Save(ctx)
	require.NoError(t, err)

	svc := service.NewSubscriptionService(NewGroupRepository(client, integrationDB), NewUserSubscriptionRepository(client), nil, client, nil)
	defer svc.Stop()
	_, err = svc.AdvanceQuotaCycle(ctx, owner.ID+1, sub.ID, service.QuotaWindowSelection{Daily: true})

	require.ErrorIs(t, err, service.ErrSubscriptionNotFound)
	stored, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	require.Equal(t, float64(10), stored.DailyUsageUsd)
}

func TestAdvanceQuotaCycle_CommitsAuthoritativeUpdatedObjectWithoutRefreshRead(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	suffix := time.Now().UnixNano()
	user, err := client.User.Create().
		SetEmail(fmt.Sprintf("quota-commit-%d@test.com", suffix)).
		SetPasswordHash("test-password-hash").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName(fmt.Sprintf("quota-commit-%d", suffix)).
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetDailyLimitUsd(10).
		Save(ctx)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	windowStart := now.Add(-time.Hour)
	sub, err := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(now.Add(-24 * time.Hour)).
		SetExpiresAt(now.Add(10 * 24 * time.Hour)).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetDailyWindowStart(windowStart).
		SetDailyUsageUsd(10).
		SetNotes("").
		Save(ctx)
	require.NoError(t, err)

	baseRepo := NewUserSubscriptionRepository(client)
	repo := &refreshReadForbiddenSubscriptionRepo{
		UserSubscriptionRepository: baseRepo,
		locking: baseRepo.(interface {
			GetByIDForUpdate(context.Context, int64) (*service.UserSubscription, error)
			GetByUserIDAndGroupIDForUpdate(context.Context, int64, int64) (*service.UserSubscription, error)
		}),
	}
	svc := service.NewSubscriptionService(NewGroupRepository(client, integrationDB), repo, nil, client, nil)
	defer svc.Stop()

	result, err := svc.AdvanceQuotaCycle(ctx, user.ID, sub.ID, service.QuotaWindowSelection{Daily: true})

	require.NoError(t, err)
	require.NotNil(t, result.Subscription)
	require.Zero(t, result.Subscription.DailyUsageUSD)
	require.NotNil(t, result.Subscription.User)
	require.Equal(t, user.ID, result.Subscription.User.ID)
	require.NotNil(t, result.Subscription.Group)
	require.Equal(t, group.ID, result.Subscription.Group.ID)
	stored, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	require.Zero(t, stored.DailyUsageUsd)
	require.Equal(t, stored.UpdatedAt.UnixNano(), result.Subscription.UpdatedAt.UnixNano())
}
