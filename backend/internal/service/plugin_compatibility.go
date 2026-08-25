package service

import (
	"fmt"
	"strings"

	pluginv1 "github.com/Wei-Shaw/sub2api/pkg/pluginapi/v1"
	"golang.org/x/mod/semver"
)

type PluginHostInfo struct {
	Version   string
	BuildType string
}

func EvaluatePluginCompatibility(manifest PluginManifest, host PluginHostInfo) PluginCompatibility {
	hostVersion := normalizePluginHostSemver(host.Version)
	result := PluginCompatibility{
		CurrentSub2API:     host.Version,
		RequiredSub2API:    manifest.Requires.Sub2API,
		RecommendedSub2API: manifest.Requires.RecommendedSub2APIVersion,
		PluginProtocol:     manifest.Requires.PluginProtocol,
		TransportAPI:       manifest.Requires.TransportAPI,
		UIBridge:           manifest.Requires.UIBridge,
	}
	if manifest.Requires.PluginProtocol != pluginv1.ProtocolVersion ||
		manifest.Requires.TransportAPI != pluginv1.TransportAPIVersion ||
		manifest.Requires.UIBridge != pluginv1.UIBridgeVersion {
		result.Status = "incompatible"
		result.Message = "插件协议版本与当前 Sub2API 不兼容"
		return result
	}
	if !matchesSemverRange(hostVersion, manifest.Requires.Sub2API) {
		result.Status = "incompatible"
		result.Message = fmt.Sprintf("当前 Sub2API %s 不满足插件要求 %s", host.Version, manifest.Requires.Sub2API)
		return result
	}
	result.Compatible = true
	for _, tested := range manifest.Requires.TestedSub2APIVersions {
		if normalizeSemver(tested) == hostVersion {
			result.Tested = true
			break
		}
	}
	if result.Tested {
		result.Status = "compatible"
		result.Message = "当前 Sub2API 版本已由插件声明测试"
	} else {
		result.Status = "untested"
		result.Message = "版本范围兼容，但插件未声明已测试当前 Sub2API 版本"
	}
	return result
}

func normalizeSemver(version string) string {
	v := version
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return ""
	}
	return v
}

func normalizePluginHostSemver(version string) string {
	if normalized := normalizeSemver(version); normalized != "" {
		return normalized
	}
	v := strings.TrimSpace(version)
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if parts := strings.Split(strings.TrimPrefix(v, "v"), "."); len(parts) == 4 {
		if parts[3] != "" && strings.IndexFunc(parts[3], func(r rune) bool { return r < '0' || r > '9' }) == -1 {
			return normalizeSemver(strings.Join(parts[:3], "."))
		}
	}
	return ""
}

func matchesSemverRange(version, expression string) bool {
	v := normalizeSemver(version)
	if v == "" || !validSemverRange(expression) {
		return false
	}
	tokens := strings.Fields(strings.ReplaceAll(expression, ",", " "))
	for _, token := range tokens {
		op := "="
		raw := token
		for _, candidate := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(token, candidate) {
				op = candidate
				raw = strings.TrimSpace(strings.TrimPrefix(token, candidate))
				break
			}
		}
		bound := normalizeSemver(raw)
		if bound == "" {
			return false
		}
		comparison := semver.Compare(v, bound)
		matched := map[string]bool{
			">=": comparison >= 0,
			"<=": comparison <= 0,
			">":  comparison > 0,
			"<":  comparison < 0,
			"=":  comparison == 0,
		}[op]
		if !matched {
			return false
		}
	}
	return true
}

func validSemverRange(expression string) bool {
	for _, segment := range strings.Split(expression, ",") {
		if strings.TrimSpace(segment) == "" {
			return false
		}
	}
	tokens := strings.Fields(strings.ReplaceAll(expression, ",", " "))
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		raw := token
		for _, candidate := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(token, candidate) {
				raw = strings.TrimSpace(strings.TrimPrefix(token, candidate))
				break
			}
		}
		if normalizeSemver(raw) == "" {
			return false
		}
	}
	return true
}
