//go:build unit

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

// termReadWriteEvent records a repository call made by a subscription term
// mutation path: which method ran and whether its context carried a
// transaction. Tests use these to prove lock/tx discipline.
type termReadWriteEvent struct {
	name string
	inTx bool
}

func (e termReadWriteEvent) String() string {
	tx := "no-tx"
	if e.inTx {
		tx = "in-tx"
	}
	return e.name + "(" + tx + ")"
}

// termLockingUserSubRepo is an in-memory UserSubscriptionRepository that
// records every read/write event and honors the locking-method shapes of the
// real repository.
type termLockingUserSubRepo struct {
	userSubRepoNoop
	mu     sync.Mutex
	events []termReadWriteEvent

	byID          map[int64]*UserSubscription
	byUserGroup   map[string]*UserSubscription
	nextID        int64
	updateVersion int64
}

func newTermLockingUserSubRepo() *termLockingUserSubRepo {
	return &termLockingUserSubRepo{
		byID:        make(map[int64]*UserSubscription),
		byUserGroup: make(map[string]*UserSubscription),
		nextID:      1,
	}
}

func (r *termLockingUserSubRepo) key(userID, groupID int64) string {
	return strconvFormatInt(userID) + ":" + strconvFormatInt(groupID)
}

func (r *termLockingUserSubRepo) record(name string, ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, termReadWriteEvent{name: name, inTx: dbent.TxFromContext(ctx) != nil})
}

func (r *termLockingUserSubRepo) snapshotEvents() []termReadWriteEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]termReadWriteEvent, len(r.events))
	copy(out, r.events)
	return out
}

func (r *termLockingUserSubRepo) seed(sub *UserSubscription) {
	if sub == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *sub
	if cp.ID == 0 {
		cp.ID = r.nextID
		r.nextID++
	}
	r.byID[cp.ID] = &cp
	r.byUserGroup[r.key(cp.UserID, cp.GroupID)] = &cp
}

func (r *termLockingUserSubRepo) lookupByID(id int64) (*UserSubscription, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.byID[id]
	if !ok {
		return nil, false
	}
	cp := *sub
	return &cp, true
}

func (r *termLockingUserSubRepo) lookupByUserGroup(userID, groupID int64) (*UserSubscription, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.byUserGroup[r.key(userID, groupID)]
	if !ok {
		return nil, false
	}
	cp := *sub
	return &cp, true
}

