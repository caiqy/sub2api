package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestSafeUpstreamURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"strips query", "https://api.anthropic.com/v1/messages?beta=true", "https://api.anthropic.com/v1/messages"},
		{"strips fragment", "https://api.openai.com/v1/responses#frag", "https://api.openai.com/v1/responses"},
		{"strips both", "https://host/path?token=secret#x", "https://host/path"},
		{"no query or fragment", "https://host/path", "https://host/path"},
		{"empty string", "", ""},
		{"whitespace only", "  ", ""},
		{"query before fragment", "https://h/p?a=1#f", "https://h/p"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, safeUpstreamURL(tt.input))
		})
	}
}

type opsUpstreamUsageCollector struct{ body string }

func (c *opsUpstreamUsageCollector) SetUsageUpstreamRequest(_ string, body string) { c.body = body }

func TestSetUsageUpstreamRequestKeepsAccurateOpsWrapper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	collector := &opsUpstreamUsageCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)
	body := `{"input":"preview"}`
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", strings.NewReader(body))
	require.NoError(t, err)
	req.ContentLength = 10 << 20

	SetUsageUpstreamRequest(c, req, body)

	raw, ok := c.Get(OpsUpstreamRequestBodyKey)
	require.True(t, ok)
	wrapper, ok := raw.(string)
	require.True(t, ok)
	require.Equal(t, collector.body, wrapper)
	require.Equal(t, uintptr(unsafe.Pointer(unsafe.StringData(collector.body))), uintptr(unsafe.Pointer(unsafe.StringData(wrapper))))
	require.Equal(t, "request_body_preview", gjson.Get(wrapper, "kind").String())
	require.Equal(t, int64(10<<20), gjson.Get(wrapper, "size").Int())
	require.True(t, gjson.Get(wrapper, "truncated").Bool())
}

func TestSetUsageUpstreamRequestWritesOpsWithoutUsageCollector(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	body := `{"input":"preview"}`
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", strings.NewReader(body))
	require.NoError(t, err)

	SetUsageUpstreamRequest(c, req, body)

	raw, ok := c.Get(OpsUpstreamRequestBodyKey)
	require.True(t, ok)
	rawBody, ok := raw.(string)
	require.True(t, ok)
	require.Equal(t, body, gjson.Get(rawBody, "preview").String())
}

func TestSetUsageUpstreamRequestFallsBackToPreviewSizeForUnknownContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	collector := &opsUpstreamUsageCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)
	body := `{"input":"preview"}`
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", strings.NewReader(body))
	require.NoError(t, err)
	req.ContentLength = 0

	SetUsageUpstreamRequest(c, req, body)

	require.Equal(t, int64(len(body)), gjson.Get(collector.body, "size").Int())
	require.False(t, gjson.Get(collector.body, "truncated").Bool())
}

func TestUsageUpstreamRequestOwnsOpsSnapshotContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	collector := &opsUpstreamUsageCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/messages", strings.NewReader(`{"messages":["preview"]}`))
	require.NoError(t, err)
	req.ContentLength = 10 << 20

	SetUsageUpstreamRequest(c, req, `{"messages":["preview"]}`)

	raw, ok := c.Get(OpsUpstreamRequestBodyKey)
	require.True(t, ok)
	require.Equal(t, collector.body, raw)
	require.Equal(t, int64(10<<20), gjson.Get(collector.body, "size").Int())
	require.True(t, gjson.Get(collector.body, "truncated").Bool())
}
