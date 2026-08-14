//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type cancelAwareSettingRepo struct{ *mockSettingRepo }

func (r *cancelAwareSettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.mockSettingRepo.GetAll(ctx)
}

func newSettingServiceForPlatformThresholdTest(seed map[string]string) *SettingService {
	accountSchedulingThresholdsSF.Forget(SettingKeyAccountSchedulingThresholds)
	accountSchedulingThresholdsCache.Store(&cachedAccountSchedulingThresholds{})
	repo := newMockSettingRepo()
	for k, v := range seed {
		repo.data[k] = v
	}
	return NewSettingService(repo, &config.Config{})
}

func TestPlatformSchedulingThresholds_RoundTrip_DefaultsAndStoredValues(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(nil)

	got := svc.parseSettings(map[string]string{})
	require.Equal(t, map[string]int{
		PlatformOpenAI:    100,
		PlatformAnthropic: 100,
		PlatformGrok:      100,
	}, got.AccountSchedulingThresholds)

	got = svc.parseSettings(map[string]string{
		SettingKeyAccountSchedulingThresholds: `{"openai":91,"grok":77,"gemini":85,"kiro":99}`,
	})
	require.Equal(t, 91, got.AccountSchedulingThresholds[PlatformOpenAI])
	require.Equal(t, 100, got.AccountSchedulingThresholds[PlatformAnthropic])
	require.Equal(t, 77, got.AccountSchedulingThresholds[PlatformGrok])
	require.NotContains(t, got.AccountSchedulingThresholds, PlatformGemini)
	require.NotContains(t, got.AccountSchedulingThresholds, "kiro")
}

