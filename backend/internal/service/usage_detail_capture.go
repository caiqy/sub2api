package service

import (
	"bytes"
	"io"
	"net/http"

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
	collector.SetUsageUpstreamRequest(FormatUsageDetailHeadersText(req.Header), body)
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
