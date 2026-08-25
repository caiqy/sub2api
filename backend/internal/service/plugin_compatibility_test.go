package service

import (
	"testing"

	pluginv1 "github.com/Wei-Shaw/sub2api/pkg/pluginapi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluatePluginCompatibility(t *testing.T) {
	manifest := testPluginManifest(nil)
	host := PluginHostInfo{Version: "0.1.179", BuildType: "release"}

	result := EvaluatePluginCompatibility(manifest, host)
	require.True(t, result.Compatible)
	assert.True(t, result.Tested)
	assert.Equal(t, "compatible", result.Status)

	manifest.Requires.TestedSub2APIVersions = []string{"0.1.178"}
	result = EvaluatePluginCompatibility(manifest, host)
	require.True(t, result.Compatible)
	assert.False(t, result.Tested)
	assert.Equal(t, "untested", result.Status)

	manifest.Requires.Sub2API = ">=0.2.0 <0.3.0"
	result = EvaluatePluginCompatibility(manifest, host)
	assert.False(t, result.Compatible)
	assert.Equal(t, "incompatible", result.Status)
}

func TestEvaluatePluginCompatibilityRejectsProtocolMismatch(t *testing.T) {
	manifest := testPluginManifest(nil)
	manifest.Requires.PluginProtocol = pluginv1.ProtocolVersion + 1

	result := EvaluatePluginCompatibility(manifest, PluginHostInfo{Version: "0.1.179"})

	assert.False(t, result.Compatible)
	assert.Equal(t, "incompatible", result.Status)
}

func TestEvaluatePluginCompatibilityAcceptsForkReleaseVersion(t *testing.T) {
	for _, tt := range []struct {
		host   string
		tested string
	}{{"0.1.180.1", "0.1.180"}, {"0.1.181.1", "0.1.181"}} {
		t.Run(tt.host, func(t *testing.T) {
			manifest := testPluginManifest(nil)
			manifest.Requires.Sub2API = ">=0.1.180 <0.2.0"
			manifest.Requires.TestedSub2APIVersions = []string{tt.tested}

			result := EvaluatePluginCompatibility(manifest, PluginHostInfo{Version: tt.host, BuildType: "release"})

			require.True(t, result.Compatible)
			assert.True(t, result.Tested)
			assert.Equal(t, "compatible", result.Status)
		})
	}
}

func TestPluginCompatibilityRejectsForkVersionSyntaxInManifestDeclarations(t *testing.T) {
	manifest := testPluginManifest(nil)
	manifest.Version = "1.2.3.4"
	require.Error(t, manifest.Validate())

	manifest = testPluginManifest(nil)
	manifest.Requires.Sub2API = ">=0.1.180.1 <0.2.0"
	result := EvaluatePluginCompatibility(manifest, PluginHostInfo{Version: "0.1.180.1"})
	assert.False(t, result.Compatible)

	manifest.Requires.Sub2API = ">=0.1.180 <0.2.0"
	manifest.Requires.TestedSub2APIVersions = []string{"0.1.180.1"}
	result = EvaluatePluginCompatibility(manifest, PluginHostInfo{Version: "0.1.180.1"})
	require.True(t, result.Compatible)
	assert.False(t, result.Tested)
	assert.Equal(t, "untested", result.Status)
}

func TestMatchesSemverRange(t *testing.T) {
	assert.True(t, matchesSemverRange("0.1.179", ">=0.1.170, <0.2.0"))
	assert.True(t, matchesSemverRange("v1.2.3", "=1.2.3"))
	assert.True(t, matchesSemverRange("1.2.3-alpha.1", "=1.2.3-alpha.1"))
	assert.True(t, matchesSemverRange("1.2.3+fork.1", "=1.2.3+fork.1"))
	assert.False(t, matchesSemverRange("0.1.169", ">=0.1.170 <0.2.0"))
	assert.False(t, matchesSemverRange("dev", ">=0.1.0"))
	assert.False(t, matchesSemverRange("0.1.179", "^0.1.0"))
}
