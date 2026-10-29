package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExclusiveRateModelsExactMatchBeforeCache(t *testing.T) {
	rate := 0.12
	repo := &userGroupRateResolverRepoStub{rate: &rate}
	svc := &GatewayService{userGroupRateResolver: newUserGroupRateResolver(repo, nil, time.Minute, nil, "test")}
	group := &Group{ID: 2, RateMultiplier: 0.225, ExclusiveRateModels: GroupExclusiveRateModels{Enabled: true, Models: []string{"client-A", "literal-*"}}}
	for _, model := range []string{"client-A", "upstream-B", "CLIENT-A", "client-A-extra", "literal-match", "client-A", ""} {
		want := group.RateMultiplier
		if model == "client-A" {
			want = rate
		}
		require.Equal(t, want, svc.ResolveUserGroupModelRateMultiplier(context.Background(), 1, group, model), model)
	}
	require.Equal(t, 1, repo.calls)
	group.ExclusiveRateModels.Models = []string{"upstream-B"}
	require.Equal(t, group.RateMultiplier, svc.ResolveUserGroupModelRateMultiplier(context.Background(), 1, group, "client-A"))
	group.ExclusiveRateModels.Enabled = false
	require.Equal(t, rate, svc.ResolveUserGroupModelRateMultiplier(context.Background(), 1, group, "anything"))
	require.Equal(t, []string{"upstream-B"}, group.ExclusiveRateModels.Models)
}

type exclusiveRatesByUserGroup struct{ UserGroupRateRepository }

func (exclusiveRatesByUserGroup) GetByUserAndGroup(_ context.Context, userID, groupID int64) (*float64, error) {
	rates := map[[2]int64]float64{{1, 2}: 0.12, {2, 2}: 0.18, {1, 3}: 0.3}
	rate, ok := rates[[2]int64{userID, groupID}]
	if !ok {
		return nil, nil
	}
	return &rate, nil
}
func TestExclusiveRateModelsUserAndGroupIsolation(t *testing.T) {
	svc := &GatewayService{userGroupRateResolver: newUserGroupRateResolver(exclusiveRatesByUserGroup{}, nil, time.Minute, nil, "test")}
	group := &Group{ID: 2, RateMultiplier: 0.225, ExclusiveRateModels: GroupExclusiveRateModels{Enabled: true, Models: []string{"client-A"}}}
	for _, tc := range []struct {
		user, group int64
		model       string
		want        float64
	}{
		{1, 2, "client-A", 0.12}, {2, 2, "client-A", 0.18}, {1, 3, "client-A", 0.3}, {3, 2, "client-A", 0.225},
		{1, 2, "client-B", 0.225}, {2, 2, "client-B", 0.225}, {1, 2, "client-A", 0.12},
	} {
		current := *group
		current.ID = tc.group
		require.Equal(t, tc.want, svc.ResolveUserGroupModelRateMultiplier(context.Background(), tc.user, &current, tc.model))
	}
}

func TestExclusiveRateModelsAuthSnapshotRoundTrip(t *testing.T) {
	svc := &APIKeyService{}
	cfg := GroupExclusiveRateModels{Enabled: true, Models: []string{"client-A", "client-B"}}
	key := &APIKey{ID: 3, UserID: 1, User: &User{ID: 1}, Group: &Group{ID: 2, ExclusiveRateModels: cfg}}
	encoded, err := json.Marshal(svc.snapshotFromAPIKey(context.Background(), key))
	require.NoError(t, err)
	var snapshot APIKeyAuthSnapshot
	require.NoError(t, json.Unmarshal(encoded, &snapshot))
	require.Equal(t, apiKeyAuthSnapshotVersion, snapshot.Version)
	require.Equal(t, cfg, svc.snapshotToAPIKey("test-key", &snapshot).Group.ExclusiveRateModels)
}

func TestNormalizeExclusiveRateModels(t *testing.T) {
	for _, models := range [][]string{nil, {}, {"", "   "}} {
		_, err := NormalizeGroupExclusiveRateModels(GroupExclusiveRateModels{Enabled: true, Models: models})
		require.ErrorContains(t, err, "至少选择一个模型")
	}
	cfg, err := NormalizeGroupExclusiveRateModels(GroupExclusiveRateModels{Enabled: true, Models: []string{" client-A ", "client-A", "CLIENT-A"}})
	require.NoError(t, err)
	require.Equal(t, []string{"client-A", "CLIENT-A"}, cfg.Models)
}
