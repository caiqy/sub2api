package openai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponsesFirstTokenClass(t *testing.T) {
	tests := []struct {
		name string
		body string
		want FirstTokenClass
	}{
		{
			name: "常驻图片工具仍为文本档",
			body: `{"stream":true,"tools":[{"type":"image_generation"}],"tool_choice":"auto"}`,
			want: FirstTokenClassText,
		},
		{
			name: "HTTP 强制图片工具为图片档",
			body: `{"stream":true,"tool_choice":{"type":"image_generation"}}`,
			want: FirstTokenClassImage,
		},
		{
			name: "WebSocket 强制图片工具为图片档",
			body: `{"type":"response.create","tool_choice":{"type":"image_generation"}}`,
			want: FirstTokenClassImage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ResponsesFirstTokenClass([]byte(tt.body)))
		})
	}
}

func TestResponsesFirstTokenClassReaderUsesFinalPayload(t *testing.T) {
	body := `{"input":{"nested":[{"text":"hello"},{"more":[1,2,3]}]},"tool_choice":{"type":"image_generation"}}`
	require.Equal(t, FirstTokenClassImage, ResponsesFirstTokenClassReader(strings.NewReader(body)))
}

func TestResponsesFirstTokenClassReaderKeepsLargeSkippedStringAllocationBounded(t *testing.T) {
	payload := `{"input":"` + strings.Repeat("a", 1<<20) + `","tool_choice":{"type":"image_generation"}}`
	result := testing.Benchmark(func(b *testing.B) {
		for range b.N {
			if got := ResponsesFirstTokenClassReader(strings.NewReader(payload)); got != FirstTokenClassImage {
				b.Fatalf("class = %s", got)
			}
		}
	})
	require.Less(t, result.AllocedBytesPerOp(), int64(128<<10))
}

func TestResponsesFirstTokenEventClassification(t *testing.T) {
	require.False(t, ResponsesEventEndsFirstTokenWait([]byte(`{"type":"response.created"}`)))
	require.False(t, ResponsesEventEndsFirstTokenWait([]byte(`{"type":"response.in_progress"}`)))
	require.True(t, ResponsesEventEndsFirstTokenWait([]byte(`{"type":"response.output_item.added","item":{"type":"image_generation_call","status":"in_progress"}}`)))
	require.True(t, ResponsesEventRecordsFirstToken([]byte(`{"type":"response.output_text.delta","delta":"x"}`)))
	require.False(t, ResponsesEventRecordsFirstToken([]byte(`{"type":"response.failed"}`)))
	require.False(t, ResponsesEventRecordsFirstToken([]byte(`{"type":"response.completed"}`)))
	require.False(t, ResponsesEventRecordsFirstToken([]byte(`{"type":"error"}`)))
}
