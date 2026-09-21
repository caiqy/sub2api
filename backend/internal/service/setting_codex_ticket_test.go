package service

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type codexTicketSettingRepo struct {
	*codexPolicyMigrationRepoStub
	err      error
	writeErr error
}

func (r *codexTicketSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return maps.Clone(r.values), r.err
}

func (r *codexTicketSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string)
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			values[key] = value
		}
	}
	return values, r.err
}

func (r *codexTicketSettingRepo) SetMultiple(_ context.Context, values map[string]string) error {
	if r.writeErr != nil {
		return r.writeErr
	}
	maps.Copy(r.values, values)
	return nil
}

func (r *codexTicketSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return r.codexPolicyMigrationRepoStub.GetValue(ctx, key)
}

func TestCodexTicketEnabledRuntimeSettingOverridesYaml(t *testing.T) {
	repo := &codexTicketSettingRepo{codexPolicyMigrationRepoStub: &codexPolicyMigrationRepoStub{values: map[string]string{}}}
	settings := NewSettingService(repo, &config.Config{})
	svc := ticketTestService(t, config.OpenAICodexTicketConfig{Enabled: false, FailClosed: true}, nil)
	svc.settingService = settings
	account := ticketTestAccount(41)
	svc.storeOpenAICodexTicket(context.Background(), account, &openAICodexTicket{
		AccountID:  41,
		Model:      "gpt-6-astra",
		State:      fakeCodexTicketState(292),
		Length:     292,
		CapturedAt: time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
	})

	h := http.Header{}
	h.Set(openAICodexTurnStateHeader, "client-state")
	require.False(t, svc.openAICodexTicketEnabled(context.Background()))
	require.NoError(t, svc.applyOpenAICodexTicket(context.Background(), account, "gpt-6-astra", h))
	require.Equal(t, "client-state", h.Get(openAICodexTurnStateHeader))

	repo.values[SettingKeyOpenAICodexTicketEnabled] = "true"
	settings.InvalidateOpenAICodexTicketEnabledCache()
	require.True(t, svc.openAICodexTicketEnabled(context.Background()))
	h = http.Header{}
	h.Set(openAICodexTurnStateHeader, "client-state")
	require.NoError(t, svc.applyOpenAICodexTicket(context.Background(), account, "gpt-6-astra", h))
	require.Equal(t, fakeCodexTicketState(292), h.Get(openAICodexTurnStateHeader))

	repo.values[SettingKeyOpenAICodexTicketEnabled] = "false"
	settings.InvalidateOpenAICodexTicketEnabledCache()
	require.False(t, svc.openAICodexTicketEnabled(context.Background()))
	h = http.Header{}
	h.Set(openAICodexTurnStateHeader, "client-state")
	require.NoError(t, svc.applyOpenAICodexTicket(context.Background(), account, "gpt-6-astra", h))
	require.Equal(t, "client-state", h.Get(openAICodexTurnStateHeader))
}

func TestRefreshOpenAICodexTickets_DisabledSkipsHarvest(t *testing.T) {
	upstream := &httpUpstreamRecorder{}
	svc := ticketTestService(t, config.OpenAICodexTicketConfig{
		Enabled:         false,
		HarvestProxyURL: "socks5h://proxy.example.com:1080",
	}, upstream)
	svc.accountRepo = &codexTicketRefreshRepo{accounts: []Account{*ticketTestAccount(41)}}
	svc.refreshOpenAICodexTickets(context.Background())
	require.Empty(t, upstream.requests)
}

func TestCodexTicketProxyRuntimeSettingAndFallback(t *testing.T) {
	repo := &codexTicketSettingRepo{codexPolicyMigrationRepoStub: &codexPolicyMigrationRepoStub{values: map[string]string{}}}
	settings := NewSettingService(repo, &config.Config{})
	svc := ticketTestService(t, config.OpenAICodexTicketConfig{HarvestProxyURL: "http://fallback.example.com:8080"}, nil)
	svc.settingService = settings
	require.Equal(t, "http://fallback.example.com:8080", svc.openAICodexTicketHarvestProxyURL(context.Background()))
	repo.values[SettingKeyOpenAICodexTicketHarvestProxyURL] = "socks5h://user:secret@first.example.com:1080"
	settings.InvalidateOpenAICodexTicketHarvestProxyCache()
	require.Equal(t, repo.values[SettingKeyOpenAICodexTicketHarvestProxyURL], svc.openAICodexTicketHarvestProxyURL(context.Background()))
	repo.values[SettingKeyOpenAICodexTicketHarvestProxyURL] = "http://second.example.com:8080"
	settings.InvalidateOpenAICodexTicketHarvestProxyCache()
	require.Equal(t, "http://second.example.com:8080", svc.openAICodexTicketHarvestProxyURL(context.Background()))
	// Simulate another instance's settings write after the local cache expires.
	repo.values[SettingKeyOpenAICodexTicketHarvestProxyURL] = "https://third.example.com:443"
	settings.openAICodexTicketHarvestProxyCache.Store(&cachedOpenAICodexTicketHarvestProxy{value: "http://second.example.com:8080", expiresAt: time.Now().Add(-time.Second).UnixNano()})
	require.Equal(t, "https://third.example.com:443", svc.openAICodexTicketHarvestProxyURL(context.Background()))
	repo.err = errors.New("database unavailable")
	settings.openAICodexTicketHarvestProxyCache.Store(&cachedOpenAICodexTicketHarvestProxy{value: "https://third.example.com:443", expiresAt: 0})
	require.Equal(t, "https://third.example.com:443", svc.openAICodexTicketHarvestProxyURL(context.Background()))
}

