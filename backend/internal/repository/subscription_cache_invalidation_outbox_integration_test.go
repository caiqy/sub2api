//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	group := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("subscription-outbox-group-%d", suffix), RateMultiplier: 1})
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("subscription-outbox-%d@example.com", suffix), Concurrency: 1})
	startsAt := time.Now().Add(-time.Hour)
	expiresAt := time.Now().Add(24 * time.Hour)
	sub, err := integrationEntClient.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(startsAt).
		SetExpiresAt(expiresAt).
		SetStatus(service.SubscriptionStatusActive).
		Save(ctx)
	require.NoError(t, err)

	clear := func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM subscription_cache_invalidation_outbox WHERE user_id = $1 AND group_id = $2", user.ID, group.ID)
		require.NoError(t, err)
	}
	count := func() int {
		var value int
		require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM subscription_cache_invalidation_outbox WHERE user_id = $1 AND group_id = $2", user.ID, group.ID).Scan(&value))
		return value
	}
	t.Cleanup(clear)
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM user_subscriptions WHERE id = $1", sub.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", user.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM groups WHERE id = $1", group.ID)
		require.NoError(t, err)
	})

	clear()
	_, err = integrationDB.ExecContext(ctx, "UPDATE user_subscriptions SET daily_usage_usd = daily_usage_usd + 1 WHERE id = $1", sub.ID)
	require.NoError(t, err)
	require.Zero(t, count(), "ordinary usage increases must not enqueue invalidations")

	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "UPDATE user_subscriptions SET daily_usage_usd = 0, daily_window_start = NOW() WHERE id = $1", sub.ID)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "INSERT INTO user_allowed_groups (user_id, group_id) VALUES (0, 0)")
	require.Error(t, err)
	require.Error(t, tx.Commit())
	require.Zero(t, count(), "a failed business transaction must not leave an outbox event")

	for _, change := range []struct {
		name string
		sql  string
	}{
		{"usage decrease", "UPDATE user_subscriptions SET daily_usage_usd = 0, daily_window_start = NOW() WHERE id = $1"},
		{"term", "UPDATE user_subscriptions SET expires_at = expires_at + interval '1 day' WHERE id = $1"},
		{"status", "UPDATE user_subscriptions SET status = 'suspended' WHERE id = $1"},
		{"daily window", "UPDATE user_subscriptions SET daily_window_start = NOW() + interval '1 day' WHERE id = $1"},
		{"weekly window", "UPDATE user_subscriptions SET weekly_window_start = NOW() + interval '1 day' WHERE id = $1"},
		{"monthly window", "UPDATE user_subscriptions SET monthly_window_start = NOW() + interval '1 day' WHERE id = $1"},
		{"soft delete", "UPDATE user_subscriptions SET deleted_at = NOW() WHERE id = $1"},
	} {
		t.Run(change.name, func(t *testing.T) {
			clear()
			_, err := integrationDB.ExecContext(ctx, change.sql, sub.ID)
			require.NoError(t, err)
			require.Equal(t, 1, count(), "%s must enqueue invalidation", change.name)
		})
	}
}

