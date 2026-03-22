package service

import (
	"context"
	"time"
)

const (
	UsageLogDetailBestEffortTimeout = 10 * time.Second
	UsageLogDetailRetentionLimit    = 500
)

func NewUsageLogDetailBestEffortContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), UsageLogDetailBestEffortTimeout)
}

// UsageLogDetail 表示持久化后的 usage log 详情实体。
type UsageLogDetail struct {
	UsageLogID             int64
	RequestHeaders         string
	RequestBody            string
	UpstreamRequestHeaders string
	UpstreamRequestBody    string
	ResponseHeaders        string
	ResponseBody           string
	CreatedAt              time.Time
}

// UsageLogDetailSnapshot 保存 usage log 的请求/响应明细快照。
// 字段内容按原样保留，用于后续持久化或跨层传递。
type UsageLogDetailSnapshot struct {
	RequestHeaders         string
	RequestBody            string
	UpstreamRequestHeaders string
	UpstreamRequestBody    string
	ResponseHeaders        string
	ResponseBody           string
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
