package service

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const UsageDetailCaptureContextKey = "usage_detail_capture"

type usageDetailUpstreamRequestSetter interface {
	SetUsageUpstreamRequest(headers, body string)
}

type usageDetailResponseSnapshotSetter interface {
	SetUsageResponseSnapshot(headers, body string)
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

func FormatUsageDetailRequestHeadersText(req *http.Request) string {
	if req == nil {
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString(":method: ")
	buf.WriteString(req.Method)
	buf.WriteByte('\n')
	buf.WriteString(":url: ")
	buf.WriteString(formatUsageDetailRequestURL(req))
	buf.WriteByte('\n')
	buf.WriteString(FormatUsageDetailHeadersText(req.Header))

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

func SetUsageUpstreamRequest(c *gin.Context, req *http.Request, body string) {
	if c == nil || req == nil {
		return
	}
	if body == "" && req.GetBody != nil {
		clone, err := req.GetBody()
		if err == nil && clone != nil {
			payload, readErr := io.ReadAll(clone)
			_ = clone.Close()
			if readErr == nil {
				body = string(payload)
			}
		}
	}
	v, ok := c.Get(UsageDetailCaptureContextKey)
	if !ok {
		return
	}
	collector, ok := v.(usageDetailUpstreamRequestSetter)
	if !ok || collector == nil {
		return
	}
	collector.SetUsageUpstreamRequest(FormatUsageDetailRequestHeadersText(req), body)
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
