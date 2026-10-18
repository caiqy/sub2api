//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

// refundSubscriptionDBRepo uses real Ent rows, soft-delete hooks/interceptors and
// the transaction from context. SQLite cannot prove PostgreSQL row-lock behavior.
// Unused repository methods panic through userSubRepoNoop.
type refundSubscriptionDBRepo struct {
	userSubRepoNoop
	t        *testing.T
	client   *dbent.Client
	extends  int
	restores int
}

func (r *refundSubscriptionDBRepo) db(ctx context.Context) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.client
}

func refundSubscriptionFromDB(row *dbent.UserSubscription, err error) (*UserSubscription, error) {
	if dbent.IsNotFound(err) {
		return nil, ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &UserSubscription{
		ID: row.ID, UserID: row.UserID, GroupID: row.GroupID,
		StartsAt: row.StartsAt, ExpiresAt: row.ExpiresAt, Status: row.Status,
		DeletedAt: row.DeletedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *refundSubscriptionDBRepo) GetByID(ctx context.Context, id int64) (*UserSubscription, error) {
	return refundSubscriptionFromDB(r.db(ctx).UserSubscription.Get(ctx, id))
}

func (r *refundSubscriptionDBRepo) GetByIDForUpdate(ctx context.Context, id int64) (*UserSubscription, error) {
	require.NotNil(r.t, dbent.TxFromContext(ctx), "locked read must use the subscription transaction")
	return r.GetByID(ctx, id)
}

func (r *refundSubscriptionDBRepo) GetByUserIDAndGroupIDForUpdate(context.Context, int64, int64) (*UserSubscription, error) {
	panic("unexpected GetByUserIDAndGroupIDForUpdate")
}

func (r *refundSubscriptionDBRepo) GetByIDIncludeDeleted(ctx context.Context, id int64) (*UserSubscription, error) {
	return r.GetByID(mixins.SkipSoftDelete(ctx), id)
}

func (r *refundSubscriptionDBRepo) GetActiveByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	return refundSubscriptionFromDB(r.db(ctx).UserSubscription.Query().Where(
		usersubscription.UserIDEQ(userID), usersubscription.GroupIDEQ(groupID),
		usersubscription.StatusEQ(SubscriptionStatusActive), usersubscription.ExpiresAtGT(time.Now()),
	).Only(ctx))
}

func (r *refundSubscriptionDBRepo) ExistsActiveByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (bool, error) {
	return r.db(ctx).UserSubscription.Query().Where(
		usersubscription.UserIDEQ(userID), usersubscription.GroupIDEQ(groupID),
	).Exist(ctx)
}

func (r *refundSubscriptionDBRepo) Delete(ctx context.Context, id int64) error {
	require.NotNil(r.t, dbent.TxFromContext(ctx))
	return r.db(ctx).UserSubscription.DeleteOneID(id).Exec(ctx)
}

func (r *refundSubscriptionDBRepo) Restore(ctx context.Context, id int64, status string) (*UserSubscription, error) {
	r.restores++
	_, err := r.db(ctx).UserSubscription.UpdateOneID(id).
		ClearDeletedAt().SetStatus(status).SetUpdatedAt(time.Now()).Save(mixins.SkipSoftDelete(ctx))
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *refundSubscriptionDBRepo) ExtendExpiry(ctx context.Context, id int64, expiry time.Time) error {
	require.NotNil(r.t, dbent.TxFromContext(ctx))
	r.extends++
	_, err := r.db(ctx).UserSubscription.UpdateOneID(id).SetExpiresAt(expiry).SetUpdatedAt(time.Now()).Save(ctx)
	return err
}

func newRefundSubscriptionHoldFixture(t *testing.T, remainingDays int) (*PaymentService, *dbent.PaymentOrder, *dbent.UserSubscription, *refundSubscriptionDBRepo, *redeemSubscriptionCacheStub) {
	t.Helper()
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createCompletedRefundableOrderForTest(t, ctx, client, "subscription-hold")
	group, err := client.Group.Create().SetName("refund-subscription").SetSubscriptionType(SubscriptionTypeSubscription).Save(ctx)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	sub, err := client.UserSubscription.Create().SetUserID(order.UserID).SetGroupID(group.ID).
		SetStartsAt(now.AddDate(0, 0, -1)).SetExpiresAt(now.AddDate(0, 0, remainingDays)).
		SetStatus(SubscriptionStatusActive).Save(ctx)
	require.NoError(t, err)
	order, err = client.PaymentOrder.UpdateOneID(order.ID).SetOrderType(payment.OrderTypeSubscription).
		SetSubscriptionGroupID(group.ID).SetSubscriptionDays(5).Save(ctx)
	require.NoError(t, err)
	repo := &refundSubscriptionDBRepo{t: t, client: client}
	cache := &redeemSubscriptionCacheStub{}
	svc := &PaymentService{entClient: client, loadBalancer: &captureLoadBalancer{}, subscriptionSvc: &SubscriptionService{
		userSubRepo: repo, entClient: client, billingCacheService: &BillingCacheService{cache: cache},
	}}
	return svc, order, sub, repo, cache
}

func executeSubscriptionHoldForTest(t *testing.T, svc *PaymentService, order *dbent.PaymentOrder) *RefundPlan {
	t.Helper()
	plan, early, err := svc.PrepareRefund(context.Background(), order.ID, 0, "subscription refund", false, true)
	require.NoError(t, err)
	require.Nil(t, early)
	result, err := svc.ExecuteRefund(context.Background(), plan)
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Warning, "pending confirmation")
	return plan
}

