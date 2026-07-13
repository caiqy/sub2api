package service

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIForwardResultUsageRecordSnapshotDropsHeavyFields(t *testing.T) {
	tier := "priority"
	effort := "high"
	firstTokenMs := 123
	result := &OpenAIForwardResult{
		RequestID:           "req-1",
		Model:               "gpt-5.1",
		UpstreamModel:       "gpt-5.1-upstream",
		ServiceTier:         &tier,
		ReasoningEffort:     &effort,
		ResponseHeaders:     http.Header{"X-Large": {"keep out of queue"}},
		FirstTokenMs:        &firstTokenMs,
		ImageOutputSizes:    []string{"1024x1024"},
		ImageSizeBreakdown:  map[string]int{"1024x1024": 2},
		wsReplayInput:       []json.RawMessage{json.RawMessage(`{"large":"payload"}`)},
		wsReplayInputExists: true,
	}

	snapshot := result.UsageRecordSnapshot()

	require.NotSame(t, result, snapshot)
	require.Equal(t, result.RequestID, snapshot.RequestID)
	require.Nil(t, snapshot.ResponseHeaders)
	require.Nil(t, snapshot.wsReplayInput)
	require.False(t, snapshot.wsReplayInputExists)
	require.NotSame(t, result.ServiceTier, snapshot.ServiceTier)
	require.NotSame(t, result.ReasoningEffort, snapshot.ReasoningEffort)
	require.NotSame(t, result.FirstTokenMs, snapshot.FirstTokenMs)
	require.Equal(t, []string{"1024x1024"}, snapshot.ImageOutputSizes)
	require.Equal(t, map[string]int{"1024x1024": 2}, snapshot.ImageSizeBreakdown)

	*result.ServiceTier = "mutated"
	*result.ReasoningEffort = "low"
	*result.FirstTokenMs = 456
	result.ImageOutputSizes[0] = "2048x2048"
	result.ImageSizeBreakdown["1024x1024"] = 9

	require.Equal(t, "priority", *snapshot.ServiceTier)
	require.Equal(t, "high", *snapshot.ReasoningEffort)
	require.Equal(t, 123, *snapshot.FirstTokenMs)
	require.Equal(t, []string{"1024x1024"}, snapshot.ImageOutputSizes)
	require.Equal(t, map[string]int{"1024x1024": 2}, snapshot.ImageSizeBreakdown)
}
