//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const quotaAdvanceReceiptScope = "user.subscriptions.advance_quota_cycle"

type failOnceQuotaAdvanceMarkSucceededRepo struct {
	service.IdempotencyRepository
	failNext atomic.Bool
}

func (r *failOnceQuotaAdvanceMarkSucceededRepo) MarkSucceeded(ctx context.Context, id int64, status int, body string, expiresAt time.Time) error {
	if r.failNext.CompareAndSwap(true, false) {
		return errors.New("mark succeeded unavailable")
	}
	return r.IdempotencyRepository.MarkSucceeded(ctx, id, status, body, expiresAt)
}

type failQuotaAdvanceReceiptRecoveryRepo struct {
	service.QuotaAdvanceReceiptRepository
	findCalls atomic.Int32
}

func (r *failQuotaAdvanceReceiptRecoveryRepo) Find(ctx context.Context, scope, keyHash string) (*service.QuotaAdvanceReceipt, error) {
	if r.findCalls.Add(1) > 1 {
		return nil, errors.New("receipt recovery unavailable")
	}
	return r.QuotaAdvanceReceiptRepository.Find(ctx, scope, keyHash)
}

func newQuotaAdvanceReceiptRouter(
	t *testing.T,
	userID int64,
	receiptRepo service.QuotaAdvanceReceiptRepository,
	idempotencyRepo service.IdempotencyRepository,
) *gin.Engine {
	t.Helper()
	previous := service.DefaultIdempotencyCoordinator()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(idempotencyRepo, service.DefaultIdempotencyConfig()))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(previous) })

	svc := service.NewSubscriptionService(
		NewGroupRepository(integrationEntClient, integrationDB),
		NewUserSubscriptionRepository(integrationEntClient),
		nil,
		integrationEntClient,
		nil,
		receiptRepo,
	)
	t.Cleanup(svc.Stop)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
		c.Next()
	})
	router.POST("/api/v1/subscriptions/:id/advance-quota-cycle", handler.NewSubscriptionHandler(svc).AdvanceQuotaCycle)
	return router
}

func createQuotaAdvanceReceiptFixture(t *testing.T) (userID, subscriptionID int64, originalExpiry time.Time) {
	t.Helper()
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	user, err := integrationEntClient.User.Create().
		SetEmail(fmt.Sprintf("quota-receipt-%d@test.com", suffix)).
		SetPasswordHash("test-password-hash").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(ctx)
	require.NoError(t, err)
	group, err := integrationEntClient.Group.Create().
		SetName(fmt.Sprintf("quota-receipt-%d", suffix)).
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeSubscription).
		SetDailyLimitUsd(10).
		SetWeeklyLimitUsd(70).
		Save(ctx)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	originalExpiry = now.Add(30 * 24 * time.Hour)
	sub, err := integrationEntClient.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(now.Add(-10 * 24 * time.Hour)).
		SetExpiresAt(originalExpiry).
		SetStatus(service.SubscriptionStatusActive).
		SetAssignedAt(now).
		SetDailyWindowStart(now.Add(-4 * time.Hour)).
		SetWeeklyWindowStart(now.Add(-2 * 24 * time.Hour)).
		SetDailyUsageUsd(10).
		SetWeeklyUsageUsd(0).
		SetNotes("").
		Save(ctx)
	require.NoError(t, err)
	return user.ID, sub.ID, originalExpiry
}

func callQuotaAdvanceReceipt(router *gin.Engine, subscriptionID int64, key, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/subscriptions/%d/advance-quota-cycle", subscriptionID), strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func deleteQuotaAdvanceIdempotencyRecord(t *testing.T, key string) {
	t.Helper()
	_, err := integrationDB.Exec(`DELETE FROM idempotency_records WHERE scope = $1 AND idempotency_key_hash = $2`, quotaAdvanceReceiptScope, service.HashIdempotencyKey(key))
	require.NoError(t, err)
}

