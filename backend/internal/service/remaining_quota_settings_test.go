package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemainingQuotaExclusionsValidationAndStorageErrors(t *testing.T) {
	ctx := context.Background()
	repo := &gatewayRuntimeSettingRepoStub{getValueByKey: map[string]string{SettingKeyRemainingQuotaExclusions: `[2,1,2]`}}
	s := &SettingService{settingRepo: repo}
	ids, err := s.GetRemainingQuotaExclusions(ctx)
	require.NoError(t, err)
	require.Equal(t, []int64{2, 1}, ids)
	for _, invalid := range [][]int64{{0}, {-1}, {9007199254740992}, make([]int64, 1001)} {
		_, err = s.SetRemainingQuotaExclusions(ctx, invalid)
		require.Error(t, err)
	}
	require.Empty(t, repo.setCalls)
	ids, err = s.SetRemainingQuotaExclusions(ctx, []int64{2, 1, 2, 9007199254740991})
	require.NoError(t, err)
	require.Equal(t, []int64{2, 1, 9007199254740991}, ids)
	require.Equal(t, `[2,1,9007199254740991]`, repo.setCalls[0].value)
	_, err = s.SetRemainingQuotaExclusions(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, `[]`, repo.setCalls[1].value)
	repo.setErr = errors.New("db write failure")
	_, err = s.SetRemainingQuotaExclusions(ctx, []int64{1})
	require.ErrorContains(t, err, "db write failure")
	for _, invalid := range []string{`null`, `[0]`, `not json`} {
		repo.getValueByKey[SettingKeyRemainingQuotaExclusions] = invalid
		_, err = s.GetRemainingQuotaExclusions(ctx)
		require.Error(t, err)
	}
	repo.getErrByKey = map[string]error{SettingKeyRemainingQuotaExclusions: ErrSettingNotFound}
	ids, err = s.GetRemainingQuotaExclusions(ctx)
	require.NoError(t, err)
	require.Equal(t, []int64{}, ids)
	repo.getErrByKey[SettingKeyRemainingQuotaExclusions] = errors.New("db read failure")
	_, err = s.GetRemainingQuotaExclusions(ctx)
	require.ErrorContains(t, err, "db read failure")
}
