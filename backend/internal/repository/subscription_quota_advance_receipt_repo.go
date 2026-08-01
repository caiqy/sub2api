package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type subscriptionQuotaAdvanceReceiptRepository struct {
	client *dbent.Client
	sql    *sql.DB
}

func NewSubscriptionQuotaAdvanceReceiptRepository(client *dbent.Client, sqlDB *sql.DB) service.QuotaAdvanceReceiptRepository {
	return &subscriptionQuotaAdvanceReceiptRepository{client: client, sql: sqlDB}
}

func (r *subscriptionQuotaAdvanceReceiptRepository) Find(ctx context.Context, scope, keyHash string) (*service.QuotaAdvanceReceipt, error) {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return nil, fmt.Errorf("quota advance receipt sql executor is not configured")
	}

	receipt := &service.QuotaAdvanceReceipt{}
	err := scanSingleRow(ctx, exec, `
		SELECT scope, idempotency_key_hash, request_fingerprint, user_id, subscription_id,
			daily, weekly, monthly, response_status, response_body, expires_at, created_at
		FROM subscription_quota_advance_receipts
		WHERE scope = $1 AND idempotency_key_hash = $2 AND expires_at > NOW()`,
		[]any{scope, keyHash},
		&receipt.Scope,
		&receipt.IdempotencyKeyHash,
		&receipt.RequestFingerprint,
		&receipt.UserID,
		&receipt.SubscriptionID,
		&receipt.Daily,
		&receipt.Weekly,
		&receipt.Monthly,
		&receipt.ResponseStatus,
		&receipt.ResponseBody,
		&receipt.ExpiresAt,
		&receipt.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (r *subscriptionQuotaAdvanceReceiptRepository) Create(ctx context.Context, receipt *service.QuotaAdvanceReceipt) error {
	if receipt == nil {
		return fmt.Errorf("quota advance receipt is nil")
	}
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return fmt.Errorf("quota advance receipt sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
		INSERT INTO subscription_quota_advance_receipts (
			scope, idempotency_key_hash, request_fingerprint, user_id, subscription_id,
			daily, weekly, monthly, response_status, response_body, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		receipt.Scope,
		receipt.IdempotencyKeyHash,
		receipt.RequestFingerprint,
		receipt.UserID,
		receipt.SubscriptionID,
		receipt.Daily,
		receipt.Weekly,
		receipt.Monthly,
		receipt.ResponseStatus,
		receipt.ResponseBody,
		receipt.ExpiresAt,
	)
	return err
}

func (r *subscriptionQuotaAdvanceReceiptRepository) DeleteExpired(ctx context.Context, now time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 500
	}
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return 0, fmt.Errorf("quota advance receipt sql executor is not configured")
	}
	result, err := exec.ExecContext(ctx, `
		WITH victims AS (
			SELECT id
			FROM subscription_quota_advance_receipts
			WHERE expires_at <= $1
			ORDER BY expires_at ASC
			LIMIT $2
		)
		DELETE FROM subscription_quota_advance_receipts
		WHERE id IN (SELECT id FROM victims)`, now, limit)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
