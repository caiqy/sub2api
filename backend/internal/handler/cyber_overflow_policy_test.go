package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestCyberOverflowPolicySharedHTTPAndWebSocket(t *testing.T) {
	cache := testutil.NewRedisGatewayCache(t)
	settings := service.NewSettingService(&contentModerationHandlerSettingRepo{values: map[string]string{service.SettingKeyCyberSessionBlockEnabled: "true"}}, nil)
	gateway := service.NewOpenAIGatewayService(nil, nil, nil, nil, nil, nil, cache, &config.Config{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, settings, nil)
	t.Cleanup(gateway.StopOpenAICodexTicketHarvester)
	h := &OpenAIGatewayHandler{gatewayService: gateway}
	store := cache.(service.CyberSessionBlockStore)
	messages := make([]map[string]string, 300)
	for i := range messages {
		messages[i] = map[string]string{"role": "user", "content": "message-" + strconv.Itoa(i)}
	}
	body, err := json.Marshal(map[string]any{"messages": messages})
	require.NoError(t, err)
	key := &service.APIKey{ID: 7, User: &service.User{ID: 3}}
	c := newTestGinContext()
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	c.Request.Header.Set("User-Agent", "client")
	scope := cyberSessionScopeKey(key.ID, c)
	require.NoError(t, store.SetCyberSessionBlocked(context.Background(), scope, []string{"unmatched"}, time.Minute))
	for _, exempt := range []bool{false, true} {
		key.User.CyberOverflowExempt = exempt
		for _, model := range []string{"gpt-5.1", "deepseek-chat"} {
			for _, format := range []cyberSessionBlockFormat{cyberBlockFormatResponses, cyberBlockFormatChat, cyberBlockFormatAnthropic} {
				require.Equal(t, !exempt, h.rejectIfCyberSessionBlocked(c, key, body, model, format))
			}
		}
		matched := h.findBlockedCyberSessionForAPIKey(c, key, body)
		if exempt {
			require.Empty(t, matched)
		} else {
			require.Equal(t, "transcript_lookup_limit_exceeded", matched)
		}
	}
	keys := service.CyberSessionTranscriptLookupKeys(key.ID, body)
	require.NoError(t, store.SetCyberSessionBlocked(context.Background(), scope, []string{keys[0]}, time.Minute))
	require.Equal(t, keys[0], h.findBlockedCyberSessionForAPIKey(c, key, body), "exempt users still match retained history")
}