func TestSubscriptionCacheInvalidationOutbox_LeaseCanBeTakenOverAfterWorkerCrash(t *testing.T) {
	ctx := context.Background()
	userID, groupID := int64(991001), int64(991002)
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO subscription_cache_invalidation_outbox (user_id, group_id, version)
		VALUES ($1, $2, 100)`, userID, groupID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM subscription_cache_invalidation_outbox WHERE user_id = $1 AND group_id = $2", userID, groupID)
		require.NoError(t, err)
	})

	repo := NewSubscriptionCacheInvalidationOutboxRepository(integrationDB)
	claimed, err := repo.Claim(ctx, "crashed-worker", 1, time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	_, err = integrationDB.ExecContext(ctx, "UPDATE subscription_cache_invalidation_outbox SET claimed_at = NOW() - interval '2 seconds' WHERE id = $1", claimed[0].ID)
	require.NoError(t, err)

	takenOver, err := repo.Claim(ctx, "recovery-worker", 1, time.Second)
	require.NoError(t, err)
	require.Len(t, takenOver, 1)
	require.Equal(t, claimed[0].ID, takenOver[0].ID)
}

func TestSubscriptionCacheInvalidationOutbox_UserGroupMoveInvalidatesOldAndNewKeys(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	oldGroup := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("subscription-outbox-old-group-%d", suffix), RateMultiplier: 1})
	newGroup := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("subscription-outbox-new-group-%d", suffix), RateMultiplier: 1})
	oldUser := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("subscription-outbox-old-%d@example.com", suffix), Concurrency: 1})
	newUser := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("subscription-outbox-new-%d@example.com", suffix), Concurrency: 1})
	sub, err := integrationEntClient.UserSubscription.Create().
		SetUserID(oldUser.ID).
		SetGroupID(oldGroup.ID).
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		SetStatus(service.SubscriptionStatusActive).
		Save(ctx)
	require.NoError(t, err)

	clear := func() {
		_, err := integrationDB.ExecContext(ctx, `
			DELETE FROM subscription_cache_invalidation_outbox
			WHERE (user_id = $1 AND group_id = $2) OR (user_id = $3 AND group_id = $4)`,
			oldUser.ID, oldGroup.ID, newUser.ID, newGroup.ID)
		require.NoError(t, err)
	}
	clear()
	t.Cleanup(clear)
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM user_subscriptions WHERE id = $1", sub.ID)
		require.NoError(t, err)
		for _, id := range []int64{oldUser.ID, newUser.ID} {
			_, err = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
			require.NoError(t, err)
		}
		for _, id := range []int64{oldGroup.ID, newGroup.ID} {
			_, err = integrationDB.ExecContext(ctx, "DELETE FROM groups WHERE id = $1", id)
			require.NoError(t, err)
		}
	})

	_, err = integrationDB.ExecContext(ctx, "UPDATE user_subscriptions SET user_id = $2, group_id = $3 WHERE id = $1", sub.ID, newUser.ID, newGroup.ID)
	require.NoError(t, err)
	rows, err := integrationDB.QueryContext(ctx, `
		SELECT user_id, group_id, version
		FROM subscription_cache_invalidation_outbox
		WHERE (user_id = $1 AND group_id = $2) OR (user_id = $3 AND group_id = $4)
		ORDER BY user_id, group_id`, oldUser.ID, oldGroup.ID, newUser.ID, newGroup.ID)
	require.NoError(t, err)
	defer rows.Close()

	got := make(map[[2]int64]int64)
	for rows.Next() {
		var userID, groupID, version int64
		require.NoError(t, rows.Scan(&userID, &groupID, &version))
		got[[2]int64{userID, groupID}] = version
	}
	require.NoError(t, rows.Err())
	require.Greater(t, got[[2]int64{oldUser.ID, oldGroup.ID}], int64(0))
	require.Greater(t, got[[2]int64{newUser.ID, newGroup.ID}], int64(0))
}

func TestSubscriptionCacheVersionWatermark_UsageIncreaseAdvancesVersionWithoutOutbox(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	group := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("subscription-watermark-group-%d", suffix), RateMultiplier: 1})
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("subscription-watermark-%d@example.com", suffix), Concurrency: 1})
	sub, err := integrationEntClient.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		SetStatus(service.SubscriptionStatusActive).
		Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM user_subscriptions WHERE id = $1", sub.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM subscription_cache_invalidation_outbox WHERE user_id = $1 AND group_id = $2", user.ID, group.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", user.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM groups WHERE id = $1", group.ID)
		require.NoError(t, err)
	})

	var initialWatermark time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT watermark_at FROM subscription_cache_version_watermarks
		WHERE user_id = $1 AND group_id = $2`, user.ID, group.ID).Scan(&initialWatermark))
	require.Equal(t, sub.UpdatedAt.UnixNano(), initialWatermark.UnixNano())
	_, err = integrationDB.ExecContext(ctx, "DELETE FROM subscription_cache_invalidation_outbox WHERE user_id = $1 AND group_id = $2", user.ID, group.ID)
	require.NoError(t, err)

	var updatedAt time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		UPDATE user_subscriptions
		SET daily_usage_usd = daily_usage_usd + 1
		WHERE id = $1
		RETURNING updated_at`, sub.ID).Scan(&updatedAt))
	var watermark time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT watermark_at FROM subscription_cache_version_watermarks
		WHERE user_id = $1 AND group_id = $2`, user.ID, group.ID).Scan(&watermark))
	require.Greater(t, updatedAt.UnixNano(), initialWatermark.UnixNano())
	require.Equal(t, updatedAt.UnixNano(), watermark.UnixNano())
	var outboxCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM subscription_cache_invalidation_outbox
		WHERE user_id = $1 AND group_id = $2`, user.ID, group.ID).Scan(&outboxCount))
	require.Zero(t, outboxCount)
}

func TestSubscriptionCacheVersionWatermark_SemanticWritesAndDeleteRecreateStrictlyIncrease(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	group := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("subscription-watermark-term-group-%d", suffix), RateMultiplier: 1})
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("subscription-watermark-term-%d@example.com", suffix), Concurrency: 1})
	create := func() *time.Time {
		sub, err := integrationEntClient.UserSubscription.Create().
			SetUserID(user.ID).
			SetGroupID(group.ID).
			SetStartsAt(time.Now().Add(-time.Hour)).
			SetExpiresAt(time.Now().Add(24 * time.Hour)).
			SetStatus(service.SubscriptionStatusActive).
			Save(ctx)
		require.NoError(t, err)
		return &sub.UpdatedAt
	}
	firstVersion := create()
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM user_subscriptions WHERE user_id = $1 AND group_id = $2", user.ID, group.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM subscription_cache_invalidation_outbox WHERE user_id = $1 AND group_id = $2", user.ID, group.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", user.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM groups WHERE id = $1", group.ID)
		require.NoError(t, err)
	})

	var firstID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT id FROM user_subscriptions WHERE user_id = $1 AND group_id = $2", user.ID, group.ID).Scan(&firstID))
	var termVersion time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		UPDATE user_subscriptions SET expires_at = expires_at + interval '1 day'
		WHERE id = $1 RETURNING updated_at`, firstID).Scan(&termVersion))
	var statusVersion time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		UPDATE user_subscriptions SET status = 'suspended'
		WHERE id = $1 RETURNING updated_at`, firstID).Scan(&statusVersion))
	var deleteVersion time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		UPDATE user_subscriptions SET deleted_at = NOW()
		WHERE id = $1 RETURNING updated_at`, firstID).Scan(&deleteVersion))
	secondVersion := create()

	require.Greater(t, termVersion.UnixNano(), firstVersion.UnixNano())
	require.Greater(t, statusVersion.UnixNano(), termVersion.UnixNano())
	require.Greater(t, deleteVersion.UnixNano(), statusVersion.UnixNano())
	require.Greater(t, secondVersion.UnixNano(), deleteVersion.UnixNano())
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	var firstTxVersion, secondTxVersion, nextTxVersion time.Time
	require.NoError(t, tx.QueryRowContext(ctx, `
		UPDATE user_subscriptions SET daily_usage_usd = daily_usage_usd + 1
		WHERE user_id = $1 AND group_id = $2 AND deleted_at IS NULL
		RETURNING updated_at`, user.ID, group.ID).Scan(&firstTxVersion))
	require.NoError(t, tx.QueryRowContext(ctx, `
		UPDATE user_subscriptions SET daily_usage_usd = daily_usage_usd + 1
		WHERE user_id = $1 AND group_id = $2 AND deleted_at IS NULL
		RETURNING updated_at`, user.ID, group.ID).Scan(&secondTxVersion))
	require.NoError(t, tx.Commit())
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		UPDATE user_subscriptions SET daily_usage_usd = daily_usage_usd + 1
		WHERE user_id = $1 AND group_id = $2 AND deleted_at IS NULL
		RETURNING updated_at`, user.ID, group.ID).Scan(&nextTxVersion))
	require.Greater(t, firstTxVersion.UnixNano(), secondVersion.UnixNano())
	require.GreaterOrEqual(t, secondTxVersion.UnixNano()-firstTxVersion.UnixNano(), int64(1000))
	require.GreaterOrEqual(t, nextTxVersion.UnixNano()-secondTxVersion.UnixNano(), int64(1000))
	var watermark time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT watermark_at FROM subscription_cache_version_watermarks
		WHERE user_id = $1 AND group_id = $2`, user.ID, group.ID).Scan(&watermark))
	require.Equal(t, nextTxVersion.UnixNano(), watermark.UnixNano())
}

