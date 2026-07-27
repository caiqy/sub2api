package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestExampleConfigGatewayBodyLimits(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "deploy", "config.example.yaml"))
	require.NoError(t, err)

	var example struct {
		Gateway struct {
			TextMaxBodySize              int64 `yaml:"text_max_body_size"`
			UpstreamResponseReadMaxBytes int64 `yaml:"upstream_response_read_max_bytes"`
		} `yaml:"gateway"`
	}
	require.NoError(t, yaml.Unmarshal(data, &example))
	require.Equal(t, int64(32*1024*1024), example.Gateway.TextMaxBodySize)
	require.Equal(t, int64(128*1024*1024), example.Gateway.UpstreamResponseReadMaxBytes)
}