func TestRefundSubscriptionHeldLifecycle(t *testing.T) {
	for _, remainingDays := range []int{2, 10} {
		for _, status := range []string{payment.ProviderStatusFailed, payment.ProviderStatusSuccess} {
			name := status + "/shortened"
			if remainingDays == 2 {
				name = status + "/revoked"
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				svc, order, original, repo, cache := newRefundSubscriptionHoldFixture(t, remainingDays)
				restore := replacePaymentProviderFactoryForTest(t, &pendingRefundProviderTestDouble{queryStatus: status})
				defer restore()
				plan := executeSubscriptionHoldForTest(t, svc, order)
				held, err := repo.GetByIDIncludeDeleted(ctx, original.ID)
				require.NoError(t, err)
				if remainingDays == 2 {
					require.NotNil(t, held.DeletedAt)
					require.True(t, original.ExpiresAt.Equal(held.ExpiresAt), "revoke never shortened the original term")
					_, err = repo.GetByID(ctx, original.ID)
					require.ErrorIs(t, err, ErrSubscriptionNotFound, "real soft-delete interceptor must hide the held row")
				} else {
					require.Nil(t, held.DeletedAt)
					require.True(t, original.ExpiresAt.AddDate(0, 0, -5).Equal(held.ExpiresAt))
				}
				detail, err := svc.latestRefundPendingDetail(ctx, order.ID)
				require.NoError(t, err)
				reloadedPlan := svc.refundFinalizePlan(order)
				detail.applyTo(reloadedPlan)
				require.Equal(t, plan.subscriptionRevoked, reloadedPlan.subscriptionRevoked, "revocation ownership must survive audit reload")
				require.Equal(t, remainingDays == 2, reloadedPlan.subscriptionRevoked)
				before := cache.versionedInvalidations.Load()
				result, err := svc.QueryAndFinalizeRefund(ctx, order.ID)
				require.NoError(t, err)
				require.Equal(t, status == payment.ProviderStatusSuccess, result.Success)
				final, err := repo.GetByIDIncludeDeleted(ctx, original.ID)
				require.NoError(t, err)
				wantStatus := OrderStatusRefunded
				if status == payment.ProviderStatusFailed {
					wantStatus = OrderStatusRefundFailed
					require.NotContains(t, result.Warning, "rollback failed")
					require.Nil(t, final.DeletedAt)
					require.Equal(t, SubscriptionStatusActive, final.Status)
					require.True(t, original.ExpiresAt.Equal(final.ExpiresAt), "failure returns precisely the original term, never an extra five days")
					require.Equal(t, before+1, cache.versionedInvalidations.Load())
				} else {
					require.Equal(t, held.DeletedAt, final.DeletedAt)
					require.True(t, held.ExpiresAt.Equal(final.ExpiresAt))
					require.Equal(t, before, cache.versionedInvalidations.Load(), "success must not restore or deduct twice")
				}
				reloadedOrder, err := svc.entClient.PaymentOrder.Get(ctx, order.ID)
				require.NoError(t, err)
				require.Equal(t, wantStatus, reloadedOrder.Status)
				require.Zero(t, countRefundAuditsForTest(t, ctx, svc.entClient, order.ID, "REFUND_ROLLBACK_FAILED"))
				calls := repo.extends + repo.restores
				_, err = svc.QueryAndFinalizeRefund(ctx, order.ID)
				require.Equal(t, "INVALID_STATUS", infraerrors.Reason(err))
				if status == payment.ProviderStatusFailed {
					_, err = svc.finalizeRefundFailed(ctx, reloadedPlan, errors.New("stale failure"))
				} else {
					_, err = svc.finalizePendingRefundSuccess(ctx, reloadedPlan)
				}
				require.Equal(t, "CONFLICT", infraerrors.Reason(err))
				require.Equal(t, calls, repo.extends+repo.restores, "stale convergence must not return or deduct twice")
			})
		}
	}
}

