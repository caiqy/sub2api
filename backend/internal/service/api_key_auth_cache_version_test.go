package service

import (
	"context"
	"testing"
)

func TestAPIKeyService_RejectsV10AuthSnapshotWithoutModelsListConfig(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-legacy-models-list", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{
			Version:  10,
			APIKeyID: 1,
			UserID:   2,
			GroupID:  &groupID,
			Status:   StatusActive,
			User: APIKeyAuthUserSnapshot{
				ID:          2,
				Status:      StatusActive,
				Role:        RoleUser,
				Balance:     10,
				Concurrency: 3,
			},
			Group: &APIKeyAuthGroupSnapshot{
				ID:               groupID,
				Name:             "openai",
				Platform:         PlatformOpenAI,
				Status:           StatusActive,
				SubscriptionType: SubscriptionTypeStandard,
				RateMultiplier:   1,
			},
		},
	})

	if err != nil {
		t.Fatalf("expected stale snapshot to be ignored without error, got %v", err)
	}
	if ok {
		t.Fatalf("expected v10 auth snapshot to be rejected after models_list_config was added")
	}
	if apiKey != nil {
		t.Fatalf("expected no API key from stale snapshot, got %#v", apiKey)
	}
}

func TestAPIKeyAuthSnapshotPreservesBatchImageMultipliers(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}
	snapshot := svc.snapshotFromAPIKey(context.Background(), &APIKey{
		ID: 1, UserID: 2, GroupID: &groupID, Status: StatusActive,
		User: &User{ID: 2, Status: StatusActive},
		Group: &Group{
			ID: groupID, Status: StatusActive, IsExclusive: true,
			BatchImageDiscountMultiplier: 0.8,
			BatchImageHoldMultiplier:     0.9,
		},
	})

	apiKey := svc.snapshotToAPIKey("key", snapshot)
	if !apiKey.Group.IsExclusive {
		t.Fatal("exclusive group flag was not preserved")
	}
	if apiKey.Group.BatchImageDiscountMultiplier != 0.8 {
		t.Fatalf("discount multiplier = %v, want 0.8", apiKey.Group.BatchImageDiscountMultiplier)
	}
	if apiKey.Group.BatchImageHoldMultiplier != 0.9 {
		t.Fatalf("hold multiplier = %v, want 0.9", apiKey.Group.BatchImageHoldMultiplier)
	}
}

func TestAPIKeyService_RejectsV17AuthSnapshotWithoutReasoningEffortPolicy(t *testing.T) {
	svc := &APIKeyService{}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-legacy-reasoning-mappings", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{Version: 17},
	})

	if err != nil {
		t.Fatalf("expected stale snapshot to be ignored without error, got %v", err)
	}
	if ok {
		t.Fatal("expected v17 auth snapshot to be rejected after reasoning effort policy was added")
	}
	if apiKey != nil {
		t.Fatalf("expected no API key from stale snapshot, got %#v", apiKey)
	}
}
