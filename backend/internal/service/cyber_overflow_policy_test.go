package service

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCyberOverflowPolicy(t *testing.T) {
	for _, tc := range []struct {
		name                                        string
		count                                       int
		active, exempt, explicit, history, disabled bool
		want                                        string
		batch                                       int
	}{
		{name: "default blocks overflow", count: 257, active: true, want: cyberSessionTranscriptLookupOverflowBlockKey},
		{name: "inactive scope", count: 257},
		{name: "inactive scope at boundary", count: 256},
		{name: "global disabled", count: 300, active: true, disabled: true},
		{name: "256 boundary", count: 256, active: true, batch: 256},
		{name: "exempt overflow", count: 300, active: true, exempt: true, batch: 256},
		{name: "exempt retained history", count: 300, active: true, exempt: true, history: true, want: "history", batch: 256},
		{name: "exempt explicit", count: 300, active: true, exempt: true, explicit: true, want: "explicit", batch: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			messages := make([]map[string]string, tc.count)
			for i := range messages {
				messages[i] = map[string]string{"role": "user", "content": "message-" + strconv.Itoa(i)}
			}
			body, err := json.Marshal(map[string]any{"messages": messages})
			require.NoError(t, err)
			headers := map[string]string{}
			if tc.explicit {
				headers["session_id"] = "blocked-session"
			}
			c, _ := newCyberBlockTestCtx(headers, string(body))
			cache := &comboCacheAndStore{store: fakeCyberBlockStore{scopes: map[string]bool{CyberSessionScopeKey(9, "203.0.113.20", "client"): tc.active}, blocked: map[string]bool{}}}
			want := tc.want
			if tc.explicit {
				want = CyberSessionExplicitBlockKey(9, c, body)
				cache.store.blocked[want] = true
			}
			if tc.history {
				keys := CyberSessionTranscriptLookupKeys(9, body)
				want = keys[0]
				cache.store.blocked[want] = true
			}
			svc := &OpenAIGatewayService{cache: cache, settingService: &SettingService{settingRepo: &fakeSettingRepo{vals: map[string]string{SettingKeyCyberSessionBlockEnabled: strconv.FormatBool(!tc.disabled)}}}}
			require.Equal(t, want, svc.FindCyberSessionBlockedForRequest(context.Background(), 9, c, body, "203.0.113.20", "client", tc.exempt))
			if tc.batch == 0 {
				require.Empty(t, cache.store.batchSizes)
			} else {
				require.Equal(t, []int{tc.batch}, cache.store.batchSizes)
			}
		})
	}
}

func TestAPIKeyAuthSnapshotCyberOverflowPolicy(t *testing.T) {
	svc := &APIKeyService{}
	for _, exempt := range []bool{false, true} {
		snapshot := svc.snapshotFromAPIKey(context.Background(), &APIKey{ID: 1, UserID: 2, User: &User{ID: 2, CyberOverflowExempt: exempt}})
		require.Equal(t, exempt, snapshot.User.CyberOverflowExempt)
		data, err := json.Marshal(snapshot)
		require.NoError(t, err)
		var cached APIKeyAuthSnapshot
		require.NoError(t, json.Unmarshal(data, &cached))
		require.Equal(t, exempt, svc.snapshotToAPIKey("key", &cached).User.CyberOverflowExempt)
	}
	key, ok, err := svc.applyAuthCacheEntry("legacy", &APIKeyAuthCacheEntry{Snapshot: &APIKeyAuthSnapshot{Version: 28}})
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, key)
}
