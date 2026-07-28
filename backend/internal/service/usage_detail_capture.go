package service

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const UsageDetailCaptureContextKey = "usage_detail_capture"

type usageDetailUpstreamRequestSetter interface {
	SetUsageUpstreamRequest(headers, body string)
}

type usageDetailUpstreamRequestHeadersSetter interface {
	SetUsageUpstreamRequestHeaders(headers string)
}

type usageDetailRequestBodySetter interface {
	SetUsageRequestBody(body string)
}

type usageDetailOriginalRequestBodySetter interface {
	SetUsageOriginalRequestBody(body string)
}

type usageDetailResponseSnapshotSetter interface {
	SetUsageResponseSnapshot(headers, body string)
}

type usageDetailUpstreamResponseSetter interface {
	SetUsageUpstreamResponse(headers, body string)
}

func FormatUsageDetailHeadersText(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	clone := headers.Clone()
	if len(clone) == 0 {
		return ""
	}
	var buf bytes.Buffer
	_ = clone.Write(&buf)
	return buf.String()
}

// FormatUsageDetailResponseHeadersText formats response headers with a
// leading `:status: <code>` pseudo-header so admin detail views can display
// the HTTP status code alongside the response headers.
func FormatUsageDetailResponseHeadersText(statusCode int, headers http.Header) string {
	var buf bytes.Buffer
	if statusCode > 0 {
		_, _ = buf.WriteString(":status: ")
		_, _ = buf.WriteString(strconv.Itoa(statusCode))
		_ = buf.WriteByte('\n')
	}
	_, _ = buf.WriteString(FormatUsageDetailHeadersText(headers))
	return buf.String()
}

func FormatUsageDetailRequestHeadersText(req *http.Request) string {
	if req == nil {
		return ""
	}

	var buf bytes.Buffer
	_, _ = buf.WriteString(":method: ")
	_, _ = buf.WriteString(req.Method)
	_ = buf.WriteByte('\n')
	_, _ = buf.WriteString(":url: ")
	_, _ = buf.WriteString(formatUsageDetailRequestURL(req))
	_ = buf.WriteByte('\n')
	_, _ = buf.WriteString(FormatUsageDetailHeadersText(req.Header))

	return buf.String()
}

func formatUsageDetailRequestURL(req *http.Request) string {
	if req == nil {
		return ""
	}
	if req.URL != nil && req.URL.IsAbs() {
		return req.URL.Redacted()
	}

	scheme := firstNonEmptyHeaderValue(req.Header, "X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if req.TLS != nil {
			scheme = "https"
		}
	}

	host := firstNonEmptyHeaderValue(req.Header, "X-Forwarded-Host")
	if host == "" {
		host = req.Host
	}
	if host == "" && req.URL != nil {
		host = req.URL.Host
	}

	requestURI := req.RequestURI
	if requestURI == "" && req.URL != nil {
		requestURI = req.URL.RequestURI()
	}

	if host == "" {
		if req.URL != nil {
			return req.URL.Redacted()
		}
		return requestURI
	}

	if requestURI == "" {
		return scheme + "://" + host
	}

	return scheme + "://" + host + requestURI
}

func firstNonEmptyHeaderValue(headers http.Header, key string) string {
	for _, value := range headers.Values(key) {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				return part
			}
		}
	}
	return ""
}

func SetUsageRequestBody(c *gin.Context, body string) {
	if c == nil {
		return
	}
	v, ok := c.Get(UsageDetailCaptureContextKey)
	if !ok {
		return
	}
	collector, ok := v.(usageDetailRequestBodySetter)
	if !ok || collector == nil {
		return
	}
	collector.SetUsageRequestBody(body)
}

func SetUsageOriginalRequestBody(c *gin.Context, body string) {
	if c == nil {
		return
	}
	v, ok := c.Get(UsageDetailCaptureContextKey)
	if !ok {
		return
	}
	collector, ok := v.(usageDetailOriginalRequestBodySetter)
	if !ok || collector == nil {
		return
	}
	collector.SetUsageOriginalRequestBody(body)
}

func SetUsageUpstreamRequest(c *gin.Context, req *http.Request, body string) {
	if c == nil || req == nil {
		return
	}
	size := req.ContentLength
	if size < 0 || size == 0 && body != "" {
		size = int64(len(body))
	}
	snapshot := RequestBodyPreviewSnapshot(body, size)
	c.Set(OpsUpstreamRequestBodyKey, snapshot)
	v, ok := c.Get(UsageDetailCaptureContextKey)
	if !ok {
		return
	}
	collector, ok := v.(usageDetailUpstreamRequestSetter)
	if !ok || collector == nil {
		return
	}
	collector.SetUsageUpstreamRequest(FormatUsageDetailRequestHeadersText(req), snapshot)
}

func SetUsageUpstreamRequestHeaders(c *gin.Context, req *http.Request) {
	if c == nil || req == nil {
		return
	}
	v, ok := c.Get(UsageDetailCaptureContextKey)
	if !ok {
		return
	}
	collector, ok := v.(usageDetailUpstreamRequestHeadersSetter)
	if !ok || collector == nil {
		return
	}
	collector.SetUsageUpstreamRequestHeaders(FormatUsageDetailRequestHeadersText(req))
}

func SetUsageResponseSnapshot(c *gin.Context, headers, body string) {
	if c == nil {
		return
	}
	v, ok := c.Get(UsageDetailCaptureContextKey)
	if !ok {
		return
	}
	collector, ok := v.(usageDetailResponseSnapshotSetter)
	if !ok || collector == nil {
		return
	}
	collector.SetUsageResponseSnapshot(headers, body)
}

func SetUsageUpstreamResponse(c *gin.Context, statusCode int, headers http.Header, body string) {
	if c == nil {
		return
	}
	v, ok := c.Get(UsageDetailCaptureContextKey)
	if !ok {
		return
	}
	collector, ok := v.(usageDetailUpstreamResponseSetter)
	if !ok || collector == nil {
		return
	}
	collector.SetUsageUpstreamResponse(FormatUsageDetailResponseHeadersText(statusCode, headers), body)
}
