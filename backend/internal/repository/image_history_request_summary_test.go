//go:build unit

package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestImageHistoryList_FiveQueriesAtBothPageSizes(t *testing.T) {
	for _, size := range []int{20, 50} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			db, mock := newSQLMock(t)
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			repo := newUsageLogRepositoryWithSQL(client, db)
			where := "WHERE user_id = $1 AND (COALESCE(inbound_endpoint, '') LIKE '%/images/generations%' OR COALESCE(inbound_endpoint, '') LIKE '%/images/edits%')"
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM usage_logs " + where)).WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(size))
			mock.ExpectQuery(regexp.QuoteMeta("SELECT to_regclass('public.usage_log_details') IS NOT NULL")).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			rows := sqlmock.NewRows(usageLogListRowColumns())
			ids := make([]string, size)
			args := []driver.Value{int64(7), service.ImageHistoryRequestBodyLimit, service.ImageHistoryRequestHeadersLimit}
			summaries := sqlmock.NewRows([]string{"usage_log_id", "request_body", "request_headers"})
			for i := range ids {
				id := int64(i + 1)
				values := usageLogListRowValues(true)
				values[0] = id
				rows.AddRow(values...)
				ids[i] = fmt.Sprintf("$%d", i+4)
				args = append(args, id)
				summaries.AddRow(id, `{"prompt":"bounded prompt"}`, "Content-Type: application/json")
			}
			pageSQL := "SELECT " + usageLogListSelectColumns + " FROM usage_logs " + where + " ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3"
			mock.ExpectQuery(regexp.QuoteMeta(pageSQL)).WithArgs(int64(7), size, 0).WillReturnRows(rows)
			// The single Key association query is exercised through the actual Ent driver.
			// No account/user/group/subscription hydration query is permitted.
			mock.ExpectQuery(`SELECT .* FROM "api_keys" WHERE .*`).WithArgs(int64(8)).WillReturnRows(sqlmock.NewRows([]string{"id"}))
			batchSQL := imageHistoryRequestSummaryProjection + strings.Join(ids, ",") + ")"
			mock.ExpectQuery(regexp.QuoteMeta(batchSQL)).WithArgs(args...).WillReturnRows(summaries)
			items, page, err := service.NewImageHistoryService(repo).List(context.Background(), 7, service.ImageHistoryListQuery{PageSize: size})
			require.NoError(t, err)
			require.Len(t, items, size)
			require.Equal(t, int64(size), page.Total)
			for _, item := range items {
				require.Equal(t, "bounded prompt", item.Prompt)
				require.True(t, item.SummaryAvailable)
			}
			for _, query := range []string{pageSQL, batchSQL} {
				require.NotRegexp(t, `(?i)\b(?:response_body|response_headers|upstream_response_body|upstream_response_headers)\b`, query)
			}
			require.NotRegexp(t, `(?i)::json|jsonb?_(?:array|each|path)`, batchSQL)
			require.Contains(t, batchSQL, "octet_length(d.request_body) > 0")
			require.Contains(t, batchSQL, "octet_length(d.request_body) <= $2")
			require.Contains(t, batchSQL, "octet_length(d.request_headers) <= $3")
			require.Contains(t, batchSQL, "WHERE ul.user_id = $1")
			require.NoError(t, mock.ExpectationsWereMet(), "exactly five SQL statements independent of row count")
		})
	}
}

func TestImageHistoryRequestSummary_EmptyAndDriverErrors(t *testing.T) {
	for _, mode := range []string{"empty", "query", "scan", "rows"} {
		t.Run(mode, func(t *testing.T) {
			db, mock := newSQLMock(t)
			repo := &usageLogRepository{sql: db}
			var ids []int64
			if mode != "empty" {
				ids = []int64{11}
				expectation := mock.ExpectQuery(regexp.QuoteMeta(imageHistoryRequestSummaryProjection+"$4)")).WithArgs(int64(7), service.ImageHistoryRequestBodyLimit, service.ImageHistoryRequestHeadersLimit, int64(11))
				switch mode {
				case "query":
					expectation.WillReturnError(errors.New("query failed"))
				case "scan":
					expectation.WillReturnRows(sqlmock.NewRows([]string{"id", "body", "headers"}).AddRow("bad ID", "", ""))
				case "rows":
					expectation.WillReturnRows(sqlmock.NewRows([]string{"id", "body", "headers"}).AddRow(11, "", "").RowError(0, errors.New("row failed")))
				}
			}
			out, err := repo.GetImageHistoryRequestSummariesByUser(context.Background(), 7, ids)
			if mode == "empty" {
				require.NoError(t, err)
				require.Empty(t, out)
			} else {
				require.Error(t, err)
				require.Nil(t, out)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
