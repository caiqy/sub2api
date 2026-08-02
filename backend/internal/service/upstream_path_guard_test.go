package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSanitizedUpstreamPathSuffixRejectsNonConformingSegments(t *testing.T) {
	// 到达业务代码的 URL.Path 已是百分号解码后的结果，因此用例按解码后的形态书写。
	rejected := []string{
		"../models",
		"../../models",
		"...",
		"/..",
		"/../..",
		"/../../x/y",
		"/./compact",
		"/compact/..",
		`/..\..\x`,
		`/compact\..`,
		`/compact\detail`,
		`compact\detail`,
		"compact?next=%2fmodels",
		"compact#fragment",
		"/?a=b",
		"/compact?a=b",
		"/compact#frag",
		"/compact%2f..",
		"/100%",
		"//double",
		"/compact//detail",
		"/compact/",
		"/ compact",
		"/compact\x00",
		"/compact\nX-Injected: 1",
		"/模型",
		"compact",
		"/a:b",
		"/a;b",
		"/a,b",
		"/a=b",
		"/a&b",
		// 允许清单是闭集：`\w` + `-` + `.` 以外的字符一律拒绝，
		// 不依赖任何"已知坏字符"清单。
		"/a~b",
		"/a@b",
		"/a+b",
		"/a|b",
		"/a*b",
		"/a$b",
		"/a(b)",
		"/a'b",
		"/a\"b",
		"/a<b",
		"/a\tb",
		"/a b",
		"/a∕b", // DIVISION SLASH
		"/a／b", // FULLWIDTH SOLIDUS
		// 只由点组成的片段一律拒绝（各实现对其解释不一致）。
		"/...",
		"/....",
		"/compact/...",
	}
	for _, suffix := range rejected {
		t.Run("reject_"+suffix, func(t *testing.T) {
			got, ok := sanitizedUpstreamPathSuffix(suffix)
			require.False(t, ok, "suffix %q must be rejected", suffix)
			require.Empty(t, got)
		})
	}

	accepted := map[string]string{
		"":                          "",
		"/compact":                  "/compact",
		"/compact/detail":           "/compact/detail",
		"/resp_68f0a1b2c3d4/cancel": "/resp_68f0a1b2c3d4/cancel",
		"/gemini-2.5-pro_v1.2":      "/gemini-2.5-pro_v1.2",
		"/a.b.c":                    "/a.b.c",
	}
	for suffix, want := range accepted {
		t.Run("accept_"+suffix, func(t *testing.T) {
			got, ok := sanitizedUpstreamPathSuffix(suffix)
			require.True(t, ok, "suffix %q must be accepted", suffix)
			require.Equal(t, want, got)
		})
	}
}

func TestSanitizedUpstreamPathSuffixEnforcesBounds(t *testing.T) {
	const tooLongSegment = "/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const tooManySegments = "/a/a/a/a/a/a/a/a/a"

	require.Equal(t, 128, maxUpstreamPathSegmentLen)
	require.Equal(t, 8, maxUpstreamPathSegments)
	require.Len(t, tooLongSegment[1:], 129)
	_, ok := sanitizedUpstreamPathSuffix(tooLongSegment)
	require.False(t, ok, "over-long segment must be rejected")

	_, ok = sanitizedUpstreamPathSuffix(tooManySegments)
	require.False(t, ok, "over-deep suffix must be rejected")
}

// TestOpenAIResponsesRequestPathSuffixRejectsNonConformingSubpaths 锁定不变式：
// /responses/*subpath 的子路径不得改变上游请求的路径结构；不合规时既不参与拼接，
// 也不会被误判成 compact 请求。
func TestOpenAIResponsesRequestPathSuffixRejectsNonConformingSubpaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, prefix := range []string{
		"/v1/responses",
		"/responses",
		"/backend-api/codex/responses",
	} {
		for _, suffix := range []string{
			"/../models",
			"/../../models",
			"/...",
			"/%2e%2e/models",
			"/%2E%2E%2Fmodels",
			"/compact%2f..%2fmodels",
			`/compact\detail`,
			"/%5c..%5cmodels",
			"//double",
			"/compact//detail",
		} {
			path := prefix + suffix
			t.Run(path, func(t *testing.T) {
				c := newResponsesSuffixTestContext(t, path)

				require.False(t, IsForwardableOpenAIResponsesRequestPath(c),
					"path %q must be rejected at the gateway edge", path)
				require.Empty(t, openAIResponsesRequestPathSuffix(c),
					"path %q must never contribute an upstream path suffix", path)
				require.Equal(t, chatgptCodexURL,
					appendOpenAIResponsesRequestPathSuffix(chatgptCodexURL, openAIResponsesRequestPathSuffix(c)))
				require.False(t, isOpenAIResponsesCompactPath(c))
			})
		}
	}

	// 合法子路径必须保持原样转发。
	for path, want := range map[string]string{
		"/v1/responses":                                       "",
		"/v1/responses/":                                      "",
		"/v1/responses/compact":                               "/compact",
		"/v1/responses/compact?next=%2fmodels":                "/compact",
		"/responses/compact?next=%2fmodels":                   "/compact",
		"/backend-api/codex/responses/compact?next=%2fmodels": "/compact",
		"/responses/compact/":                                 "/compact",
		"/backend-api/codex/responses/compact":                "/compact",
	} {
		t.Run("forwardable_"+path, func(t *testing.T) {
			c := newResponsesSuffixTestContext(t, path)
			require.True(t, IsForwardableOpenAIResponsesRequestPath(c))
			require.Equal(t, want, openAIResponsesRequestPathSuffix(c))
		})
	}
}

func TestAppendOpenAIResponsesRequestPathSuffixRefusesUnsafeSuffix(t *testing.T) {
	// 调用方漏了校验时，拼接函数本身也不得把不合规片段带进上游 URL。
	require.Equal(t, chatgptCodexURL, appendOpenAIResponsesRequestPathSuffix(chatgptCodexURL, "/../../x"))
	require.Equal(t, chatgptCodexURL, appendOpenAIResponsesRequestPathSuffix(chatgptCodexURL, "/?a=b"))
	require.Equal(t, chatgptCodexURL+"/compact", appendOpenAIResponsesRequestPathSuffix(chatgptCodexURL, "/compact"))
}

func newResponsesSuffixTestContext(t *testing.T, path string) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, path, nil)
	return c
}
