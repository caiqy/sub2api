package dto

import (
	"reflect"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserFromServiceAdmin_MapsActivityTimestamps(t *testing.T) {
	t.Parallel()

	lastLoginAt := time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC)
	lastActiveAt := lastLoginAt.Add(15 * time.Minute)
	lastUsedAt := lastLoginAt.Add(45 * time.Minute)

	out := UserFromServiceAdmin(&service.User{
		ID:           42,
		Email:        "admin@example.com",
		Username:     "admin",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
		LastActiveAt: &lastActiveAt,
		LastUsedAt:   &lastUsedAt,
	})

	require.NotNil(t, out)
	require.NotNil(t, out.LastActiveAt)
	require.NotNil(t, out.LastUsedAt)
	require.WithinDuration(t, lastActiveAt, *out.LastActiveAt, time.Second)
	require.WithinDuration(t, lastUsedAt, *out.LastUsedAt, time.Second)
}

func TestUserFromServiceDoesNotExposeBlockedGroups(t *testing.T) {
	out := UserFromService(&service.User{ID: 42, BlockedGroups: []int64{10}})

	require.NotNil(t, out)
	require.NotContains(t, fieldNames(t, *out), "BlockedGroups")
}

func TestUserFromServiceAdminMapsBlockedGroups(t *testing.T) {
	out := UserFromServiceAdmin(&service.User{ID: 42, BlockedGroups: []int64{10}})

	require.NotNil(t, out)
	require.Equal(t, []int64{10}, out.BlockedGroups)
}

func fieldNames(t *testing.T, v any) map[string]struct{} {
	t.Helper()
	typ := reflect.TypeOf(v)
	out := make(map[string]struct{}, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		out[typ.Field(i).Name] = struct{}{}
	}
	return out
}