func (r *termLockingUserSubRepo) GetByID(ctx context.Context, id int64) (*UserSubscription, error) {
	r.record("GetByID", ctx)
	sub, ok := r.lookupByID(id)
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

func (r *termLockingUserSubRepo) GetByIDForUpdate(ctx context.Context, id int64) (*UserSubscription, error) {
	r.record("GetByIDForUpdate", ctx)
	sub, ok := r.lookupByID(id)
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

func (r *termLockingUserSubRepo) GetByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	r.record("GetByUserIDAndGroupID", ctx)
	sub, ok := r.lookupByUserGroup(userID, groupID)
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

func (r *termLockingUserSubRepo) GetByUserIDAndGroupIDForUpdate(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	r.record("GetByUserIDAndGroupIDForUpdate", ctx)
	sub, ok := r.lookupByUserGroup(userID, groupID)
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

func (r *termLockingUserSubRepo) Update(ctx context.Context, sub *UserSubscription) error {
	r.record("Update", ctx)
	if sub == nil {
		return ErrSubscriptionNilInput
	}
	if r.updateVersion > 0 {
		sub.UpdatedAt = time.Unix(0, r.updateVersion)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.byID[sub.ID]
	if !ok {
		return ErrSubscriptionNotFound
	}
	delete(r.byUserGroup, r.key(existing.UserID, existing.GroupID))
	cp := *sub
	r.byID[cp.ID] = &cp
	r.byUserGroup[r.key(cp.UserID, cp.GroupID)] = &cp
	return nil
}

func (r *termLockingUserSubRepo) ExtendExpiry(ctx context.Context, subscriptionID int64, newExpiresAt time.Time) error {
	r.record("ExtendExpiry", ctx)
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.byID[subscriptionID]
	if !ok {
		return ErrSubscriptionNotFound
	}
	cp := *sub
	cp.ExpiresAt = newExpiresAt
	r.byID[subscriptionID] = &cp
	r.byUserGroup[r.key(cp.UserID, cp.GroupID)] = &cp
	return nil
}

func (r *termLockingUserSubRepo) UpdateStatus(ctx context.Context, subscriptionID int64, status string) error {
	r.record("UpdateStatus", ctx)
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.byID[subscriptionID]
	if !ok {
		return ErrSubscriptionNotFound
	}
	cp := *sub
	cp.Status = status
	r.byID[subscriptionID] = &cp
	r.byUserGroup[r.key(cp.UserID, cp.GroupID)] = &cp
	return nil
}

func (r *termLockingUserSubRepo) UpdateNotes(ctx context.Context, subscriptionID int64, notes string) error {
	r.record("UpdateNotes", ctx)
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.byID[subscriptionID]
	if !ok {
		return ErrSubscriptionNotFound
	}
	cp := *sub
	cp.Notes = notes
	r.byID[subscriptionID] = &cp
	r.byUserGroup[r.key(cp.UserID, cp.GroupID)] = &cp
	return nil
}

func (r *termLockingUserSubRepo) ExistsByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (bool, error) {
	r.record("ExistsByUserIDAndGroupID", ctx)
	_, ok := r.lookupByUserGroup(userID, groupID)
	return ok, nil
}

func (r *termLockingUserSubRepo) Create(ctx context.Context, sub *UserSubscription) error {
	r.record("Create", ctx)
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *sub
	if cp.ID == 0 {
		cp.ID = r.nextID
		r.nextID++
	}
	sub.ID = cp.ID
	r.byID[cp.ID] = &cp
	r.byUserGroup[r.key(cp.UserID, cp.GroupID)] = &cp
	return nil
}

// assertLockedReadDiscipline asserts that before the first FOR UPDATE read of
// the row only a pure existence probe is allowed, that the locked read runs
// inside a transaction, and that no unlocked content read of the row happens
// before it (reads after the lock, inside the same transaction, are allowed —
// the row lock is held until commit).
func assertLockedReadDiscipline(t *testing.T, events []termReadWriteEvent, lockedName, plainName string) {
	t.Helper()
	firstLocked := -1
	for i, e := range events {
		if e.name == lockedName {
			firstLocked = i
			break
		}
	}
	require.GreaterOrEqual(t, firstLocked, 0, "path never performed the FOR UPDATE read %s (events=%v)", lockedName, events)
	require.True(t, events[firstLocked].inTx, "locked read must run inside the subscription update transaction")
	for i, e := range events {
		if i >= firstLocked {
			break
		}
		if e.name == plainName {
			t.Fatalf("path used unlocked read %s before the locked read %s", e, lockedName)
		}
		if e.name != "ExistsByUserIDAndGroupID" {
			t.Fatalf("unexpected read %s before the locked read %s", e, lockedName)
		}
	}
}

func TestExtendSubscription_LockReadAndWriteInsideTransaction(t *testing.T) {
	tx := &dbent.Tx{}
	ctx := dbent.NewTxContext(context.Background(), tx)
	repo := newTermLockingUserSubRepo()
	now := time.Now()
	repo.seed(&UserSubscription{
		ID: 1, UserID: 10, GroupID: 20,
		StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(30 * 24 * time.Hour),
		Status: SubscriptionStatusActive,
	})
	svc := &SubscriptionService{userSubRepo: repo}

	sub, err := svc.ExtendSubscription(ctx, 1, 10)

	require.NoError(t, err)
	require.Equal(t, now.Add(30*24*time.Hour).AddDate(0, 0, 10), sub.ExpiresAt)

	events := repo.snapshotEvents()
	assertLockedReadDiscipline(t, events, "GetByIDForUpdate", "GetByID")
	require.Contains(t, events, termReadWriteEvent{name: "ExtendExpiry", inTx: true}, "expiry write must run inside the same transaction")
}

func TestExtendSubscription_ExpiredRenewalWriteInsideTransaction(t *testing.T) {
	client := newPaymentOrderLifecycleTestClient(t)
	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	ctx := dbent.NewTxContext(context.Background(), tx)
	repo := newTermLockingUserSubRepo()
	now := time.Now()
	repo.seed(&UserSubscription{
		ID: 1, UserID: 10, GroupID: 20,
		StartsAt: now.AddDate(0, 0, -40), ExpiresAt: now.Add(-time.Hour),
		Status: SubscriptionStatusExpired,
	})
	svc := &SubscriptionService{userSubRepo: repo}

	sub, err := svc.ExtendSubscription(ctx, 1, 10)

	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusActive, sub.Status)

	events := repo.snapshotEvents()
	assertLockedReadDiscipline(t, events, "GetByIDForUpdate", "GetByID")
	require.Contains(t, events, termReadWriteEvent{name: "Update", inTx: true}, "renewal write must run inside the same transaction")
}

func TestAssignOrExtendSubscription_LockedReadInsideTransaction(t *testing.T) {
	tx := &dbent.Tx{}
	ctx := dbent.NewTxContext(context.Background(), tx)
	repo := newTermLockingUserSubRepo()
	now := time.Now()
	repo.seed(&UserSubscription{
		ID: 1, UserID: 10, GroupID: 20,
		StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(30 * 24 * time.Hour),
		Status: SubscriptionStatusActive,
	})
	svc := &SubscriptionService{
		groupRepo:   &subscriptionGroupRepoStub{group: &Group{ID: 20, SubscriptionType: SubscriptionTypeSubscription}},
		userSubRepo: repo,
	}

	sub, reused, _, err := svc.assignOrExtendSubscription(ctx, &AssignSubscriptionInput{
		UserID: 10, GroupID: 20, ValidityDays: 10, Notes: "renew",
	}, true)

	require.NoError(t, err)
	require.True(t, reused)
	require.Equal(t, now.Add(30*24*time.Hour).AddDate(0, 0, 10), sub.ExpiresAt)

	events := repo.snapshotEvents()
	assertLockedReadDiscipline(t, events, "GetByUserIDAndGroupIDForUpdate", "GetByUserIDAndGroupID")
	require.Contains(t, events, termReadWriteEvent{name: "ExtendExpiry", inTx: true}, "existing subscription renewal must run inside the same transaction")
}

func TestAssignSubscriptionWithReuse_LockedReadInsideTransaction(t *testing.T) {
	tx := &dbent.Tx{}
	ctx := dbent.NewTxContext(context.Background(), tx)
	repo := newTermLockingUserSubRepo()
	start := time.Now().Add(-time.Hour)
	repo.seed(&UserSubscription{
		ID: 1, UserID: 10, GroupID: 20,
		StartsAt: start, ExpiresAt: start.AddDate(0, 0, 30),
		Status: SubscriptionStatusActive,
		Notes:  "init",
	})
	svc := &SubscriptionService{
		groupRepo:   &subscriptionGroupRepoStub{group: &Group{ID: 20, SubscriptionType: SubscriptionTypeSubscription}},
		userSubRepo: repo,
	}

	sub, reused, err := svc.assignSubscriptionWithReuse(ctx, &AssignSubscriptionInput{
		UserID: 10, GroupID: 20, ValidityDays: 30, Notes: "init",
	})

	require.NoError(t, err)
	require.True(t, reused)
	require.Equal(t, start.AddDate(0, 0, 30), sub.ExpiresAt)

	events := repo.snapshotEvents()
	assertLockedReadDiscipline(t, events, "GetByUserIDAndGroupIDForUpdate", "GetByUserIDAndGroupID")
}

func TestReduceOrCancelSubscription_LockedReadAndSingleWrite(t *testing.T) {
	tx := &dbent.Tx{}
	ctx := dbent.NewTxContext(context.Background(), tx)
	repo := newTermLockingUserSubRepo()
	now := time.Now()
	repo.seed(&UserSubscription{
		ID: 1, UserID: 10, GroupID: 20,
		StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(10 * 24 * time.Hour),
		Status: SubscriptionStatusActive,
	})
	redeemSvc := &RedeemService{subscriptionService: &SubscriptionService{userSubRepo: repo}}

	err := redeemSvc.reduceOrCancelSubscription(ctx, 10, 20, 2, "REDUCE-CODE")

	require.NoError(t, err)
	sub, ok := repo.lookupByID(1)
	require.True(t, ok)
	require.Equal(t, now.Add(10*24*time.Hour).AddDate(0, 0, -2), sub.ExpiresAt)
	require.Contains(t, sub.Notes, "REDUCE-CODE")

	events := repo.snapshotEvents()
	assertLockedReadDiscipline(t, events, "GetByUserIDAndGroupIDForUpdate", "GetByUserIDAndGroupID")
	require.Contains(t, events, termReadWriteEvent{name: "Update", inTx: true})
	require.NotContains(t, events, termReadWriteEvent{name: "UpdateStatus", inTx: true}, "reduce must not write status separately")
	require.NotContains(t, events, termReadWriteEvent{name: "UpdateNotes", inTx: true}, "reduce must not write notes separately")
	require.NotContains(t, events, termReadWriteEvent{name: "ExtendExpiry", inTx: true}, "reduce must not write expiry separately")
}

type renewalVersionedCache struct {
	billingCacheWorkerStub
	versionedCalls int
	plainCalls     int
	version        int64
}

func (c *renewalVersionedCache) InvalidateSubscriptionCache(context.Context, int64, int64) error {
	c.plainCalls++
	return nil
}

func (c *renewalVersionedCache) InvalidateSubscriptionCacheVersioned(_ context.Context, _ int64, _ int64, version int64) error {
	c.versionedCalls++
	c.version = version
	return nil
}

func TestExpiredRenewalsUseVersionedTombstoneAfterCommit(t *testing.T) {
	const renewalVersion int64 = 2234567890123456000
	paths := []struct {
		name string
		call func(*SubscriptionService, context.Context) error
	}{
		{
			name: "assign_subscription",
			call: func(s *SubscriptionService, ctx context.Context) error {
				_, err := s.AssignSubscription(ctx, &AssignSubscriptionInput{UserID: 10, GroupID: 20, ValidityDays: 30})
				return err
			},
		},
		{
			name: "assign_or_extend_subscription",
			call: func(s *SubscriptionService, ctx context.Context) error {
				_, _, err := s.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{UserID: 10, GroupID: 20, ValidityDays: 30})
				return err
			},
		},
		{
			name: "extend_subscription",
			call: func(s *SubscriptionService, ctx context.Context) error {
				_, err := s.ExtendSubscription(ctx, 1, 30)
				return err
			},
		},
	}

	for _, path := range paths {
		t.Run(path.name, func(t *testing.T) {
			client := newPaymentOrderLifecycleTestClient(t)
			repo := newTermLockingUserSubRepo()
			repo.updateVersion = renewalVersion
			now := time.Now()
			repo.seed(&UserSubscription{
				ID: 1, UserID: 10, GroupID: 20,
				StartsAt: now.AddDate(0, 0, -30), ExpiresAt: now.Add(-time.Hour),
				Status: SubscriptionStatusExpired,
			})
			cache := &renewalVersionedCache{}
			svc := &SubscriptionService{
				groupRepo:           &subscriptionGroupRepoStub{group: &Group{ID: 20, SubscriptionType: SubscriptionTypeSubscription}},
				userSubRepo:         repo,
				billingCacheService: &BillingCacheService{cache: cache},
				entClient:           client,
			}
			tx, err := client.Tx(context.Background())
			require.NoError(t, err)
			t.Cleanup(func() { _ = tx.Rollback() })

			err = path.call(svc, dbent.NewTxContext(context.Background(), tx))

			require.NoError(t, err)
			require.Zero(t, cache.versionedCalls, "versioned tombstone must wait for the outer transaction commit")
			require.NoError(t, tx.Commit())
			require.Equal(t, 1, cache.versionedCalls)
			require.Equal(t, renewalVersion, cache.version)
			require.Zero(t, cache.plainCalls, "expired renewal must not replace its tombstone with a plain cache delete")
		})
	}
}