func TestSubscriptionCacheVersionWatermark_MoveUsesOldDeleteAndNewVersions(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	oldGroup := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("subscription-watermark-old-group-%d", suffix), RateMultiplier: 1})
	newGroup := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("subscription-watermark-new-group-%d", suffix), RateMultiplier: 1})
	oldUser := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("subscription-watermark-old-%d@example.com", suffix), Concurrency: 1})
	newUser := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("subscription-watermark-new-%d@example.com", suffix), Concurrency: 1})
	sub, err := integrationEntClient.UserSubscription.Create().
		SetUserID(oldUser.ID).
		SetGroupID(oldGroup.ID).
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		SetStatus(service.SubscriptionStatusActive).
		Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM user_subscriptions WHERE id = $1", sub.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM subscription_cache_invalidation_outbox WHERE user_id IN ($1, $2)", oldUser.ID, newUser.ID)
		require.NoError(t, err)
		for _, id := range []int64{oldUser.ID, newUser.ID} {
			_, err = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
			require.NoError(t, err)
		}
		for _, id := range []int64{oldGroup.ID, newGroup.ID} {
			_, err = integrationDB.ExecContext(ctx, "DELETE FROM groups WHERE id = $1", id)
			require.NoError(t, err)
		}
	})
	_, err = integrationDB.ExecContext(ctx, "DELETE FROM subscription_cache_invalidation_outbox WHERE user_id = $1 AND group_id = $2", oldUser.ID, oldGroup.ID)
	require.NoError(t, err)

	var newVersion time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		UPDATE user_subscriptions SET user_id = $2, group_id = $3
		WHERE id = $1 RETURNING updated_at`, sub.ID, newUser.ID, newGroup.ID).Scan(&newVersion))
	rows, err := integrationDB.QueryContext(ctx, `
		SELECT user_id, group_id, version FROM subscription_cache_invalidation_outbox
		WHERE (user_id = $1 AND group_id = $2) OR (user_id = $3 AND group_id = $4)`,
		oldUser.ID, oldGroup.ID, newUser.ID, newGroup.ID)
	require.NoError(t, err)
	defer rows.Close()
	versions := make(map[[2]int64]int64)
	for rows.Next() {
		var userID, groupID, version int64
		require.NoError(t, rows.Scan(&userID, &groupID, &version))
		versions[[2]int64{userID, groupID}] = version
	}
	require.NoError(t, rows.Err())
	require.Greater(t, versions[[2]int64{oldUser.ID, oldGroup.ID}], sub.UpdatedAt.UnixNano())
	require.Equal(t, newVersion.UnixNano(), versions[[2]int64{newUser.ID, newGroup.ID}])
}

func TestSubscriptionCacheInvalidationOutbox_ConcurrentClaimOwnershipAndStages(t *testing.T) {
	ctx := context.Background()
	userID, groupID := int64(991101), int64(991102)
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO subscription_cache_invalidation_outbox (user_id, group_id, version)
		VALUES ($1, $2, 1), ($1, $2, 2), ($1, $2, 3)`, userID, groupID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM subscription_cache_invalidation_outbox WHERE user_id = $1 AND group_id = $2", userID, groupID)
		require.NoError(t, err)
	})

	repo := NewSubscriptionCacheInvalidationOutboxRepository(integrationDB)
	start := make(chan struct{})
	type claimResult struct {
		events []service.SubscriptionCacheInvalidationEvent
		err    error
	}
	claimed := make(chan claimResult, 2)
	var wg sync.WaitGroup
	for _, workerID := range []string{"worker-a", "worker-b"} {
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()
			<-start
			events, claimErr := repo.Claim(ctx, workerID, 1, time.Minute)
			claimed <- claimResult{events: events, err: claimErr}
		}(workerID)
	}
	close(start)
	wg.Wait()
	close(claimed)
	var ids []int64
	for result := range claimed {
		require.NoError(t, result.err)
		require.Len(t, result.events, 1)
		ids = append(ids, result.events[0].ID)
	}
	require.Len(t, ids, 2)
	require.NotEqual(t, ids[0], ids[1], "concurrent workers must not claim the same event")

	stageEvents, err := repo.Claim(ctx, "stage-owner", 1, time.Minute)
	require.NoError(t, err)
	require.Len(t, stageEvents, 1)
	event := stageEvents[0]
	require.Error(t, repo.RetryClaimed(ctx, event.ID, "other-worker", time.Now(), "lost claim"))
	require.NoError(t, repo.ScheduleSecondPass(ctx, event.ID, "stage-owner", time.Now()))
	var stage int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT delivery_stage FROM subscription_cache_invalidation_outbox WHERE id = $1", event.ID).Scan(&stage))
	require.Equal(t, 1, stage, "first pass must retain the event")

	stageTwo, err := repo.Claim(ctx, "stage-two", 1, time.Minute)
	require.NoError(t, err)
	require.Len(t, stageTwo, 1)
	require.Equal(t, event.ID, stageTwo[0].ID)
	require.Equal(t, 1, stageTwo[0].Stage)
	require.NoError(t, repo.DeleteClaimed(ctx, event.ID, "stage-two"))
	var count int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM subscription_cache_invalidation_outbox WHERE id = $1", event.ID).Scan(&count))
	require.Zero(t, count, "second pass may acknowledge the event")
}