func TestCodexTicketProxyMaskAndValidation(t *testing.T) {
	for _, raw := range []string{"http://user:secret@proxy.example.com:8080", "socks5h://user:secret@proxy.example.com:1080", "https://user:secret@[::1]:443"} {
		require.NoError(t, ValidateOpenAICodexTicketHarvestProxyURL(raw))
		masked := MaskProxyURL(raw)
		require.NotContains(t, masked, "secret")
		require.True(t, IsMaskedProxyURL(masked))
	}
	require.True(t, IsMaskedProxyURL(""))
	require.False(t, IsMaskedProxyURL("http://user:secret***suffix@proxy.example.com:8080"))
	for _, raw := range []string{"user:secret@host:1234", "http://user:secret@", "ftp://user:secret@host:1234", "http://user:secret@host:99999", "http://host:1234/?password=secret", "http://host:1234/#secret", "http://user:secret%zz@host:1234"} {
		err := ValidateOpenAICodexTicketHarvestProxyURL(raw)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "secret")
		require.Empty(t, MaskProxyURL(raw))
	}
}

func TestCodexTicketSettingsRefreshDoesNotMutateSharedConfig(t *testing.T) {
	cfg := &config.Config{}
	svc := NewSettingService(&codexTicketSettingRepo{codexPolicyMigrationRepoStub: &codexPolicyMigrationRepoStub{values: map[string]string{SettingKeyOpenAICodexTicketEnabled: "true"}}}, cfg)
	require.True(t, svc.GetOpenAICodexTicketEnabled(context.Background(), false))
	require.False(t, cfg.Gateway.OpenAICodexTicket.Enabled, "runtime settings must not write the shared immutable startup configuration")
}

func TestCodexTicketRuntimeSettingsHotReloadAndFallback(t *testing.T) {
	repo := &codexTicketSettingRepo{codexPolicyMigrationRepoStub: &codexPolicyMigrationRepoStub{values: map[string]string{}}}
	settings := NewSettingService(repo, &config.Config{})
	fallback := OpenAICodexTicketRuntimeSettings{
		TargetLength:         292,
		TTLSeconds:           3600,
		RefreshBeforeSeconds: 600,
		ProbeIntervalSeconds: 6,
		MaxConcurrentProbes:  8,
	}
	// 后台未覆盖时逐项回退 yaml/env 基线。
	require.Equal(t, fallback, settings.GetOpenAICodexTicketRuntimeSettings(context.Background(), fallback))

	repo.values[SettingKeyOpenAICodexTicketTargetLength] = "512"
	repo.values[SettingKeyOpenAICodexTicketTTLSeconds] = "1800"
	repo.values[SettingKeyOpenAICodexTicketRefreshBefore] = "300"
	repo.values[SettingKeyOpenAICodexTicketProbeInterval] = "15"
	repo.values[SettingKeyOpenAICodexTicketMaxConcurrent] = "3"
	settings.InvalidateOpenAICodexTicketRuntimeCache()
	got := settings.GetOpenAICodexTicketRuntimeSettings(context.Background(), fallback)
	require.Equal(t, 512, got.TargetLength)
	require.Equal(t, 1800, got.TTLSeconds)
	require.Equal(t, 300, got.RefreshBeforeSeconds)
	require.Equal(t, 15, got.ProbeIntervalSeconds)
	require.Equal(t, 3, got.MaxConcurrentProbes)

	// 非法值不生效：回退到 yaml/env 基线，而不是把节流清零。
	repo.values[SettingKeyOpenAICodexTicketMaxConcurrent] = "0"
	settings.InvalidateOpenAICodexTicketRuntimeCache()
	require.Equal(t, fallback.MaxConcurrentProbes, settings.GetOpenAICodexTicketRuntimeSettings(context.Background(), fallback).MaxConcurrentProbes)
}

