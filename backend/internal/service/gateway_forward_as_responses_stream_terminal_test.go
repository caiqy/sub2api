package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type responsesScannerErrorBody struct {
	data []byte
	sent bool
}

func (b *responsesScannerErrorBody) Read(p []byte) (int, error) {
	if b.sent {
		return 0, errors.New("upstream stream read failed")
	}
	b.sent = true
	return copy(p, b.data), nil
}

func (*responsesScannerErrorBody) Close() error { return nil }

func TestHandleResponsesStreamingResponse_ReturnsScannerErrorWithoutCompleting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	resp := &http.Response{
		Header: http.Header{"x-request-id": []string{"rid_scan_error"}},
		Body: &responsesScannerErrorBody{data: []byte(strings.Join([]string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_scan_error","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4.5","stop_reason":"","usage":{"input_tokens":2}}}`,
			``,
		}, "\n"))},
	}

	result, err := (&GatewayService{}).handleResponsesStreamingResponse(resp, c, "claude-sonnet-4.5", "claude-sonnet-4.5", nil, time.Now(), apicompat.ResponsesClientToolMapping{})

	require.ErrorContains(t, err, "upstream stream read failed")
	require.Nil(t, result)
	require.NotContains(t, rec.Body.String(), "response.completed")
	require.NotContains(t, rec.Body.String(), `{"error":`)
}

func TestHandleResponsesStreamingResponse_IgnoresScannerErrorAfterCompletedTerminal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	resp := &http.Response{
		Header: http.Header{"x-request-id": []string{"rid_completed_then_error"}},
		Body: &responsesScannerErrorBody{data: []byte(strings.Join([]string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_completed","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4.5","stop_reason":"","usage":{"input_tokens":2}}}`,
			``,
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			``,
			`event: content_block_delta`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"done"}}`,
			``,
			`event: message_delta`,
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":1}}`,
			``,
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			``,
		}, "\n"))},
	}

	result, err := (&GatewayService{}).handleResponsesStreamingResponse(resp, c, "claude-sonnet-4.5", "claude-sonnet-4.5", nil, time.Now(), apicompat.ResponsesClientToolMapping{})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, strings.Count(rec.Body.String(), "event: response.completed\n"), rec.Body.String())
}

func TestWriteResponsesError_DoesNotAppendJSONAfterCommittedStream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.WriteHeader(http.StatusOK)
	_, err := c.Writer.WriteString("event: response.created\n\n")
	require.NoError(t, err)

	writeResponsesError(c, http.StatusServiceUnavailable, "server_error", "Failed to spool request body")

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "event: response.created\n\n", rec.Body.String())
}
