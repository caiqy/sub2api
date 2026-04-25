package service

import (
	"context"
	"sync/atomic"
	"time"
)

type UsageLogDetailType string

const (
	UsageLogDetailTypeNormal UsageLogDetailType = "normal"
	UsageLogDetailTypeImage  UsageLogDetailType = "image"

	UsageLogDetailBestEffortTimeout          = 10 * time.Second
	UsageLogDetailRetentionLimitDefault      = 300
	ImageUsageLogDetailRetentionLimitDefault = 300

	UsageLogDetailRetentionLimit = UsageLogDetailRetentionLimitDefault
)

var usageLogDetailRetentionLimit atomic.Int64
var imageUsageLogDetailRetentionLimit atomic.Int64

func init() {
	usageLogDetailRetentionLimit.Store(UsageLogDetailRetentionLimitDefault)
	imageUsageLogDetailRetentionLimit.Store(ImageUsageLogDetailRetentionLimitDefault)
}

// SetUsageLogDetailRetentionLimits updates runtime retention limits.
// Negative values fall back to defaults; 0 is valid and means retain none.
func SetUsageLogDetailRetentionLimits(normalLimit int, imageLimit int) {
	if normalLimit < 0 {
		normalLimit = UsageLogDetailRetentionLimitDefault
	}
	if imageLimit < 0 {
		imageLimit = ImageUsageLogDetailRetentionLimitDefault
	}
	usageLogDetailRetentionLimit.Store(int64(normalLimit))
	imageUsageLogDetailRetentionLimit.Store(int64(imageLimit))
}

// GetUsageLogDetailRetentionLimits returns the current runtime retention limits.
func GetUsageLogDetailRetentionLimits() (normalLimit int, imageLimit int) {
	return int(usageLogDetailRetentionLimit.Load()), int(imageUsageLogDetailRetentionLimit.Load())
}

func NewUsageLogDetailBestEffortContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), UsageLogDetailBestEffortTimeout)
}

// UsageLogDetail 表示持久化后的 usage log 详情实体。
type UsageLogDetail struct {
	UsageLogID int64
	// DetailType is normal or image; persistence uses it to separate retention pools.
	DetailType              UsageLogDetailType
	RequestHeaders          string
	RequestBody             string
	UpstreamRequestHeaders  string
	UpstreamRequestBody     string
	ResponseHeaders         string
	ResponseBody            string
	UpstreamResponseHeaders string
	UpstreamResponseBody    string
	CreatedAt               time.Time
}

// UsageLogDetailSnapshot 保存 usage log 的请求/响应明细快照。
// 字段内容按原样保留，用于后续持久化或跨层传递。
type UsageLogDetailSnapshot struct {
	RequestHeaders          string
	RequestBody             string
	UpstreamRequestHeaders  string
	UpstreamRequestBody     string
	ResponseHeaders         string
	ResponseBody            string
	UpstreamResponseHeaders string
	UpstreamResponseBody    string
}

// Normalize 返回一个可安全持久化/传递的快照副本。
//
// 约定：
//   - nil-safe：nil 接收者直接返回 nil。
//   - 不 trim / 不格式化 / 不改写原文。
//   - 仅复制当前快照值，返回独立副本，避免后续调用方继续复用同一实例。
func (s *UsageLogDetailSnapshot) Normalize() *UsageLogDetailSnapshot {
	if s == nil {
		return nil
	}

	snapshot := *s
	return &snapshot
}
