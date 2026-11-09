package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Read both parts from one snapshot, including disabled users and negative balances.
func (r *userRepository) ReadRemainingQuota(ctx context.Context, excluded []int64, now time.Time) (float64, []service.UserSubscription, error) {
	tx, err := r.client.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()
	client := tx.Client()
	users := client.User.Query().Where(user.DeletedAtIsNil())
	subs := client.UserSubscription.Query().Where(usersubscription.DeletedAtIsNil(), usersubscription.StatusEQ(service.SubscriptionStatusActive), usersubscription.ExpiresAtGT(now), usersubscription.HasUserWith(user.DeletedAtIsNil()))
	if len(excluded) > 0 {
		users.Where(user.IDNotIn(excluded...))
		subs.Where(usersubscription.UserIDNotIn(excluded...))
	}
	var sums []struct {
		Balance float64 `json:"balance"`
	}
	err = users.Aggregate(func(s *entsql.Selector) string {
		return entsql.As("COALESCE("+entsql.Sum(s.C(user.FieldBalance))+", 0)", "balance")
	}).Scan(ctx, &sums)
	if err != nil {
		return 0, nil, fmt.Errorf("sum user balances: %w", err)
	}
	models, err := subs.WithGroup().All(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("load subscription quota windows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}
	return sums[0].Balance, userSubscriptionEntitiesToService(models), nil
}