func loadQuotaAdvanceSubscription(t *testing.T, subscriptionID int64) *service.UserSubscription {
	t.Helper()
	stored, err := integrationEntClient.UserSubscription.Get(context.Background(), subscriptionID)
	require.NoError(t, err)
	return &service.UserSubscription{
		ExpiresAt:        stored.ExpiresAt,
		DailyUsageUSD:    stored.DailyUsageUsd,
		WeeklyUsageUSD:   stored.WeeklyUsageUsd,
		DailyWindowStart: stored.DailyWindowStart,
	}
}

func TestAdvanceQuotaCycleReceipt_RecoversMarkSucceededFailure(t *testing.T) {
	receipts := NewSubscriptionQuotaAdvanceReceiptRepository(integrationEntClient, integrationDB)
	idempotency := &failOnceQuotaAdvanceMarkSucceededRepo{IdempotencyRepository: NewIdempotencyRepository(integrationEntClient, integrationDB)}
	idempotency.failNext.Store(true)
	userID, subscriptionID, originalExpiry := createQuotaAdvanceReceiptFixture(t)
	router := newQuotaAdvanceReceiptRouter(t, userID, receipts, idempotency)

	first := callQuotaAdvanceReceipt(router, subscriptionID, "receipt-mark-succeeded", `{"daily":true}`)
	second := callQuotaAdvanceReceipt(router, subscriptionID, "receipt-mark-succeeded", `{"daily":true}`)

	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, "true", first.Header().Get("X-Idempotency-Recovered"))
	require.Equal(t, "true", second.Header().Get("X-Idempotency-Recovered"))
	require.JSONEq(t, first.Body.String(), second.Body.String())
	stored := loadQuotaAdvanceSubscription(t, subscriptionID)
	require.Zero(t, stored.DailyUsageUSD)
	require.WithinDuration(t, originalExpiry.Add(-20*time.Hour), stored.ExpiresAt, 3*time.Second)
}

func TestAdvanceQuotaCycleReceipt_ReplaysWhileGenericRecordProcessing(t *testing.T) {
	receipts := NewSubscriptionQuotaAdvanceReceiptRepository(integrationEntClient, integrationDB)
	key := "receipt-processing"
	userID, subscriptionID, _ := createQuotaAdvanceReceiptFixture(t)
	router := newQuotaAdvanceReceiptRouter(t, userID, receipts, NewIdempotencyRepository(integrationEntClient, integrationDB))

	first := callQuotaAdvanceReceipt(router, subscriptionID, key, `{"daily":true}`)
	require.Equal(t, http.StatusOK, first.Code)
	_, err := integrationDB.Exec(
		`UPDATE idempotency_records SET status = $1, response_status = NULL, response_body = NULL, locked_until = $2 WHERE scope = $3 AND idempotency_key_hash = $4`,
		service.IdempotencyStatusProcessing,
		time.Now().Add(time.Minute),
		quotaAdvanceReceiptScope,
		service.HashIdempotencyKey(key),
	)
	require.NoError(t, err)

	retry := callQuotaAdvanceReceipt(router, subscriptionID, key, `{"daily":true}`)

	require.Equal(t, http.StatusOK, retry.Code)
	require.Equal(t, "true", retry.Header().Get("X-Idempotency-Recovered"))
	require.JSONEq(t, first.Body.String(), retry.Body.String())
	require.Zero(t, loadQuotaAdvanceSubscription(t, subscriptionID).DailyUsageUSD)
}

func TestAdvanceQuotaCycleReceipt_ReplaysAfterGenericRecordCleanup(t *testing.T) {
	receipts := NewSubscriptionQuotaAdvanceReceiptRepository(integrationEntClient, integrationDB)
	key := "receipt-expired-record"
	userID, subscriptionID, originalExpiry := createQuotaAdvanceReceiptFixture(t)
	router := newQuotaAdvanceReceiptRouter(t, userID, receipts, NewIdempotencyRepository(integrationEntClient, integrationDB))

	first := callQuotaAdvanceReceipt(router, subscriptionID, key, `{"daily":true}`)
	require.Equal(t, http.StatusOK, first.Code)
	deleteQuotaAdvanceIdempotencyRecord(t, key)

	retry := callQuotaAdvanceReceipt(router, subscriptionID, key, `{"daily":true}`)

	require.Equal(t, http.StatusOK, retry.Code)
	require.JSONEq(t, first.Body.String(), retry.Body.String())
	stored := loadQuotaAdvanceSubscription(t, subscriptionID)
	require.Zero(t, stored.DailyUsageUSD)
	require.WithinDuration(t, originalExpiry.Add(-20*time.Hour), stored.ExpiresAt, 3*time.Second)
}

