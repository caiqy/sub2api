//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestTempUnschedCache_CompareDeletePreservesNewerState(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := &tempUnschedCache{rdb: rdb}
	accountID := int64(5014)

	expected := &service.TempUnschedState{UntilUnix: time.Now().Add(time.Hour).Unix(), ErrorMessage: "threshold"}
	require.NoError(t, cache.SetTempUnsched(ctx, accountID, expected))
	newer := &service.TempUnschedState{UntilUnix: time.Now().Add(2 * time.Hour).Unix(), ErrorMessage: "rate limit"}
	require.NoError(t, cache.SetTempUnsched(ctx, accountID, newer))

	deleted, err := cache.CompareDeleteTempUnsched(ctx, accountID, expected)

	require.NoError(t, err)
	require.False(t, deleted, "a newer cached generation must survive the threshold release")
	state, err := cache.GetTempUnsched(ctx, accountID)
	require.NoError(t, err)
	require.Equal(t, newer, state)
}
