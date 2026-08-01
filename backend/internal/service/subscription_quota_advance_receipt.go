package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrQuotaAdvanceReceiptUnavailable = infraerrors.ServiceUnavailable("QUOTA_ADVANCE_RECEIPT_UNAVAILABLE", "quota cycle advance receipt storage is unavailable")

type QuotaAdvanceReceipt struct {
	Scope              string
	IdempotencyKeyHash string
	RequestFingerprint string
	UserID             int64
	SubscriptionID     int64
	Daily              bool
	Weekly             bool
	Monthly            bool
	ResponseStatus     int
	ResponseBody       string
	ExpiresAt          time.Time
	CreatedAt          time.Time
}

type QuotaAdvanceReceiptRepository interface {
	Find(ctx context.Context, scope, keyHash string) (*QuotaAdvanceReceipt, error)
	Create(ctx context.Context, receipt *QuotaAdvanceReceipt) error
	DeleteExpired(ctx context.Context, now time.Time, limit int) (int64, error)
}

type QuotaAdvanceReceiptInput struct {
	Scope              string
	IdempotencyKeyHash string
	RequestFingerprint string
	ExpiresAt          time.Time
}

type QuotaAdvanceResponseBuilder func(*QuotaCycleAdvanceResult) (any, error)

// QuotaAdvanceReceiptRetention outlives the generic idempotency row so a cleaned
// or ambiguous generic record can still replay the committed endpoint result.
func QuotaAdvanceReceiptRetention() time.Duration {
	return DefaultWriteIdempotencyTTL() + time.Hour
}

func (s *SubscriptionService) AdvanceQuotaCycleWithReceipt(
	ctx context.Context,
	userID, subscriptionID int64,
	selection QuotaWindowSelection,
	receiptInput QuotaAdvanceReceiptInput,
	buildResponse QuotaAdvanceResponseBuilder,
) (json.RawMessage, error) {
	if s.quotaAdvanceReceiptRepo == nil || buildResponse == nil || receiptInput.Scope == "" ||
		receiptInput.IdempotencyKeyHash == "" || receiptInput.RequestFingerprint == "" {
		return nil, ErrQuotaAdvanceReceiptUnavailable
	}
	if receiptInput.ExpiresAt.IsZero() {
		receiptInput.ExpiresAt = time.Now().Add(QuotaAdvanceReceiptRetention())
	}
	_, response, err := s.advanceQuotaCycle(ctx, userID, subscriptionID, selection, &receiptInput, buildResponse)
	return response, err
}

func (s *SubscriptionService) RecoverAdvanceQuotaCycleReceipt(ctx context.Context, receiptInput QuotaAdvanceReceiptInput) (json.RawMessage, bool, error) {
	if s.quotaAdvanceReceiptRepo == nil {
		return nil, false, ErrQuotaAdvanceReceiptUnavailable
	}
	receipt, err := s.quotaAdvanceReceiptRepo.Find(ctx, receiptInput.Scope, receiptInput.IdempotencyKeyHash)
	if err != nil {
		return nil, false, err
	}
	if receipt == nil {
		return nil, false, nil
	}
	if receipt.RequestFingerprint != receiptInput.RequestFingerprint {
		return nil, false, ErrIdempotencyKeyConflict
	}
	return json.RawMessage(receipt.ResponseBody), true, nil
}

func (s *SubscriptionService) advanceQuotaCycle(
	ctx context.Context,
	userID, subscriptionID int64,
	selection QuotaWindowSelection,
	receiptInput *QuotaAdvanceReceiptInput,
	buildResponse QuotaAdvanceResponseBuilder,
) (*QuotaCycleAdvanceResult, json.RawMessage, error) {
	if !selection.Daily && !selection.Weekly && !selection.Monthly {
		return nil, nil, ErrQuotaAdvanceSelectionRequired
	}
	if err := s.requireVersionedSubscriptionInvalidation(); err != nil {
		return nil, nil, err
	}
	lockingRepo, ok := s.userSubRepo.(quotaCycleLockingRepository)
	if !ok || s.entClient == nil {
		return nil, nil, infraerrors.ServiceUnavailable("QUOTA_ADVANCE_NOT_READY", "quota cycle advance is not available")
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	if receiptInput != nil {
		receipt, findErr := s.quotaAdvanceReceiptRepo.Find(txCtx, receiptInput.Scope, receiptInput.IdempotencyKeyHash)
		if findErr != nil {
			return nil, nil, findErr
		}
		if receipt != nil {
			if receipt.RequestFingerprint != receiptInput.RequestFingerprint {
				return nil, nil, ErrIdempotencyKeyConflict
			}
			return nil, json.RawMessage(receipt.ResponseBody), nil
		}
	}

	sub, err := lockingRepo.GetByIDForUpdate(txCtx, subscriptionID)
	if err != nil {
		return nil, nil, err
	}
	if sub.UserID != userID {
		return nil, nil, ErrSubscriptionNotFound
	}
	result, err := calculateQuotaCycleAdvance(sub, selection, time.Now())
	if err != nil {
		return nil, nil, err
	}
	if err := s.userSubRepo.Update(txCtx, result.Subscription); err != nil {
		return nil, nil, err
	}

	var response json.RawMessage
	if receiptInput != nil {
		data, buildErr := buildResponse(result)
		if buildErr != nil {
			return nil, nil, buildErr
		}
		response, err = json.Marshal(data)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal quota advance receipt response: %w", err)
		}
		if err := s.quotaAdvanceReceiptRepo.Create(txCtx, &QuotaAdvanceReceipt{
			Scope:              receiptInput.Scope,
			IdempotencyKeyHash: receiptInput.IdempotencyKeyHash,
			RequestFingerprint: receiptInput.RequestFingerprint,
			UserID:             userID,
			SubscriptionID:     subscriptionID,
			Daily:              selection.Daily,
			Weekly:             selection.Weekly,
			Monthly:            selection.Monthly,
			ResponseStatus:     200,
			ResponseBody:       string(response),
			ExpiresAt:          receiptInput.ExpiresAt,
		}); err != nil {
			return nil, nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	s.deferSubscriptionCacheInvalidation(ctx, result.Subscription)
	return result, response, nil
}
