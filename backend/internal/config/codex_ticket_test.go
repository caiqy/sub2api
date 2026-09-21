package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadCodexTicketRefreshDefaultsAndExplicitZero(t *testing.T) {
	for _, source := range []string{"defaults", "yaml", "env"} {
		t.Run(source, func(t *testing.T) {
			resetViperWithJWTSecret(t)
			t.Setenv("GATEWAY_OPENAI_CODEX_TICKET_TTL_SECONDS", "")
			t.Setenv("GATEWAY_OPENAI_CODEX_TICKET_REFRESH_BEFORE_SECONDS", "")
			path := filepath.Join(t.TempDir(), "config.yaml")
			contents := "{}\n"
			if source == "yaml" {
				contents = "gateway:\n  openai_codex_ticket:\n    ttl_seconds: 300\n    refresh_before_seconds: 0\n"
			}
			require.NoError(t, os.WriteFile(path, []byte(contents), 0600))
			t.Setenv("CONFIG_FILE", path)
			if source == "env" {
				t.Setenv("GATEWAY_OPENAI_CODEX_TICKET_TTL_SECONDS", "300")
				t.Setenv("GATEWAY_OPENAI_CODEX_TICKET_REFRESH_BEFORE_SECONDS", "0")
			}
			cfg, err := Load()
			require.NoError(t, err)
			if source == "defaults" {
				require.Equal(t, 3600, cfg.Gateway.OpenAICodexTicket.TTLSeconds)
				require.Equal(t, 600, cfg.Gateway.OpenAICodexTicket.RefreshBeforeSeconds)
			} else {
				require.Equal(t, 300, cfg.Gateway.OpenAICodexTicket.TTLSeconds)
				require.Zero(t, cfg.Gateway.OpenAICodexTicket.RefreshBeforeSeconds)
			}
		})
	}
}
