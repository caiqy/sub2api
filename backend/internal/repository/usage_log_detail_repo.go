package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
			upstream_response_headers,
			upstream_response_body,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, detail.UsageLogID, detail.RequestHeaders, detail.RequestBody, detail.UpstreamRequestHeaders, detail.UpstreamRequestBody, detail.ResponseHeaders, detail.ResponseBody, detail.UpstreamResponseHeaders, detail.UpstreamResponseBody, createdAt)
	if err != nil {
		return fmt.Errorf("insert usage log detail: %w", err)
	}
	if err := r.PruneToRecentLimit(ctx, service.UsageLogDetailRetentionLimit); err != nil {
		return fmt.Errorf("prune usage log detail: %w", err)
	}
	return nil
}

func (r *usageLogDetailRepository) CreateBatch(ctx context.Context, details []*service.UsageLogDetail) error {
	if r == nil || r.sql == nil || len(details) == 0 {
		return nil
	}
	query, args := buildUsageLogDetailBatchInsertQuery(details)
	if _, err := r.sql.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("insert usage log detail batch: %w", err)
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
		SELECT usage_log_id, request_headers, request_body, upstream_request_headers, upstream_request_body, response_headers, response_body, upstream_response_headers, upstream_response_body, created_at
		FROM usage_log_details
		WHERE usage_log_id = $1
	`, []any{usageLogID}, &detail.UsageLogID, &detail.RequestHeaders, &detail.RequestBody, &detail.UpstreamRequestHeaders, &detail.UpstreamRequestBody, &detail.ResponseHeaders, &detail.ResponseBody, &detail.UpstreamResponseHeaders, &detail.UpstreamResponseBody, &detail.CreatedAt)
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
		UsageLogID:              usageLogID,
		RequestHeaders:          snapshot.RequestHeaders,
		RequestBody:             snapshot.RequestBody,
		UpstreamRequestHeaders:  snapshot.UpstreamRequestHeaders,
		UpstreamRequestBody:     snapshot.UpstreamRequestBody,
		ResponseHeaders:         snapshot.ResponseHeaders,
		ResponseBody:            snapshot.ResponseBody,
		UpstreamResponseHeaders: snapshot.UpstreamResponseHeaders,
		UpstreamResponseBody:    snapshot.UpstreamResponseBody,
		CreatedAt:               createdAt,
	}
}

func buildUsageLogDetailBatchInsertQuery(details []*service.UsageLogDetail) (string, []any) {
	var query strings.Builder
	_, _ = query.WriteString(`
		INSERT INTO usage_log_details (
			usage_log_id,
			request_headers,
			request_body,
			upstream_request_headers,
			upstream_request_body,
			response_headers,
			response_body,
			upstream_response_headers,
			upstream_response_body,
			created_at
		) VALUES `)

	args := make([]any, 0, len(details)*10)
	argPos := 1
	for idx, detail := range details {
		if idx > 0 {
			_, _ = query.WriteString(",")
		}
		createdAt := detail.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		_, _ = query.WriteString("(")
		for i := 0; i < 10; i++ {
			if i > 0 {
				_, _ = query.WriteString(",")
			}
			_, _ = query.WriteString("$")
			_, _ = query.WriteString(strconv.Itoa(argPos))
			argPos++
		}
		_, _ = query.WriteString(")")
		args = append(args,
			detail.UsageLogID,
			detail.RequestHeaders,
			detail.RequestBody,
			detail.UpstreamRequestHeaders,
			detail.UpstreamRequestBody,
			detail.ResponseHeaders,
			detail.ResponseBody,
			detail.UpstreamResponseHeaders,
			detail.UpstreamResponseBody,
			createdAt,
		)
	}
	_, _ = query.WriteString(`
		ON CONFLICT (usage_log_id) DO NOTHING
	`)
	return query.String(), args
}

func isUsageLogDetailMissing(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
