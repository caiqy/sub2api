package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// SQL projects one effective request, never either image response. Test emptiness
// before bounding: an oversized nonempty client body must not fall back upstream.
const imageHistoryRequestSummaryProjection = `SELECT d.usage_log_id,
	CASE WHEN octet_length(d.request_body) > 0 THEN
		CASE WHEN octet_length(d.request_body) <= $2 THEN d.request_body ELSE '' END
	ELSE CASE WHEN octet_length(d.upstream_request_body) <= $2 THEN d.upstream_request_body ELSE '' END END,
	CASE WHEN octet_length(d.request_body) > 0 THEN
		CASE WHEN octet_length(d.request_headers) <= $3 THEN d.request_headers ELSE '' END
	ELSE CASE WHEN octet_length(d.upstream_request_headers) <= $3 THEN d.upstream_request_headers ELSE '' END END
	FROM usage_log_details d JOIN usage_logs ul ON ul.id = d.usage_log_id
	WHERE ul.user_id = $1
	AND (COALESCE(ul.inbound_endpoint, '') LIKE '%/images/generations%' OR COALESCE(ul.inbound_endpoint, '') LIKE '%/images/edits%')
	AND d.usage_log_id IN (`

func (r *usageLogRepository) GetImageHistoryRequestSummariesByUser(ctx context.Context, userID int64, usageLogIDs []int64) (out map[int64]service.ImageHistoryRequestSummary, err error) {
	out = make(map[int64]service.ImageHistoryRequestSummary, len(usageLogIDs))
	if len(usageLogIDs) == 0 {
		return out, nil
	}
	args := []any{userID, service.ImageHistoryRequestBodyLimit, service.ImageHistoryRequestHeadersLimit}
	placeholders := make([]string, 0, len(usageLogIDs))
	for _, id := range usageLogIDs {
		args = append(args, id)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	rows, err := r.sql.QueryContext(ctx, imageHistoryRequestSummaryProjection+strings.Join(placeholders, ",")+")", args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			out, err = nil, closeErr
		}
	}()
	for rows.Next() {
		var id int64
		var summary service.ImageHistoryRequestSummary
		if err := rows.Scan(&id, &summary.RequestBody, &summary.RequestHeaders); err != nil {
			return nil, err
		}
		out[id] = summary
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
