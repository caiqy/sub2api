package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type usageLogDetailRepository struct {
	sql sqlExecutor
}

func newUsageLogDetailRepositoryWithSQL(sqlq sqlExecutor) *usageLogDetailRepository {
	return &usageLogDetailRepository{sql: sqlq}
}

func (r *usageLogDetailRepository) Create(ctx context.Context, detail *service.UsageLogDetail) error {
	if r == nil || r.sql == nil || detail == nil || detail.UsageLogID <= 0 {
		return nil
	}
	createdAt := detail.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.sql.ExecContext(ctx, `
		INSERT INTO usage_log_details (
			usage_log_id,
			request_headers,
			request_body,
			upstream_request_headers,
			upstream_request_body,
			response_headers,
			response_body,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, detail.UsageLogID, detail.RequestHeaders, detail.RequestBody, detail.UpstreamRequestHeaders, detail.UpstreamRequestBody, detail.ResponseHeaders, detail.ResponseBody, createdAt)
	if err != nil {
		return fmt.Errorf("insert usage log detail: %w", err)
	}
	if err := r.PruneToRecentLimit(ctx, service.UsageLogDetailRetentionLimit); err != nil {
		return fmt.Errorf("prune usage log detail: %w", err)
	}
	return nil
}

func (r *usageLogDetailRepository) GetByUsageLogID(ctx context.Context, usageLogID int64) (*service.UsageLogDetail, error) {
	if r == nil || r.sql == nil {
		return nil, sql.ErrNoRows
	}
	detail := &service.UsageLogDetail{}
	err := scanSingleRow(ctx, r.sql, `
		SELECT usage_log_id, request_headers, request_body, upstream_request_headers, upstream_request_body, response_headers, response_body, created_at
		FROM usage_log_details
		WHERE usage_log_id = $1
	`, []any{usageLogID}, &detail.UsageLogID, &detail.RequestHeaders, &detail.RequestBody, &detail.UpstreamRequestHeaders, &detail.UpstreamRequestBody, &detail.ResponseHeaders, &detail.ResponseBody, &detail.CreatedAt)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (r *usageLogDetailRepository) PruneToRecentLimit(ctx context.Context, limit int) error {
	if r == nil || r.sql == nil || limit <= 0 {
		return nil
	}
	_, err := r.sql.ExecContext(ctx, `
		DELETE FROM usage_log_details
		WHERE id IN (
			SELECT id
			FROM usage_log_details
			ORDER BY created_at DESC, id DESC
			OFFSET $1
		)
	`, limit)
	return err
}

func usageLogDetailFromSnapshot(usageLogID int64, createdAt time.Time, snapshot *service.UsageLogDetailSnapshot) *service.UsageLogDetail {
	if snapshot == nil || usageLogID <= 0 {
		return nil
	}
	return &service.UsageLogDetail{
		UsageLogID:             usageLogID,
		RequestHeaders:         snapshot.RequestHeaders,
		RequestBody:            snapshot.RequestBody,
		UpstreamRequestHeaders: snapshot.UpstreamRequestHeaders,
		UpstreamRequestBody:    snapshot.UpstreamRequestBody,
		ResponseHeaders:        snapshot.ResponseHeaders,
		ResponseBody:           snapshot.ResponseBody,
		CreatedAt:              createdAt,
	}
}

func isUsageLogDetailMissing(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