type failedSubscriptionRefundProvider struct{ refundProviderTestDouble }

func (failedSubscriptionRefundProvider) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return &payment.RefundResponse{Status: payment.ProviderStatusFailed}, nil
}

func TestRefundSubscriptionGatewayFailureRestoresOwnRevocation(t *testing.T) {
	ctx := context.Background()
	svc, order, original, repo, _ := newRefundSubscriptionHoldFixture(t, 2)
	restore := replacePaymentProviderFactoryForTest(t, failedSubscriptionRefundProvider{})
	defer restore()
	plan, early, err := svc.PrepareRefund(ctx, order.ID, 0, "immediate failure", false, true)
	require.NoError(t, err)
	require.Nil(t, early)
	result, err := svc.ExecuteRefund(ctx, plan)
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Warning, "rolled back")
	final, err := repo.GetByID(ctx, original.ID)
	require.NoError(t, err)
	require.True(t, original.ExpiresAt.Equal(final.ExpiresAt))
	require.Equal(t, 1, repo.restores)
	require.Zero(t, repo.extends)
	reloadedOrder, err := svc.entClient.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, reloadedOrder.Status)
}

func TestRefundSubscriptionFailureDoesNotRestoreIndependentRevocation(t *testing.T) {
	ctx := context.Background()
	svc, order, original, repo, _ := newRefundSubscriptionHoldFixture(t, 10)
	restore := replacePaymentProviderFactoryForTest(t, &pendingRefundProviderTestDouble{queryStatus: payment.ProviderStatusFailed})
	defer restore()
	executeSubscriptionHoldForTest(t, svc, order)
	require.NoError(t, svc.subscriptionSvc.RevokeSubscription(ctx, original.ID), "administrator independently revokes the shortened subscription")
	result, err := svc.QueryAndFinalizeRefund(ctx, order.ID)
	require.NoError(t, err)
	require.Contains(t, result.Warning, "rollback failed")
	final, err := repo.GetByIDIncludeDeleted(ctx, original.ID)
	require.NoError(t, err)
	require.NotNil(t, final.DeletedAt)
	require.True(t, original.ExpiresAt.AddDate(0, 0, -5).Equal(final.ExpiresAt))
	require.Zero(t, repo.restores)
	require.Equal(t, 1, countRefundAuditsForTest(t, ctx, svc.entClient, order.ID, "REFUND_ROLLBACK_FAILED"))
}

