package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRemovedLayeredConfigurationIsIgnored(t *testing.T) {
	v := viper.New()
	v.Set("gateway.openai_ws.scheduler_mode", "layered")
	v.Set("gateway.openai_ws.scheduler_layered.probe_interval_seconds", 1)
	v.Set("gateway.openai_ws.scheduler_layered.error_penalty_threshold", 0.01)
	v.Set("gateway.openai_ws.lb_top_k", 9)
	v.Set("gateway.sticky.openai.enabled", true)
	var cfg Config
	require.NoError(t, v.Unmarshal(&cfg))
	require.Equal(t, GatewayOpenAIWSConfig{LBTopK: 9}, cfg.Gateway.OpenAIWS)
	require.True(t, cfg.GatewayControlRuntime().StickyOpenAIEnabled)
}
