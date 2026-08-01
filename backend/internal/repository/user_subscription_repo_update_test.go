package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func TestUserSubscriptionUpdateReturnsRowIterationError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	rowErr := errors.New("returning row interrupted")
	mock.ExpectQuery(`(?s)UPDATE user_subscriptions`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(int64(1)).RowError(0, rowErr))

	repo := &userSubscriptionRepository{client: client}
	err = repo.Update(context.Background(), &service.UserSubscription{
		ID: 1, UserID: 2, GroupID: 3, StartsAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), Status: service.SubscriptionStatusActive,
	})

	require.ErrorIs(t, err, rowErr)
	require.NoError(t, mock.ExpectationsWereMet())
}
