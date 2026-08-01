//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type revokeCacheUserSubRepoStub struct {
	userSubRepoNoop

	sub                    *UserSubscription
	deleted                bool
	getActiveCalls          int
	authoritativeUpdatedAt time.Time
	lockTx                 *dbent.Tx
	deleteTx               *dbent.Tx
	includeDeletedTx       *dbent.Tx
}

func (r *revokeCacheUserSubRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id || r.deleted {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *revokeCacheUserSubRepoStub) GetByIDForUpdate(ctx context.Context, id int64) (*UserSubscription, error) {
	r.lockTx = dbent.TxFromContext(ctx)
	return r.GetByID(ctx, id)
}

func (r *revokeCacheUserSubRepoStub) GetByUserIDAndGroupIDForUpdate(context.Context, int64, int64) (*UserSubscription, error) {
	panic("unexpected GetByUserIDAndGroupIDForUpdate call")
}

func (r *revokeCacheUserSubRepoStub) GetByIDIncludeDeleted(ctx context.Context, id int64) (*UserSubscription, error) {
	r.includeDeletedTx = dbent.TxFromContext(ctx)
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *revokeCacheUserSubRepoStub) Delete(ctx context.Context, id int64) error {
	r.deleteTx = dbent.TxFromContext(ctx)
	if r.sub == nil || r.sub.ID != id || r.deleted {
		return ErrSubscriptionNotFound
	}
	r.deleted = true
	if r.authoritativeUpdatedAt.IsZero() {
		r.authoritativeUpdatedAt = time.Unix(0, 1000)
	}
	r.sub.UpdatedAt = r.authoritativeUpdatedAt
	return nil
}

type revokeCacheInvalidationStub struct {
	billingCacheWorkerStub

	version    int64
	events     []string
	publishErr error
}

func (s *revokeCacheInvalidationStub) InvalidateSubscriptionCacheVersioned(_ context.Context, _, _ int64, version int64) error {
	s.version = version
	s.events = append(s.events, "tombstone")
	return nil
}

func (s *revokeCacheInvalidationStub) PublishSubscriptionCacheInvalidation(context.Context, string) error {
	s.events = append(s.events, "publish")
	return s.publishErr
}

func (s *revokeCacheInvalidationStub) SubscribeSubscriptionCacheInvalidation(context.Context, func(string)) error {
	return nil
}

func (r *revokeCacheUserSubRepoStub) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	r.getActiveCalls++
	if r.deleted || r.sub == nil || r.sub.UserID != userID || r.sub.GroupID != groupID {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func TestRevokeSubscription_InvalidatesL1CacheSynchronously(t *testing.T) {
	repo := &revokeCacheUserSubRepoStub{
		sub: &UserSubscription{
			ID:        1,
			UserID:    10,
			GroupID:   20,
			Status:    SubscriptionStatusActive,
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, &config.Config{
		SubscriptionCache: config.SubscriptionCacheConfig{
			L1Size:       16,
			L1TTLSeconds: 60,
		},
	})
	t.Cleanup(svc.Stop)

	_, err := svc.GetActiveSubscription(context.Background(), 10, 20)
	require.NoError(t, err)
	svc.subCacheL1.Wait()
	require.Equal(t, 1, repo.getActiveCalls)

	err = svc.RevokeSubscription(context.Background(), 1)
	require.NoError(t, err)

	_, err = svc.GetActiveSubscription(context.Background(), 10, 20)
	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	require.Equal(t, 2, repo.getActiveCalls, "撤销后应回源确认订阅已不存在，不能命中旧 L1")
}

func TestRevokeSubscription_UsesVersionedCrossInstanceInvalidation(t *testing.T) {
	repo := &revokeCacheUserSubRepoStub{sub: &UserSubscription{
		ID: 1, UserID: 10, GroupID: 20, Status: SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(time.Hour),
	}}
	cache := &redeemSubscriptionCacheStub{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, &BillingCacheService{cache: cache}, nil, nil)

	err := svc.RevokeSubscription(context.Background(), 1)

	require.NoError(t, err)
	require.Equal(t, int32(1), cache.versionedInvalidations.Load())
	require.Equal(t, int32(1), cache.publications.Load())
}

func TestRevokeSubscription_OuterTransactionInvalidation(t *testing.T) {
	authoritativeUpdatedAt := time.Unix(0, 123456789)
	tests := []struct {
		name       string
		commit     bool
		publishErr error
		wantEvents []string
	}{
		{name: "commit", commit: true, wantEvents: []string{"tombstone", "publish"}},
		{name: "rollback", wantEvents: nil},
		{name: "publication failure", commit: true, publishErr: errors.New("publish failed"), wantEvents: []string{"tombstone", "publish"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newPaymentOrderLifecycleTestClient(t)
			tx, err := client.Tx(context.Background())
			require.NoError(t, err)
			t.Cleanup(func() { _ = tx.Rollback() })
			txCtx := dbent.NewTxContext(context.Background(), tx)
			repo := &revokeCacheUserSubRepoStub{
				sub: &UserSubscription{
					ID: 1, UserID: 10, GroupID: 20, Status: SubscriptionStatusActive,
					ExpiresAt: time.Now().Add(time.Hour),
				},
				authoritativeUpdatedAt: authoritativeUpdatedAt,
			}
			cache := &revokeCacheInvalidationStub{publishErr: tt.publishErr}
			svc := NewSubscriptionService(groupRepoNoop{}, repo, &BillingCacheService{cache: cache}, client, nil)

			err = svc.RevokeSubscription(txCtx, 1)
			require.NoError(t, err)
			require.Same(t, tx, repo.lockTx)
			require.Same(t, tx, repo.deleteTx)
			require.Same(t, tx, repo.includeDeletedTx)
			require.Empty(t, cache.events, "outer transaction must not invalidate before commit")
			require.Zero(t, cache.version)

			if tt.commit {
				require.NoError(t, tx.Commit())
				require.Equal(t, authoritativeUpdatedAt.UnixNano(), cache.version)
				require.Equal(t, tt.wantEvents, cache.events)
				return
			}

			require.NoError(t, tx.Rollback())
			require.Zero(t, cache.version)
			require.Empty(t, cache.events, "rollback must not invalidate caches")
		})
	}
}

type restoreUserSubRepoStub struct {
	userSubRepoNoop

	sub            *UserSubscription
	existsActive   bool
	restoreCalls   int
	restoredStatus string
}

func (r *restoreUserSubRepoStub) GetByIDIncludeDeleted(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *restoreUserSubRepoStub) ExistsActiveByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	return r.existsActive, nil
}

func (r *restoreUserSubRepoStub) Restore(_ context.Context, id int64, restoredStatus string) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	r.restoreCalls++
	r.restoredStatus = restoredStatus
	cp := *r.sub
	cp.Status = restoredStatus
	cp.DeletedAt = nil
	r.sub = &cp
	return &cp, nil
}

func TestRestoreSubscription_ExpiredActiveRestoresAsExpired(t *testing.T) {
	deletedAt := time.Now().Add(-time.Hour)
	repo := &restoreUserSubRepoStub{
		sub: &UserSubscription{
			ID:        1,
			UserID:    10,
			GroupID:   20,
			Status:    SubscriptionStatusActive,
			ExpiresAt: time.Now().Add(-time.Minute),
			DeletedAt: &deletedAt,
		},
	}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)

	restored, err := svc.RestoreSubscription(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, 1, repo.restoreCalls)
	require.Equal(t, SubscriptionStatusExpired, repo.restoredStatus)
	require.Equal(t, SubscriptionStatusExpired, restored.Status)
	require.Nil(t, restored.DeletedAt)
}

func TestRestoreSubscription_NotRevokedReturnsConflict(t *testing.T) {
	repo := &restoreUserSubRepoStub{
		sub: &UserSubscription{
			ID:        1,
			UserID:    10,
			GroupID:   20,
			Status:    SubscriptionStatusActive,
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)

	_, err := svc.RestoreSubscription(context.Background(), 1)
	require.ErrorIs(t, err, ErrSubscriptionNotRevoked)
	require.Zero(t, repo.restoreCalls)
}

func TestRestoreSubscription_LiveSubscriptionConflict(t *testing.T) {
	deletedAt := time.Now().Add(-time.Hour)
	repo := &restoreUserSubRepoStub{
		existsActive: true,
		sub: &UserSubscription{
			ID:        1,
			UserID:    10,
			GroupID:   20,
			Status:    SubscriptionStatusExpired,
			ExpiresAt: time.Now().Add(-time.Hour),
			DeletedAt: &deletedAt,
		},
	}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)

	_, err := svc.RestoreSubscription(context.Background(), 1)
	require.ErrorIs(t, err, ErrSubscriptionRestoreConflict)
	require.Zero(t, repo.restoreCalls)
}
