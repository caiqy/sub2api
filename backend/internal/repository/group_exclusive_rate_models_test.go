package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestExclusiveRateModelsAtomicSave(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback rates failure"}[fail], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			repo := &userGroupRateRepository{sql: db}
			cfg := service.GroupExclusiveRateModels{Enabled: true, Models: []string{"client-A"}}
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE groups SET exclusive_rate_models").WithArgs(int64(2), `{"enabled":true,"models":["client-A"]}`).WillReturnResult(sqlmock.NewResult(0, 1))
			rates := mock.ExpectExec("UPDATE user_group_rate_multipliers").WithArgs(int64(2))
			if fail {
				rates.WillReturnError(errors.New("write failed"))
				mock.ExpectRollback()
			} else {
				rates.WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("DELETE FROM user_group_rate_multipliers").WithArgs(int64(2)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("INSERT INTO scheduler_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			}
			err = repo.SyncGroupRateMultipliersWithModels(context.Background(), 2, nil, cfg)
			if fail {
				require.ErrorContains(t, err, "write failed")
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	err = (&userGroupRateRepository{sql: db}).SyncGroupRateMultipliersWithModels(context.Background(), 2, nil, service.GroupExclusiveRateModels{Enabled: true, Models: []string{" "}})
	require.ErrorContains(t, err, "至少选择一个模型")
	require.NoError(t, mock.ExpectationsWereMet(), "invalid config must not touch storage")
}