func TestSubscriptionCacheInvalidationMigration_RawRerunIsIdempotent(t *testing.T) {
	db := newEmptyIsolatedMigrationDB(t)
	ctx := context.Background()
	require.NoError(t, ApplyMigrations(ctx, db))
	migration, err := dbmigrations.FS.ReadFile("192_subscription_cache_invalidation_outbox.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)
}

func TestSubscriptionCacheInvalidationMigration_RawRerunBackfillsMaximumWithoutRegressingWatermarks(t *testing.T) {
	db := newEmptyIsolatedMigrationDB(t)
	ctx := context.Background()
	require.NoError(t, ApplyMigrations(ctx, db))
	migration, err := dbmigrations.FS.ReadFile("192_subscription_cache_invalidation_outbox.sql")
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash) VALUES
		('subscription-rerun-low@example.com', 'test'),
		('subscription-rerun-high@example.com', 'test')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO groups (name) VALUES ('subscription-rerun-low'), ('subscription-rerun-high')`)
	require.NoError(t, err)
	var lowUserID, highUserID, lowGroupID, highGroupID int64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = 'subscription-rerun-low@example.com'").Scan(&lowUserID))
	require.NoError(t, db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = 'subscription-rerun-high@example.com'").Scan(&highUserID))
	require.NoError(t, db.QueryRowContext(ctx, "SELECT id FROM groups WHERE name = 'subscription-rerun-low'").Scan(&lowGroupID))
	require.NoError(t, db.QueryRowContext(ctx, "SELECT id FROM groups WHERE name = 'subscription-rerun-high'").Scan(&highGroupID))

	_, err = db.ExecContext(ctx, "ALTER TABLE user_subscriptions DISABLE TRIGGER USER")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := db.ExecContext(ctx, "ALTER TABLE user_subscriptions ENABLE TRIGGER USER")
		require.NoError(t, cleanupErr)
	})
	lowMax := time.Date(2024, 1, 2, 3, 4, 5, 6000, time.UTC)
	highMax := time.Date(2024, 2, 3, 4, 5, 6, 7000, time.UTC)
	lowWatermark := lowMax.Add(-time.Hour)
	highWatermark := highMax.Add(time.Hour)
	for _, row := range []struct {
		userID, groupID int64
		updatedAt       time.Time
		watermark       time.Time
	}{
		{lowUserID, lowGroupID, lowMax, lowWatermark},
		{highUserID, highGroupID, highMax, highWatermark},
	} {
		_, err = db.ExecContext(ctx, `
			INSERT INTO user_subscriptions (user_id, group_id, starts_at, expires_at, status, notes, updated_at)
			VALUES ($1, $2, NOW(), NOW() + interval '1 day', 'active', '', $3)`, row.userID, row.groupID, row.updatedAt)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `
			INSERT INTO subscription_cache_version_watermarks (user_id, group_id, watermark_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, group_id) DO UPDATE SET watermark_at = EXCLUDED.watermark_at`, row.userID, row.groupID, row.watermark)
		require.NoError(t, err)
	}

	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	for _, expected := range []struct {
		userID, groupID int64
		watermark       time.Time
	}{
		{lowUserID, lowGroupID, lowMax},
		{highUserID, highGroupID, highWatermark},
	} {
		var actual time.Time
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT watermark_at FROM subscription_cache_version_watermarks WHERE user_id = $1 AND group_id = $2`, expected.userID, expected.groupID).Scan(&actual))
		require.Equal(t, expected.watermark.UnixNano(), actual.UnixNano())
	}

	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	var rerunWatermark time.Time
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT watermark_at FROM subscription_cache_version_watermarks WHERE user_id = $1 AND group_id = $2`, highUserID, highGroupID).Scan(&rerunWatermark))
	require.Equal(t, highWatermark.UnixNano(), rerunWatermark.UnixNano(), "raw reruns must not regress a newer watermark")
}

func TestSubscriptionCacheVersionWatermark_ConcurrentSameKeyWritesAreStrictlyUnique(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	group := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("subscription-concurrent-version-group-%d", suffix), RateMultiplier: 1})
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("subscription-concurrent-version-%d@example.com", suffix), Concurrency: 1})
	sub, err := integrationEntClient.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		SetStatus(service.SubscriptionStatusActive).
		Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := integrationDB.ExecContext(ctx, "DELETE FROM user_subscriptions WHERE id = $1", sub.ID)
		require.NoError(t, cleanupErr)
		_, cleanupErr = integrationDB.ExecContext(ctx, "DELETE FROM subscription_cache_invalidation_outbox WHERE user_id = $1 AND group_id = $2", user.ID, group.ID)
		require.NoError(t, cleanupErr)
		_, cleanupErr = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", user.ID)
		require.NoError(t, cleanupErr)
		_, cleanupErr = integrationDB.ExecContext(ctx, "DELETE FROM groups WHERE id = $1", group.ID)
		require.NoError(t, cleanupErr)
	})

	const writers = 8
	versions := make(chan int64, writers)
	errs := make(chan error, writers)
	start := make(chan struct{})
	for range writers {
		go func() {
			<-start
			var updatedAt time.Time
			err := integrationDB.QueryRowContext(ctx, `
				UPDATE user_subscriptions SET daily_usage_usd = daily_usage_usd + 1
				WHERE id = $1 RETURNING updated_at`, sub.ID).Scan(&updatedAt)
			if err == nil {
				versions <- updatedAt.UnixNano()
			}
			errs <- err
		}()
	}
	close(start)
	got := make([]int64, 0, writers)
	for range writers {
		require.NoError(t, <-errs)
		got = append(got, <-versions)
	}
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	for i := 1; i < len(got); i++ {
		require.Greater(t, got[i], got[i-1], "concurrent same-key writes must receive unique increasing versions")
	}
}

func TestSubscriptionCacheInvalidationOutbox_RetryAndStageOwnership(t *testing.T) {
	ctx := context.Background()
	userID, groupID := int64(991201), int64(991202)
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO subscription_cache_invalidation_outbox (user_id, group_id, version)
		VALUES ($1, $2, 1)`, userID, groupID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := integrationDB.ExecContext(ctx, "DELETE FROM subscription_cache_invalidation_outbox WHERE user_id = $1 AND group_id = $2", userID, groupID)
		require.NoError(t, cleanupErr)
	})

	repo := NewSubscriptionCacheInvalidationOutboxRepository(integrationDB)
	claimed, err := repo.Claim(ctx, "owner", 1, time.Minute)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	retryAt := time.Now().UTC().Add(time.Minute).Truncate(time.Microsecond)
	require.NoError(t, repo.RetryClaimed(ctx, claimed[0].ID, "owner", retryAt, "redis unavailable"))
	var attempts, stage int
	var availableAt time.Time
	var lastError, claimedBy sql.NullString
	var claimedAt sql.NullTime
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT attempts, delivery_stage, available_at, last_error, claimed_at, claimed_by
		FROM subscription_cache_invalidation_outbox WHERE id = $1`, claimed[0].ID).Scan(&attempts, &stage, &availableAt, &lastError, &claimedAt, &claimedBy))
	require.Equal(t, 1, attempts)
	require.Zero(t, stage)
	require.GreaterOrEqual(t, availableAt.UnixNano(), retryAt.UnixNano())
	require.Equal(t, "redis unavailable", lastError.String)
	require.False(t, claimedAt.Valid)
	require.False(t, claimedBy.Valid)
	other, err := repo.Claim(ctx, "other", 1, time.Minute)
	require.NoError(t, err)
	require.Empty(t, other, "retry delay must prevent early claims")
	require.Error(t, repo.RetryClaimed(ctx, claimed[0].ID, "other", time.Now(), "lost ownership"))

	_, err = integrationDB.ExecContext(ctx, "UPDATE subscription_cache_invalidation_outbox SET available_at = NOW() WHERE id = $1", claimed[0].ID)
	require.NoError(t, err)
	stageZero, err := repo.Claim(ctx, "stage-owner", 1, time.Minute)
	require.NoError(t, err)
	require.Len(t, stageZero, 1)
	safetyAt := time.Now().UTC().Add(time.Minute).Truncate(time.Microsecond)
	require.NoError(t, repo.ScheduleSecondPass(ctx, stageZero[0].ID, "stage-owner", safetyAt))
	var remaining int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM subscription_cache_invalidation_outbox WHERE id = $1", stageZero[0].ID).Scan(&remaining))
	require.Equal(t, 1, remaining, "stage 1 completion must retain the event")
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT attempts, delivery_stage, available_at, claimed_at, claimed_by
		FROM subscription_cache_invalidation_outbox WHERE id = $1`, stageZero[0].ID).Scan(&attempts, &stage, &availableAt, &claimedAt, &claimedBy))
	require.Equal(t, 1, attempts)
	require.Equal(t, 1, stage)
	require.GreaterOrEqual(t, availableAt.UnixNano(), safetyAt.UnixNano())
	require.False(t, claimedAt.Valid)
	require.False(t, claimedBy.Valid)
	require.Empty(t, mustClaimSubscriptionInvalidation(t, repo, ctx, "early-stage-two", 1), "safety delay must prevent early stage 2 claims")

	_, err = integrationDB.ExecContext(ctx, "UPDATE subscription_cache_invalidation_outbox SET available_at = NOW() WHERE id = $1", stageZero[0].ID)
	require.NoError(t, err)
	stageTwo := mustClaimSubscriptionInvalidation(t, repo, ctx, "stage-two", 1)
	require.Len(t, stageTwo, 1)
	require.Equal(t, 1, stageTwo[0].Stage)
	require.Error(t, repo.DeleteClaimed(ctx, stageTwo[0].ID, "other"))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM subscription_cache_invalidation_outbox WHERE id = $1", stageTwo[0].ID).Scan(&remaining))
	require.Equal(t, 1, remaining)
	require.NoError(t, repo.DeleteClaimed(ctx, stageTwo[0].ID, "stage-two"))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM subscription_cache_invalidation_outbox WHERE id = $1", stageTwo[0].ID).Scan(&remaining))
	require.Zero(t, remaining, "stage 2 completion may delete the event")
}

func mustClaimSubscriptionInvalidation(t *testing.T, repo service.SubscriptionCacheInvalidationOutboxRepository, ctx context.Context, workerID string, limit int) []service.SubscriptionCacheInvalidationEvent {
	t.Helper()
	events, err := repo.Claim(ctx, workerID, limit, time.Minute)
	require.NoError(t, err)
	return events
}