func TestBuildSystemSettingsUpdates_PersistsAccountSchedulingThresholds(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(nil)

	updates, err := svc.buildSystemSettingsUpdates(context.Background(), &SystemSettings{
		AccountSchedulingThresholds: map[string]int{
			PlatformOpenAI:    91,
			PlatformAnthropic: 88,
			PlatformGrok:      77,
		},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"openai":91,"anthropic":88,"grok":77}`, updates[SettingKeyAccountSchedulingThresholds])
}

func TestValidateAndNormalizeAccountSchedulingThresholds_FillsMissingPlatforms(t *testing.T) {
	normalized, err := validateAndNormalizeAccountSchedulingThresholds(map[string]int{
		PlatformOpenAI: 91,
	})
	require.NoError(t, err)
	require.Equal(t, 91, normalized[PlatformOpenAI])
	require.Equal(t, 100, normalized[PlatformAnthropic])
	require.Equal(t, 100, normalized[PlatformGrok])
	require.NotContains(t, normalized, PlatformGemini)
	require.NotContains(t, normalized, "kiro")
	require.NotContains(t, normalized, PlatformAntigravity)
}

func TestValidateAndNormalizeAccountSchedulingThresholds_RejectsUnsupportedPlatforms(t *testing.T) {
	_, err := validateAndNormalizeAccountSchedulingThresholds(map[string]int{
		PlatformGemini: 85,
	})
	require.Error(t, err)
}

func TestUpdateSettings_StoresAccountSchedulingThresholds(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(nil)

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		AccountSchedulingThresholds: map[string]int{
			PlatformOpenAI:    92,
			PlatformAnthropic: 89,
			PlatformGrok:      76,
		},
	})
	require.NoError(t, err)

	got := svc.parseSettings(map[string]string{
		SettingKeyAccountSchedulingThresholds: svc.settingRepo.(*mockSettingRepo).data[SettingKeyAccountSchedulingThresholds],
	})
	require.Equal(t, 92, got.AccountSchedulingThresholds[PlatformOpenAI])
	require.Equal(t, 89, got.AccountSchedulingThresholds[PlatformAnthropic])
	require.Equal(t, 76, got.AccountSchedulingThresholds[PlatformGrok])
	require.NotContains(t, got.AccountSchedulingThresholds, "kiro")
}

func TestGetAccountSchedulingThresholds_ReadsStoredValue(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(map[string]string{
		SettingKeyAccountSchedulingThresholds: `{"openai":93,"grok":88,"kiro":87}`,
	})

	got := svc.GetAccountSchedulingThresholds(context.Background())

	require.Equal(t, 93, got[PlatformOpenAI])
	require.Equal(t, 100, got[PlatformAnthropic])
	require.Equal(t, 88, got[PlatformGrok])
	require.NotContains(t, got, "kiro")
}

func TestGetAccountSchedulingThresholds_MissingSettingUsesDefaultsAndNormalCacheTTL(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(nil)
	repo := svc.settingRepo.(*mockSettingRepo)
	repo.getValueErr = ErrSettingNotFound
	// fork 的 NewSettingService 构造时会急切加载 gateway runtime/control 设置
	// （14 次 GetValue）；计数只针对本次热路径读取，与上游纯构造器契约不同。
	repo.getValueCalls = 0

	got := svc.GetAccountSchedulingThresholds(context.Background())
	require.Equal(t, defaultAccountSchedulingThresholds(), got)
	require.Equal(t, 1, repo.getValueCalls)

	repo.data[SettingKeyAccountSchedulingThresholds] = `{"openai":91}`
	got = svc.GetAccountSchedulingThresholds(context.Background())
	require.Equal(t, 100, got[PlatformOpenAI], "missing-setting defaults should remain cached for the normal TTL")
	require.Equal(t, 1, repo.getValueCalls)

	cached, ok := accountSchedulingThresholdsCache.Load().(*cachedAccountSchedulingThresholds)
	require.True(t, ok)
	require.Greater(t, cached.expiresAt, time.Now().Add(accountSchedulingThresholdsCacheTTL-time.Second).UnixNano())
}

func TestUpdateSettings_OmittedAccountSchedulingThresholdsDoesNotCacheDefaults(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(map[string]string{
		SettingKeyAccountSchedulingThresholds: `{"openai":85,"grok":88,"kiro":87}`,
	})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		FrontendURL: "https://example.test",
	})
	require.NoError(t, err)

	got := svc.GetAccountSchedulingThresholds(context.Background())
	require.Equal(t, 85, got[PlatformOpenAI])
	require.Equal(t, 88, got[PlatformGrok])
	require.NotContains(t, got, "kiro")
}

func TestUpdateSettingsOmittingRefreshesRuntimeAfterRequestCancellation(t *testing.T) {
	repo := &cancelAwareSettingRepo{mockSettingRepo: newMockSettingRepo()}
	repo.data[SettingKeyChannelMonitorEnabled] = "true"
	repo.data[SettingKeyChannelMonitorMode] = ChannelMonitorModeV1
	svc := NewSettingService(repo, &config.Config{})
	svc.channelMonitorModeAdmission.desired = ChannelMonitorModeV1
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.UpdateSettingsOmitting(ctx, &SystemSettings{ChannelMonitorEnabled: true, ChannelMonitorMode: ChannelMonitorModeV2}, OmittedSettingKeys{
		SettingKeySiteName: {},
	})
	require.NoError(t, err)

	svc.channelMonitorModeAdmission.mu.Lock()
	desired := svc.channelMonitorModeAdmission.desired
	svc.channelMonitorModeAdmission.mu.Unlock()
	require.Equal(t, ChannelMonitorModeV2, desired)
}

func TestAccountSchedulingThresholds_InvalidStoredValueUsesSameDefaultsInSettingsAndCache(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(map[string]string{
		SettingKeyAccountSchedulingThresholds: `{"openai":0,"grok":88,"kiro":87}`,
	})

	settings := svc.parseSettings(map[string]string{
		SettingKeyAccountSchedulingThresholds: `{"openai":0,"grok":88,"kiro":87}`,
	})
	cached := svc.GetAccountSchedulingThresholds(context.Background())

	require.Equal(t, settings.AccountSchedulingThresholds, cached)
	require.Equal(t, 100, cached[PlatformOpenAI])
	require.Equal(t, 88, cached[PlatformGrok])
	require.NotContains(t, cached, "kiro")
}

func TestGetAccountSchedulingThresholds_NilRepoReturnsDefaults(t *testing.T) {
	svc := &SettingService{}
	got := svc.GetAccountSchedulingThresholds(context.Background())
	require.Equal(t, map[string]int{
		PlatformOpenAI:    100,
		PlatformAnthropic: 100,
		PlatformGrok:      100,
	}, got)
}

// --- review-fix E：阈值热更新时沿 SettingService/update → AccountRepo/调度边界解除暂停 ---

type schedulingThresholdReleaserStub struct {
	calls     int
	platforms []string
	ctxErr    error
}

func (r *schedulingThresholdReleaserStub) ReleaseAccountSchedulingThresholdPauses(ctx context.Context, platforms []string) {
	r.calls++
	r.platforms = append(r.platforms, platforms...)
	r.ctxErr = ctx.Err()
}

func TestUpdateSettings_RaisedThresholdReleasesOnlyDisabledPlatforms(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(map[string]string{
		SettingKeyAccountSchedulingThresholds: `{"openai":90,"grok":80}`,
	})
	releaser := &schedulingThresholdReleaserStub{}
	svc.SetAccountSchedulingThresholdPauseReleaser(releaser)

	// grok 提到 100（+anthropic 缺省 100）：只解除这两个平台的阈值来源暂停，openai 仍为 90 不解除。
	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		AccountSchedulingThresholds: map[string]int{PlatformOpenAI: 90, PlatformGrok: 100},
	})
	require.NoError(t, err)
	require.Equal(t, 1, releaser.calls)
	require.ElementsMatch(t, []string{PlatformAnthropic, PlatformGrok}, releaser.platforms)
}

func TestUpdateSettings_EmptyThresholdsMapReleasesAllPlatforms(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(map[string]string{
		SettingKeyAccountSchedulingThresholds: `{"openai":90,"grok":80}`,
	})
	releaser := &schedulingThresholdReleaserStub{}
	svc.SetAccountSchedulingThresholdPauseReleaser(releaser)

	// 清空阈值（空 map → 全平台默认 100，不再产生暂停）：解除全部平台的阈值来源暂停。
	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		AccountSchedulingThresholds: map[string]int{},
	})
	require.NoError(t, err)
	require.Equal(t, 1, releaser.calls)
	require.ElementsMatch(t, []string{PlatformOpenAI, PlatformAnthropic, PlatformGrok}, releaser.platforms)
}

func TestUpdateSettings_OmittedThresholdsDoNotReleasePauses(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(map[string]string{
		SettingKeyAccountSchedulingThresholds: `{"openai":90}`,
	})
	releaser := &schedulingThresholdReleaserStub{}
	svc.SetAccountSchedulingThresholdPauseReleaser(releaser)

	err := svc.UpdateSettings(context.Background(), &SystemSettings{SiteName: "x"})
	require.NoError(t, err)
	require.Equal(t, 0, releaser.calls, "未携带阈值字段的更新不得触发解除")
}

func TestUpdateSettings_CanceledRequestStillReleasesThresholdPauses(t *testing.T) {
	svc := newSettingServiceForPlatformThresholdTest(map[string]string{
		SettingKeyAccountSchedulingThresholds: `{"openai":90}`,
	})
	releaser := &schedulingThresholdReleaserStub{}
	svc.SetAccountSchedulingThresholdPauseReleaser(releaser)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.UpdateSettings(ctx, &SystemSettings{AccountSchedulingThresholds: map[string]int{}})
	require.NoError(t, err)
	require.Equal(t, 1, releaser.calls)
	require.NoError(t, releaser.ctxErr, "accepted settings update must run bounded cleanup after client cancellation")
}