func TestAdvanceQuotaCycleReceipt_RejectsDifferentFingerprintAfterGenericRecordCleanup(t *testing.T) {
	receipts := NewSubscriptionQuotaAdvanceReceiptRepository(integrationEntClient, integrationDB)
	key := "receipt-fingerprint-conflict"
	userID, subscriptionID, _ := createQuotaAdvanceReceiptFixture(t)
	router := newQuotaAdvanceReceiptRouter(t, userID, receipts, NewIdempotencyRepository(integrationEntClient, integrationDB))

	first := callQuotaAdvanceReceipt(router, subscriptionID, key, `{"daily":true}`)
	require.Equal(t, http.StatusOK, first.Code)
	before := loadQuotaAdvanceSubscription(t, subscriptionID)
	deleteQuotaAdvanceIdempotencyRecord(t, key)

	conflict := callQuotaAdvanceReceipt(router, subscriptionID, key, `{"weekly":true}`)

	require.Equal(t, http.StatusConflict, conflict.Code)
	require.Contains(t, conflict.Body.String(), "IDEMPOTENCY_KEY_CONFLICT")
	after := loadQuotaAdvanceSubscription(t, subscriptionID)
	require.Equal(t, before.ExpiresAt, after.ExpiresAt)
	require.Equal(t, before.WeeklyUsageUSD, after.WeeklyUsageUSD)
}

func TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails(t *testing.T) {
	userID, subscriptionID, originalExpiry := createQuotaAdvanceReceiptFixture(t)
	key := "receipt-write-failure"
	keyHash := service.HashIdempotencyKey(key)
	_, err := integrationDB.Exec(fmt.Sprintf(`
		CREATE OR REPLACE FUNCTION fail_quota_advance_receipt_insert() RETURNS trigger AS $$
		BEGIN
			IF NEW.idempotency_key_hash = '%s' THEN
				RAISE EXCEPTION 'receipt insert unavailable';
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER fail_quota_advance_receipt_insert_trigger
		BEFORE INSERT ON subscription_quota_advance_receipts
		FOR EACH ROW EXECUTE FUNCTION fail_quota_advance_receipt_insert();`, keyHash))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = integrationDB.Exec(`DROP TRIGGER IF EXISTS fail_quota_advance_receipt_insert_trigger ON subscription_quota_advance_receipts`)
		_, _ = integrationDB.Exec(`DROP FUNCTION IF EXISTS fail_quota_advance_receipt_insert()`)
	})
	router := newQuotaAdvanceReceiptRouter(t, userID, NewSubscriptionQuotaAdvanceReceiptRepository(integrationEntClient, integrationDB), NewIdempotencyRepository(integrationEntClient, integrationDB))

	response := callQuotaAdvanceReceipt(router, subscriptionID, key, `{"daily":true}`)

	require.Equal(t, http.StatusInternalServerError, response.Code)
	stored := loadQuotaAdvanceSubscription(t, subscriptionID)
	require.Equal(t, originalExpiry, stored.ExpiresAt)
	require.Equal(t, float64(10), stored.DailyUsageUSD)
}

func TestAdvanceQuotaCycleReceipt_PreservesStoreErrorWhenRecoveryLookupFails(t *testing.T) {
	baseReceipts := NewSubscriptionQuotaAdvanceReceiptRepository(integrationEntClient, integrationDB)
	receipts := &failQuotaAdvanceReceiptRecoveryRepo{QuotaAdvanceReceiptRepository: baseReceipts}
	idempotency := &failOnceQuotaAdvanceMarkSucceededRepo{IdempotencyRepository: NewIdempotencyRepository(integrationEntClient, integrationDB)}
	idempotency.failNext.Store(true)
	userID, subscriptionID, _ := createQuotaAdvanceReceiptFixture(t)
	router := newQuotaAdvanceReceiptRouter(t, userID, receipts, idempotency)

	response := callQuotaAdvanceReceipt(router, subscriptionID, "receipt-recovery-read-failure", `{"daily":true}`)

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Contains(t, response.Body.String(), "IDEMPOTENCY_STORE_UNAVAILABLE")
	require.Empty(t, response.Header().Get("X-Idempotency-Recovered"))
	require.Zero(t, loadQuotaAdvanceSubscription(t, subscriptionID).DailyUsageUSD)
}

