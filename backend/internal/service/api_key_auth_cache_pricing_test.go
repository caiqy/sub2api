package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyAuthSnapshotPreservesGroupPricingPolicy(t *testing.T) {
	groupID := int64(35)
	inputPrice := 2e-6
	apiKey := &APIKey{
		ID:      297,
		UserID:  1,
		GroupID: &groupID,
		Status:  StatusActive,
		User:    &User{ID: 1, Status: StatusActive},
		Group: &Group{
			ID:                        groupID,
			Platform:                  PlatformOpenAI,
			Status:                    StatusActive,
			LongContextPricingEnabled: true,
			ModelPricing: []ChannelModelPricing{{
				Models:     []string{"gpt-5.6-terra"},
				InputPrice: &inputPrice,
			}},
		},
	}

	svc := &APIKeyService{}
	snapshot := svc.snapshotFromAPIKey(context.Background(), apiKey)
	require.NotNil(t, snapshot)

	payload, err := json.Marshal(&APIKeyAuthCacheEntry{Snapshot: snapshot})
	require.NoError(t, err)
	var cached APIKeyAuthCacheEntry
	require.NoError(t, json.Unmarshal(payload, &cached))

	materialized, used, err := svc.applyAuthCacheEntry("sk-test", &cached)
	require.NoError(t, err)
	require.True(t, used)
	require.NotNil(t, materialized.Group)
	require.True(t, materialized.Group.LongContextPricingEnabled)
	require.Equal(t, apiKey.Group.ModelPricing, materialized.Group.ModelPricing)
}
