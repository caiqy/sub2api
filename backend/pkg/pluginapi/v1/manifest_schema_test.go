package pluginv1

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
)

func TestManifestSchemaIsValidJSON(t *testing.T) {
	raw, err := os.ReadFile("manifest.schema.json")
	if err != nil {
		t.Fatalf("读取插件清单 Schema 失败: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("插件清单 Schema 不是有效 JSON: %v", err)
	}
	if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("插件清单 Schema 版本不符合预期: %v", schema["$schema"])
	}
}

func TestManifestSchemaVersionPatternsRejectForkSyntax(t *testing.T) {
	raw, err := os.ReadFile("manifest.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	defs, ok := schema["$defs"].(map[string]any)
	if !ok {
		t.Fatal("Schema 缺少 $defs")
	}
	for name, values := range map[string]struct {
		valid   []string
		invalid []string
	}{
		"semver": {
			valid:   []string{"0.1.180", "1.2.3-alpha.1", "1.2.3+fork.1"},
			invalid: []string{"0.1.180.1", " 1.2.3 ", "1.2.3-01"},
		},
		"semverRange": {
			valid:   []string{">=0.1.180 <0.2.0"},
			invalid: []string{">=0.1.180.1 <0.2.0", ">=0.1.180,,<0.2.0", ">=1.2.3-01 <2.0.0"},
		},
	} {
		definition, ok := defs[name].(map[string]any)
		if !ok {
			t.Fatalf("Schema 缺少 %s 定义", name)
		}
		pattern, ok := definition["pattern"].(string)
		if !ok {
			t.Fatalf("Schema %s 缺少 pattern", name)
		}
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("Schema %s pattern 无效: %v", name, err)
		}
		for _, value := range values.valid {
			if !compiled.MatchString(value) {
				t.Fatalf("Schema %s 拒绝合法值 %q", name, value)
			}
		}
		for _, value := range values.invalid {
			if compiled.MatchString(value) {
				t.Fatalf("Schema %s 接受非法值 %q", name, value)
			}
		}
	}
}