func TestRefundSubscriptionFailurePreservesRestoreConflict(t *testing.T) {
	ctx := context.Background()
	svc, order, original, repo, cache := newRefundSubscriptionHoldFixture(t, 2)
	restore := replacePaymentProviderFactoryForTest(t, &pendingRefundProviderTestDouble{queryStatus: payment.ProviderStatusFailed})
	defer restore()
	executeSubscriptionHoldForTest(t, svc, order)
	_, err := svc.entClient.UserSubscription.Create().SetUserID(original.UserID).SetGroupID(original.GroupID).
		SetStartsAt(original.StartsAt).SetExpiresAt(original.ExpiresAt).Save(ctx)
	require.NoError(t, err)
	before := cache.versionedInvalidations.Load()
	result, err := svc.QueryAndFinalizeRefund(ctx, order.ID)
	require.NoError(t, err)
	require.Contains(t, result.Warning, "rollback failed")
	final, err := repo.GetByIDIncludeDeleted(ctx, original.ID)
	require.NoError(t, err)
	require.NotNil(t, final.DeletedAt, "a replacement subscription must prevent restoration")
	require.Zero(t, repo.restores)
	require.Equal(t, before, cache.versionedInvalidations.Load())
	require.Equal(t, 1, countRefundAuditsForTest(t, ctx, svc.entClient, order.ID, "REFUND_ROLLBACK_FAILED"))
}

func TestRefundSubscriptionFailureAuditIsAtomicAndCacheWaitsForCommit(t *testing.T) {
	for _, remainingDays := range []int{2, 10} {
		name := "shortened"
		if remainingDays == 2 {
			name = "revoked"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			svc, order, original, repo, cache := newRefundSubscriptionHoldFixture(t, remainingDays)
			restore := replacePaymentProviderFactoryForTest(t, &pendingRefundProviderTestDouble{queryStatus: payment.ProviderStatusFailed})
			defer restore()
			executeSubscriptionHoldForTest(t, svc, order)
			held, err := repo.GetByIDIncludeDeleted(ctx, original.ID)
			require.NoError(t, err)
			before := cache.versionedInvalidations.Load()
			beforePublications := cache.publications.Load()
			failAudit := true
			auditCalls := 0
			svc.entClient.PaymentAuditLog.Use(func(next ent.Mutator) ent.Mutator {
				return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
					if action, _ := m.(*dbent.PaymentAuditLogMutation).Action(); action == "REFUND_FAILED" {
						auditCalls++
						tx := dbent.TxFromContext(ctx)
						require.NotNil(t, tx)
						returned, err := repo.GetByID(ctx, original.ID)
						require.NoError(t, err)
						require.True(t, original.ExpiresAt.Equal(returned.ExpiresAt), "audit observes the restored row inside the same transaction")
						require.Equal(t, before, cache.versionedInvalidations.Load())
						require.Equal(t, beforePublications, cache.publications.Load())
						if failAudit {
							return nil, errors.New("injected refund failure audit error")
						}
					}
					return next.Mutate(ctx, m)
				})
			})
			result, err := svc.QueryAndFinalizeRefund(ctx, order.ID)
			require.Nil(t, result)
			require.ErrorContains(t, err, "injected refund failure audit error")
			rolledBack, err := repo.GetByIDIncludeDeleted(ctx, original.ID)
			require.NoError(t, err)
			require.Equal(t, held.DeletedAt, rolledBack.DeletedAt)
			require.True(t, held.ExpiresAt.Equal(rolledBack.ExpiresAt))
			reloadedOrder, err := svc.entClient.PaymentOrder.Get(ctx, order.ID)
			require.NoError(t, err)
			require.Equal(t, OrderStatusRefundPending, reloadedOrder.Status)
			require.Zero(t, countRefundAuditsForTest(t, ctx, svc.entClient, order.ID, "REFUND_FAILED"))
			require.Equal(t, before, cache.versionedInvalidations.Load(), "rollback must discard cache callbacks")
			require.Equal(t, beforePublications, cache.publications.Load())
			failAudit = false
			result, err = svc.QueryAndFinalizeRefund(ctx, order.ID)
			require.NoError(t, err)
			require.NotContains(t, result.Warning, "rollback failed")
			require.Equal(t, 2, auditCalls)
			require.Equal(t, before+1, cache.versionedInvalidations.Load())
			require.Equal(t, beforePublications+1, cache.publications.Load())
		})
	}
}
