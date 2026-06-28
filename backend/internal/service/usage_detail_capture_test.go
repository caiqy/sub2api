package service

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatUsageDetailRequestHeadersText_IncludesMethodURLAndHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/messages?debug=1", nil)
	require.NoError(t, err)
	req.Header.Add("X-Multi", "a")
	req.Header.Add("X-Multi", "b")
	req.Header.Set("Authorization", "Bearer secret-token")

	formatted := FormatUsageDetailRequestHeadersText(req)
	require.Contains(t, formatted, ":method: POST")
	require.Contains(t, formatted, ":url: https://example.com/v1/messages?debug=1")
	require.Contains(t, formatted, "Authorization: Bearer secret-token")
	require.Contains(t, formatted, "X-Multi: a")
	require.Contains(t, formatted, "X-Multi: b")
}

func TestFormatUsageDetailRequestHeadersText_StillIncludesMetaWhenHeadersEmpty(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/empty", nil)
	require.NoError(t, err)

	require.Equal(
		t,
		":method: GET\n:url: https://example.com/empty\n",
		FormatUsageDetailRequestHeadersText(req),
	)
}

func TestFormatUsageDetailRequestHeadersText_UsesHTTPSWhenRelativeURLHasTLS(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/relative?debug=1", nil)
	require.NoError(t, err)
	req.Host = "api.example.com"
	req.TLS = &tls.ConnectionState{}

	require.Equal(
		t,
		":method: GET\n:url: https://api.example.com/relative?debug=1\n",
		FormatUsageDetailRequestHeadersText(req),
	)
}

func TestFormatUsageDetailRequestHeadersText_PreservesEscapedRelativeRequestURI(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/v1/files/%2Ftmp?redirect=%2Ffoo%2Bbar&x=%252F", nil)
	require.NoError(t, err)
	req.Host = "api.example.com"
	req.RequestURI = "/v1/files/%2Ftmp?redirect=%2Ffoo%2Bbar&x=%252F"

	require.Equal(
		t,
		":method: GET\n:url: http://api.example.com/v1/files/%2Ftmp?redirect=%2Ffoo%2Bbar&x=%252F\n",
		FormatUsageDetailRequestHeadersText(req),
	)
}
