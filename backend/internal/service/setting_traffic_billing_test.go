//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrafficBillingSettingsValidation(t *testing.T) {
	ctx := context.Background()
	repo := &settingUpdateRepoStub{updates: map[string]string{SettingKeyTrafficBillingEnabled: "true", SettingKeyTrafficBillingThresholdMB: "10", SettingKeyTrafficBillingPricePerMB: "0.01"}}
	svc := &SettingService{settingRepo: repo}
	defaults := parseTrafficBillingSettings(nil)
	require.False(t, defaults.TrafficBillingEnabled)
	require.Equal(t, 10.0, defaults.TrafficBillingThresholdMB)
	require.Zero(t, defaults.TrafficBillingPricePerMB)
	for _, updates := range []map[string]string{
		{SettingKeyTrafficBillingPricePerMB: "0"},
		{SettingKeyTrafficBillingPricePerMB: "NaN"},
		{SettingKeyTrafficBillingPricePerMB: "+Inf"},
		{SettingKeyTrafficBillingPricePerMB: "-1"},
		{SettingKeyTrafficBillingThresholdMB: "0"},
		{SettingKeyTrafficBillingThresholdMB: "NaN"},
		{SettingKeyTrafficBillingThresholdMB: "+Inf"},
	} {
		require.Error(t, svc.validateTrafficBillingUpdates(ctx, updates))
	}
	require.NoError(t, svc.validateTrafficBillingUpdates(ctx, map[string]string{SettingKeyTrafficBillingThresholdMB: "12.5"}))
	require.NoError(t, svc.validateTrafficBillingUpdates(ctx, map[string]string{SettingKeyTrafficBillingEnabled: "false", SettingKeyTrafficBillingPricePerMB: "0"}))
}

func TestTrafficBillingAuthSnapshotAndProfileInvalidation(t *testing.T) {
	ctx := context.Background()
	repo := &userRepoStub{user: &User{ID: 42, Email: "u@example.com"}}
	invalidator := &authCacheInvalidatorStub{}
	svc := NewUserService(repo, nil, invalidator, nil)
	auth := &APIKeyService{}
	for _, consent := range []bool{true, false} {
		updated, err := svc.UpdateProfile(ctx, 42, UpdateProfileRequest{AllowLargeRequests: &consent})
		require.NoError(t, err)
		require.Equal(t, consent, updated.AllowLargeRequests)
		snapshot := auth.snapshotFromAPIKey(ctx, &APIKey{ID: 1, UserID: 42, User: updated})
		data, err := json.Marshal(snapshot)
		require.NoError(t, err)
		var cached APIKeyAuthSnapshot
		require.NoError(t, json.Unmarshal(data, &cached))
		require.Equal(t, consent, auth.snapshotToAPIKey("key", &cached).User.AllowLargeRequests)
	}
	require.Equal(t, []int64{42, 42}, invalidator.userIDs)
	name := "renamed"
	_, err := svc.UpdateProfile(ctx, 42, UpdateProfileRequest{Username: &name})
	require.NoError(t, err)
	require.Equal(t, []int64{42, 42}, invalidator.userIDs)
}