func TestSubscriptionQuotaAdvanceReceiptRepository_CleansExpiredRowsInBatchesAndStoresOnlyHash(t *testing.T) {
	repo := NewSubscriptionQuotaAdvanceReceiptRepository(integrationEntClient, integrationDB)
	now := time.Now().UTC()
	for _, key := range []string{"receipt-cleanup-one", "receipt-cleanup-two"} {
		err := repo.Create(context.Background(), &service.QuotaAdvanceReceipt{
			Scope:              quotaAdvanceReceiptScope,
			IdempotencyKeyHash: service.HashIdempotencyKey(key),
			RequestFingerprint: "fingerprint-" + key,
			UserID:             1,
			SubscriptionID:     1,
			Daily:              true,
			ResponseStatus:     http.StatusOK,
			ResponseBody:       `{"subscription":{"id":1},"deducted_seconds":72000}`,
			ExpiresAt:          now.Add(-time.Minute),
		})
		require.NoError(t, err)
	}

	deleted, err := repo.DeleteExpired(context.Background(), now, 1)

	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
	var remaining int
	require.NoError(t, integrationDB.QueryRow(`SELECT COUNT(*) FROM subscription_quota_advance_receipts WHERE expires_at <= $1`, now).Scan(&remaining))
	require.Equal(t, 1, remaining)
	require.Greater(t, service.QuotaAdvanceReceiptRetention(), service.DefaultWriteIdempotencyTTL())
	rows, err := integrationDB.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'subscription_quota_advance_receipts'`)
	require.NoError(t, err)
	defer rows.Close()
	columns := make([]string, 0)
	for rows.Next() {
		var column string
		require.NoError(t, rows.Scan(&column))
		columns = append(columns, column)
	}
	require.NoError(t, rows.Err())
	require.NotContains(t, columns, "idempotency_key")
	require.Contains(t, columns, "idempotency_key_hash")
}

func TestSubscriptionQuotaAdvanceReceiptRepository_UsesEntTransactionAndRollbackDoesNotPersist(t *testing.T) {
	ctx := context.Background()
	observer, err := integrationDB.Conn(ctx)
	require.NoError(t, err)
	defer observer.Close()

	repo := NewSubscriptionQuotaAdvanceReceiptRepository(integrationEntClient, integrationDB)
	receipt := &service.QuotaAdvanceReceipt{
		Scope:              quotaAdvanceReceiptScope,
		IdempotencyKeyHash: service.HashIdempotencyKey("receipt-tx-rollback"),
		RequestFingerprint: "receipt-tx-rollback-fingerprint",
		UserID:             1,
		SubscriptionID:     1,
		Daily:              true,
		ResponseStatus:     http.StatusOK,
		ResponseBody:       `{"subscription":{"id":1},"deducted_seconds":72000}`,
		ExpiresAt:          time.Now().Add(time.Hour),
	}
	tx, err := integrationEntClient.Tx(ctx)
	require.NoError(t, err)
	txCtx := dbent.NewTxContext(ctx, tx)
	require.NoError(t, repo.Create(txCtx, receipt))
	found, err := repo.Find(txCtx, receipt.Scope, receipt.IdempotencyKeyHash)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, receipt.ResponseBody, found.ResponseBody)
	require.NoError(t, tx.Rollback())

	var count int
	require.NoError(t, observer.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscription_quota_advance_receipts WHERE scope = $1 AND idempotency_key_hash = $2`, receipt.Scope, receipt.IdempotencyKeyHash).Scan(&count))
	require.Zero(t, count)
}
