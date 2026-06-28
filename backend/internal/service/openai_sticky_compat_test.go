package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestGetStickySessionAccountID_FallbackToLegacyKey(t *testing.T) {
	beforeFallbackTotal, beforeFallbackHit, _ := openAIStickyCompatStats()

	cache := &stubGatewayCache{
		sessionBindings: map[string]int64{
			"openai:legacy-hash": 42,
		},
	}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				Sticky: config.GatewayStickyConfig{OpenAI: config.GatewayStickyPlatformConfig{Enabled: true}},
				OpenAIWS: config.GatewayOpenAIWSConfig{
					SessionHashReadOldFallback: true,
				},
			},
		},
	}

	ctx := withOpenAILegacySessionHash(context.Background(), "legacy-hash")
	accountID, err := svc.getStickySessionAccountID(ctx, nil, "new-hash")
	require.NoError(t, err)
	require.Equal(t, int64(42), accountID)

	afterFallbackTotal, afterFallbackHit, _ := openAIStickyCompatStats()
	require.Equal(t, beforeFallbackTotal+1, afterFallbackTotal)
	require.Equal(t, beforeFallbackHit+1, afterFallbackHit)
}

func TestSetStickySessionAccountID_DualWriteOldEnabled(t *testing.T) {
	_, _, beforeDualWriteTotal := openAIStickyCompatStats()

	cache := &stubGatewayCache{sessionBindings: map[string]int64{}}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				Sticky: config.GatewayStickyConfig{OpenAI: config.GatewayStickyPlatformConfig{Enabled: true}},
				OpenAIWS: config.GatewayOpenAIWSConfig{
					SessionHashDualWriteOld: true,
				},
			},
		},
	}

	ctx := withOpenAILegacySessionHash(context.Background(), "legacy-hash")
	err := svc.setStickySessionAccountID(ctx, nil, "new-hash", 9, openaiStickySessionTTL)
	require.NoError(t, err)
	require.Equal(t, int64(9), cache.sessionBindings["openai:new-hash"])
	require.Equal(t, int64(9), cache.sessionBindings["openai:legacy-hash"])

	_, _, afterDualWriteTotal := openAIStickyCompatStats()
	require.Equal(t, beforeDualWriteTotal+1, afterDualWriteTotal)
}

func TestSetStickySessionAccountID_DualWriteOldDisabled(t *testing.T) {
	cache := &stubGatewayCache{sessionBindings: map[string]int64{}}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				Sticky: config.GatewayStickyConfig{OpenAI: config.GatewayStickyPlatformConfig{Enabled: true}},
				OpenAIWS: config.GatewayOpenAIWSConfig{
					SessionHashDualWriteOld: false,
				},
			},
		},
	}

	ctx := withOpenAILegacySessionHash(context.Background(), "legacy-hash")
	err := svc.setStickySessionAccountID(ctx, nil, "new-hash", 9, openaiStickySessionTTL)
	require.NoError(t, err)
	require.Equal(t, int64(9), cache.sessionBindings["openai:new-hash"])
	_, exists := cache.sessionBindings["openai:legacy-hash"]
	require.False(t, exists)
}

func TestSnapshotOpenAICompatibilityFallbackMetrics(t *testing.T) {
	before := SnapshotOpenAICompatibilityFallbackMetrics()

	ctx := context.WithValue(context.Background(), ctxkey.ThinkingEnabled, true)
	_, _ = ThinkingEnabledFromContext(ctx)

	after := SnapshotOpenAICompatibilityFallbackMetrics()
	require.GreaterOrEqual(t, after.MetadataLegacyFallbackTotal, before.MetadataLegacyFallbackTotal+1)
	require.GreaterOrEqual(t, after.MetadataLegacyFallbackThinkingEnabledTotal, before.MetadataLegacyFallbackThinkingEnabledTotal+1)
}

func TestOpenAIStickyHelpers_DisabledDoNotTouchCacheOrCompatCounters(t *testing.T) {
	beforeFallbackTotal, beforeFallbackHit, beforeDualWriteTotal := openAIStickyCompatStats()
	cache := &stubGatewayCache{sessionBindings: map[string]int64{"openai:new-hash": 11, "openai:legacy-hash": 22}}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg:   &config.Config{Gateway: config.GatewayConfig{Sticky: config.GatewayStickyConfig{}}},
	}
	ctx := withOpenAILegacySessionHash(context.Background(), "legacy-hash")

	accountID, err := svc.getStickySessionAccountID(ctx, nil, "new-hash")
	require.NoError(t, err)
	require.Zero(t, accountID)
	require.NoError(t, svc.setStickySessionAccountID(ctx, nil, "new-hash", 9, openaiStickySessionTTL))
	require.NoError(t, svc.refreshStickySessionTTL(ctx, nil, "new-hash", openaiStickySessionTTL))
	require.NoError(t, svc.deleteStickySessionAccountID(ctx, nil, "new-hash"))

	afterFallbackTotal, afterFallbackHit, afterDualWriteTotal := openAIStickyCompatStats()
	require.Equal(t, beforeFallbackTotal, afterFallbackTotal)
	require.Equal(t, beforeFallbackHit, afterFallbackHit)
	require.Equal(t, beforeDualWriteTotal, afterDualWriteTotal)
	require.Zero(t, cache.getCalls["openai:new-hash"])
	require.Zero(t, cache.getCalls["openai:legacy-hash"])
	require.Zero(t, cache.setCalls["openai:new-hash"])
	require.Zero(t, cache.setCalls["openai:legacy-hash"])
	require.Zero(t, cache.refreshCalls["openai:new-hash"])
	require.Zero(t, cache.refreshCalls["openai:legacy-hash"])
	require.Nil(t, cache.deletedSessions)
	require.Equal(t, int64(11), cache.sessionBindings["openai:new-hash"])
	require.Equal(t, int64(22), cache.sessionBindings["openai:legacy-hash"])
}
