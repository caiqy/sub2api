package admin

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSettingsCodexTicketRuntimeWriteReadAndHotReload(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{})
	ctx := context.Background()
	defaults := service.OpenAICodexTicketRuntimeSettings{TargetLength: 292, TTLSeconds: 3600, RefreshBeforeSeconds: 600, ProbeIntervalSeconds: 6, MaxConcurrentProbes: 8}
	require.Equal(t, defaults, h.settingService.GetOpenAICodexTicketRuntimeSettings(ctx, service.OpenAICodexTicketRuntimeSettings{}))
	for _, values := range [][]int{{512, 1800, 300, 15, 3}, {32, 60, 0, 1, 1}, {4096, 86400, 43200, 3600, 256}} {
		keys := []string{service.SettingKeyOpenAICodexTicketTargetLength, service.SettingKeyOpenAICodexTicketTTLSeconds, service.SettingKeyOpenAICodexTicketRefreshBefore, service.SettingKeyOpenAICodexTicketProbeInterval, service.SettingKeyOpenAICodexTicketMaxConcurrent}
		body := map[string]any{}
		for i, key := range keys {
			body[key] = values[i]
		}
		rec := doUpdateSettings(t, h, body, nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		get := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(get)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
		h.GetSettings(c)
		require.Equal(t, http.StatusOK, get.Code)
		var response struct {
			Data map[string]any `json:"data"`
		}
		require.NoError(t, json.Unmarshal(get.Body.Bytes(), &response))
		for i, key := range keys {
			require.Equal(t, strconv.Itoa(values[i]), repo.values[key], key)
			require.Equal(t, float64(values[i]), response.Data[key], key)
		}
		want := service.OpenAICodexTicketRuntimeSettings{TargetLength: values[0], TTLSeconds: values[1], RefreshBeforeSeconds: values[2], ProbeIntervalSeconds: values[3], MaxConcurrentProbes: values[4]}
		require.Equal(t, want, h.settingService.GetOpenAICodexTicketRuntimeSettings(ctx, defaults))
		rec = doUpdateSettings(t, h, map[string]any{"site_name": "keep ticket settings"}, nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, want, h.settingService.GetOpenAICodexTicketRuntimeSettings(ctx, defaults))
		for i, key := range keys {
			require.Equal(t, strconv.Itoa(values[i]), repo.values[key], key)
		}
	}
}

func TestSettingsCodexTicketRuntimeRejectsInvalidAtomically(t *testing.T) {
	for key, invalid := range map[string][]int{
		service.SettingKeyOpenAICodexTicketTargetLength:  {0, -1, 31, 4097},
		service.SettingKeyOpenAICodexTicketTTLSeconds:    {0, -1, 59, 86401, 600},
		service.SettingKeyOpenAICodexTicketRefreshBefore: {-1, 43201, 3600},
		service.SettingKeyOpenAICodexTicketProbeInterval: {0, -1, 3601},
		service.SettingKeyOpenAICodexTicketMaxConcurrent: {0, -1, 257},
	} {
		for _, value := range invalid {
			t.Run(key+"/"+strconv.Itoa(value), func(t *testing.T) {
				for _, stored := range []map[string]string{{}, {
					service.SettingKeyOpenAICodexTicketTTLSeconds:    "3600",
					service.SettingKeyOpenAICodexTicketRefreshBefore: "600",
				}} {
					h, repo := newStepUpSwitchTestHandler(t, stored)
					before := maps.Clone(repo.values)
					runtime := h.settingService.GetOpenAICodexTicketRuntimeSettings(context.Background(), service.OpenAICodexTicketRuntimeSettings{})
					rec := doUpdateSettings(t, h, map[string]any{key: value, "site_name": "must not persist"}, nil)
					require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
					require.Equal(t, before, repo.values)
					require.Empty(t, repo.history)
					require.Equal(t, runtime, h.settingService.GetOpenAICodexTicketRuntimeSettings(context.Background(), service.OpenAICodexTicketRuntimeSettings{}))
				}
			})
		}
	}
}

func TestSettingsCodexTicketProxyWriteReadAndHotReload(t *testing.T) {
	key := service.SettingKeyOpenAICodexTicketHarvestProxyURL
	oldProxy := "http://user:old-secret@old.example.com:8080"
	newProxy := "socks5h://user:new-secret@new.example.com:1080"
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{key: oldProxy})
	require.Equal(t, oldProxy, h.settingService.GetOpenAICodexTicketHarvestProxyURL(context.Background()))
	rec := doUpdateSettings(t, h, map[string]any{key: newProxy}, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, newProxy, repo.values[key])
	require.Equal(t, newProxy, h.settingService.GetOpenAICodexTicketHarvestProxyURL(context.Background()))
	require.NotContains(t, rec.Body.String(), "new-secret")
	require.Contains(t, rec.Body.String(), `"openai_codex_ticket_harvest_proxy_configured":true`)
	// Omission, empty input and the masked GET value all preserve the real secret.
	for _, body := range []map[string]any{{"site_name": "updated"}, {key: ""}, {key: service.MaskProxyURL(newProxy)}} {
		rec = doUpdateSettings(t, h, body, nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, newProxy, repo.values[key])
	}
	get := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(get)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	h.GetSettings(c)
	require.Equal(t, http.StatusOK, get.Code)
	require.NotContains(t, get.Body.String(), "new-secret")
	require.Contains(t, get.Body.String(), "new.example.com")
}

func TestSettingsCodexTicketRejectInvalidProxyWithoutLeakingPassword(t *testing.T) {
	key := service.SettingKeyOpenAICodexTicketHarvestProxyURL
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{key: "http://previous.example.com:8080"})
	rec := doUpdateSettings(t, h, map[string]any{key: "ftp://user:invalid-secret@proxy.example.com:21"}, nil)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.NotContains(t, rec.Body.String(), "invalid-secret")
	require.Equal(t, "http://previous.example.com:8080", repo.values[key])
}