func TestCodexTicketRuntimeSettingsExpiredCacheSurvivesStorageFailure(t *testing.T) {
	repo := &codexTicketSettingRepo{codexPolicyMigrationRepoStub: &codexPolicyMigrationRepoStub{values: map[string]string{SettingKeyOpenAICodexTicketTTLSeconds: "1800"}}}
	settings := NewSettingService(repo, &config.Config{})
	ctx := context.Background()
	known := settings.GetOpenAICodexTicketRuntimeSettings(ctx, OpenAICodexTicketRuntimeSettings{})
	require.Equal(t, 1800, known.TTLSeconds)
	settings.openAICodexTicketRuntimeCache.Store(&cachedOpenAICodexTicketRuntime{value: known, expiresAt: time.Now().Add(-time.Second).UnixNano()})
	repo.err = errors.New("storage unavailable")
	require.Equal(t, known, settings.GetOpenAICodexTicketRuntimeSettings(ctx, OpenAICodexTicketRuntimeSettings{}))
	settings.InvalidateOpenAICodexTicketRuntimeCache()
	require.Equal(t, known, settings.GetOpenAICodexTicketRuntimeSettings(ctx, OpenAICodexTicketRuntimeSettings{}))
	repo.err = nil
	repo.values[SettingKeyOpenAICodexTicketTTLSeconds] = "2400"
	require.Equal(t, 2400, settings.GetOpenAICodexTicketRuntimeSettings(ctx, OpenAICodexTicketRuntimeSettings{}).TTLSeconds)
}

func TestCodexTicketRuntimeSettingsZeroRefreshReachesGateway(t *testing.T) {
	repo := &codexTicketSettingRepo{codexPolicyMigrationRepoStub: &codexPolicyMigrationRepoStub{values: map[string]string{}}}
	settings := NewSettingService(repo, &config.Config{})
	svc := &OpenAIGatewayService{cfg: &config.Config{}, settingService: settings}
	ctx := context.Background()
	got := svc.openAICodexTicketConfig(ctx)
	require.Equal(t, 292, got.TargetLength)
	require.Equal(t, 3600, got.TTLSeconds)
	require.Equal(t, 600, got.RefreshBeforeSeconds)
	require.Equal(t, 6, got.HarvestProbeIntervalSeconds)
	require.Equal(t, 8, got.MaxConcurrentProbes)
	repo.values[SettingKeyOpenAICodexTicketRefreshBefore] = "0"
	settings.InvalidateOpenAICodexTicketRuntimeCache()
	require.Zero(t, svc.openAICodexTicketConfig(ctx).RefreshBeforeSeconds)
}

func TestCodexTicketRuntimeSettingsPartialServiceUpdates(t *testing.T) {
	repo := &codexTicketSettingRepo{codexPolicyMigrationRepoStub: &codexPolicyMigrationRepoStub{values: map[string]string{}}}
	settings := NewSettingService(repo, &config.Config{})
	ctx := context.Background()
	current, err := settings.GetAllSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, 292, current.OpenAICodexTicketTargetLength)
	require.Equal(t, 3600, current.OpenAICodexTicketTTLSeconds)
	require.Equal(t, 600, current.OpenAICodexTicketRefreshBeforeSeconds)
	require.Equal(t, 6, current.OpenAICodexTicketProbeIntervalSeconds)
	require.Equal(t, 8, current.OpenAICodexTicketMaxConcurrentProbes)
	for _, update := range []*SystemSettings{
		{OpenAICodexTicketTTLSeconds: 600},
		{OpenAICodexTicketRefreshBeforeSeconds: 3600},
	} {
		require.Error(t, settings.UpdateSettings(ctx, update))
		require.Empty(t, repo.values)
	}
	require.NoError(t, settings.UpdateSettings(ctx, &SystemSettings{OpenAICodexTicketTTLSeconds: 1800}))
	require.NoError(t, settings.UpdateSettings(ctx, &SystemSettings{OpenAICodexTicketRefreshBeforeSeconds: 900}))
	before := maps.Clone(repo.values)
	known := settings.GetOpenAICodexTicketRuntimeSettings(ctx, OpenAICodexTicketRuntimeSettings{})
	require.Equal(t, 1800, known.TTLSeconds)
	require.Equal(t, 900, known.RefreshBeforeSeconds)
	for _, update := range []*SystemSettings{
		{OpenAICodexTicketTTLSeconds: 900},
		{OpenAICodexTicketRefreshBeforeSeconds: 1800},
	} {
		require.Error(t, settings.UpdateSettings(ctx, update))
		require.Equal(t, before, repo.values)
	}
	repo.writeErr = errors.New("write failed")
	require.ErrorIs(t, settings.UpdateSettings(ctx, &SystemSettings{OpenAICodexTicketTTLSeconds: 2400}), repo.writeErr)
	require.Equal(t, before, repo.values)
	require.Equal(t, known, settings.GetOpenAICodexTicketRuntimeSettings(ctx, OpenAICodexTicketRuntimeSettings{}))
	repo.writeErr = nil
	require.NoError(t, settings.UpdateSettings(ctx, &SystemSettings{OpenAICodexTicketRefreshBeforeSecondsSet: true}))
	require.Equal(t, "0", repo.values[SettingKeyOpenAICodexTicketRefreshBefore])
	require.NoError(t, settings.UpdateSettings(ctx, &SystemSettings{SiteName: "unrelated"}))
	current, err = settings.GetAllSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, 1800, current.OpenAICodexTicketTTLSeconds)
	require.Zero(t, current.OpenAICodexTicketRefreshBeforeSeconds)
	require.Zero(t, settings.GetOpenAICodexTicketRuntimeSettings(ctx, OpenAICodexTicketRuntimeSettings{}).RefreshBeforeSeconds)
}

