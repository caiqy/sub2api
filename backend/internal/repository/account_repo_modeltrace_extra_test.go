package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestModelTraceExtraMergeCleanup(t *testing.T) {
	for _, bulk := range []bool{false, true} {
		for _, tt := range []struct {
			name          string
			extra         map[string]any
			clearMarker   bool
			clearInterval bool
		}{
			{"disable", map[string]any{"modeltrace_quarantine_enabled": false}, true, false},
			{"inherit", map[string]any{"modeltrace_interval_minutes": nil}, false, true},
			{"disable and inherit", map[string]any{"modeltrace_quarantine_enabled": false, "modeltrace_interval_minutes": nil}, true, true},
			{"enable and override", map[string]any{"modeltrace_quarantine_enabled": true, "modeltrace_interval_minutes": 120}, false, false},
			{"null policy", map[string]any{"modeltrace_quarantine_enabled": nil}, false, false},
			{"unrelated", map[string]any{"ordinary": true}, false, false},
		} {
			for _, outboxFails := range []bool{false, true} {
				name := "UpdateExtra/" + tt.name
				if bulk {
					name = "BulkUpdate/" + tt.name
				}
				if outboxFails {
					name += "/outbox failure"
				}
				t.Run(name, func(t *testing.T) {
					var updateSQL string
					db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(expected, actual string) error {
						if strings.HasPrefix(actual, "UPDATE accounts") {
							updateSQL = actual
						}
						return sqlmock.QueryMatcherRegexp.Match(expected, actual)
					})))
					require.NoError(t, err)
					defer func() { _ = db.Close() }()
					client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
					defer func() { _ = client.Close() }()
					repo := newAccountRepositoryWithSQL(client, db, nil)
					extra := copyJSONMap(tt.extra)
					payload, err := json.Marshal(extra)
					require.NoError(t, err)
					extra["modeltrace_quarantined"] = true
					extra["modeltrace_latest"] = "forged"
					mock.ExpectBegin()
					update := mock.ExpectExec(`UPDATE accounts SET extra =`)
					if bulk {
						update.WithArgs(payload, "{27,28}").WillReturnResult(sqlmock.NewResult(0, 2))
					} else {
						update.WithArgs(string(payload), int64(27)).WillReturnResult(sqlmock.NewResult(0, 1))
					}
					outbox := mock.ExpectExec(`INSERT INTO scheduler_outbox`)
					if outboxFails {
						outbox.WillReturnError(errors.New("outbox failed"))
						mock.ExpectRollback()
					} else {
						outbox.WillReturnResult(sqlmock.NewResult(0, 1))
						mock.ExpectCommit()
					}
					if bulk {
						_, err = repo.BulkUpdate(context.Background(), []int64{27, 28}, service.AccountBulkUpdate{Extra: extra})
					} else {
						err = repo.UpdateExtra(context.Background(), 27, extra)
					}
					if outboxFails {
						require.EqualError(t, err, "outbox failed")
					} else {
						require.NoError(t, err)
					}
					require.Equal(t, tt.clearMarker, strings.Contains(updateSQL, "- 'modeltrace_quarantined'"))
					require.Equal(t, tt.clearInterval, strings.Contains(updateSQL, "- 'modeltrace_interval_minutes'"))
					require.NotContains(t, updateSQL, "temp_unschedulable")
					require.NotContains(t, updateSQL, "- 'modeltrace_latest'")
					require.NoError(t, mock.ExpectationsWereMet())
				})
			}
		}
	}
}