func TestCodexTicketInvalidatedCachesPreserveLastKnownValues(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(strconv.FormatBool(enabled), func(t *testing.T) {
			repo := &codexTicketSettingRepo{codexPolicyMigrationRepoStub: &codexPolicyMigrationRepoStub{values: map[string]string{
				SettingKeyOpenAICodexTicketEnabled:         strconv.FormatBool(enabled),
				SettingKeyOpenAICodexTicketHarvestProxyURL: "http://stored.example.com:8080",
			}}}
			settings := NewSettingService(repo, &config.Config{})
			base := config.OpenAICodexTicketConfig{Enabled: !enabled, HarvestProxyURL: "http://yaml.example.com:8080"}
			ctx := context.Background()
			known := settings.ApplyOpenAICodexTicketOverrides(ctx, base)
			require.Equal(t, enabled, known.Enabled)
			require.Equal(t, "http://stored.example.com:8080", known.HarvestProxyURL)
			settings.InvalidateOpenAICodexTicketEnabledCache()
			settings.InvalidateOpenAICodexTicketHarvestProxyCache()
			repo.err = errors.New("temporary DB failure")
			got := settings.ApplyOpenAICodexTicketOverrides(ctx, base)
			assert.Equal(t, known.Enabled, got.Enabled)
			assert.Equal(t, known.HarvestProxyURL, got.HarvestProxyURL)
			repo.err = nil
			repo.values[SettingKeyOpenAICodexTicketEnabled] = strconv.FormatBool(!enabled)
			repo.values[SettingKeyOpenAICodexTicketHarvestProxyURL] = "http://recovered.example.com:8080"
			settings.InvalidateOpenAICodexTicketEnabledCache()
			settings.InvalidateOpenAICodexTicketHarvestProxyCache()
			got = settings.ApplyOpenAICodexTicketOverrides(ctx, base)
			require.Equal(t, !enabled, got.Enabled)
			require.Equal(t, "http://recovered.example.com:8080", got.HarvestProxyURL)
		})
	}
}

func TestCodexTicketConfiguredZeroRefresh(t *testing.T) {
	for _, ttl := range []int{0, 300} {
		t.Run(strconv.Itoa(ttl), func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Gateway.OpenAICodexTicket.TTLSeconds = ttl
			want := 0
			if ttl == 0 {
				want = 600
			}
			repo := &codexTicketSettingRepo{codexPolicyMigrationRepoStub: &codexPolicyMigrationRepoStub{values: map[string]string{}}}
			settings := NewSettingService(repo, cfg)
			ctx := context.Background()
			view, err := settings.GetAllSettings(ctx)
			require.NoError(t, err)
			assert.Equal(t, want, view.OpenAICodexTicketRefreshBeforeSeconds, "settings view")
			for _, settingService := range []*SettingService{nil, settings} {
				svc := &OpenAIGatewayService{cfg: cfg, settingService: settingService}
				assert.Equal(t, want, svc.openAICodexTicketConfig(ctx).RefreshBeforeSeconds, "gateway")
			}
		})
	}
}
