package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImageRawStreamKeepsStandardObservation(t *testing.T) {
	payload := `{"type":"image_edit.completed","b64_json":"AA==","usage":{"input_tokens":1558,"output_tokens":343}}`
	for _, async := range []bool{false, true} {
		for i, wire := range []string{
			"data: " + payload + "\nunknown: ignored\n\n",
			"data: " + strings.ReplaceAll(payload, `,"usage"`, ",\ndata: \"usage\"") + "\n\n",
			"data: " + payload + "\ndata: {\"usage\":{\"input_tokens\":1558}}\n\n",
			"data: " + strings.ReplaceAll(payload, `,"usage"`, ",\n: comment\nevent: image_edit.completed\nid: opaque\nretry: 10\n\"usage\"") + "\n\n",
		} {
			t.Run(fmt.Sprintf("async_%t/wire_%d", async, i), func(t *testing.T) {
				cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: 1024}}
				if async {
					cfg.Gateway.ImageStreamDataIntervalTimeout = 1
				}
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
				resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(wire))}
				svc := &OpenAIGatewayService{cfg: cfg}
				usage, count, _, _, err := svc.handleOpenAIImagesStreamingResponse(resp, c, time.Now(), nil)
				require.NoError(t, err)
				require.Equal(t, wire, rec.Body.String())
				require.Equal(t, OpenAIUsage{InputTokens: 1558, OutputTokens: 343}, usage)
				require.Equal(t, 1, count)
			})
		}
	}
}

func TestImageRawStreamRejectsOverLimitObservation(t *testing.T) {
	completed := "event: image_edit.completed\ndata: {\"type\":\"image_edit.completed\",\"b64_json\":\"AA==\",\"usage\":{\n \"input_tokens\":1558,\n \"output_tokens\":343\n }}\n"
	for _, async := range []bool{false, true} {
		for _, limit := range []string{"line", "bytes", "lines", "whitespace"} {
			t.Run(fmt.Sprintf("async_%t/%s", async, limit), func(t *testing.T) {
				cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: 1024, SSEQueueMaxBytes: 16}}
				tail := ":" + strings.Repeat("x", 1024) + "\n\n"
				switch limit {
				case "bytes":
					cfg.Gateway.SSEEventMaxBytes = int64(len(completed))
					tail = ":overflow\n\n"
				case "lines":
					cfg.Gateway.SSEEventMaxLines = 5
					tail = ":overflow\n\n"
				case "whitespace":
					cfg.Gateway.SSEEventMaxLines = 6
					tail = " \n:overflow\n\n"
				}
				if async {
					cfg.Gateway.ImageStreamDataIntervalTimeout = 1
				}
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
				body := &imageLimitTestBody{Reader: strings.NewReader(completed + tail)}
				resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: body}
				svc := &OpenAIGatewayService{cfg: cfg}
				usage, count, _, _, err := svc.handleOpenAIImagesStreamingResponse(resp, c, time.Now(), nil)
				require.ErrorIs(t, err, errSSETransportLimit)
				require.True(t, body.closed.Load())
				require.Zero(t, usage.InputTokens)
				require.Zero(t, usage.OutputTokens)
				require.Zero(t, count)
				require.NotContains(t, rec.Body.String(), "image_edit.completed")
				requireImageTransportLimitError(t, rec.Body.String())
			})
		}
	}
}

func TestImageRawStreamObservesUnprefixedJSONContinuations(t *testing.T) {
	for _, async := range []bool{false, true} {
		for _, eof := range []bool{false, true} {
			for _, disconnected := range []bool{false, true} {
				t.Run(fmt.Sprintf("async_%t/eof_%t/disconnected_%t", async, eof, disconnected), func(t *testing.T) {
					wire := "event: image_edit.completed\ndata: {\"type\":\"image_edit.completed\",\"b64_json\":\"AA==\",\"usage\":{\n \"input_tokens\":1558,\n \"output_tokens\":343\n },\"model\":\"gpt-image-2.6-sunburst\"}"
					if !eof {
						wire += "\n\n"
					}
					if eof {
						wire = strings.ReplaceAll(wire, "\n", "\r\n")
					}
					if disconnected {
						wire = ": heartbeat\n\n" + wire // Fail before the image, then drain for billing.
					}
					cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: 1024, SSEQueueMaxBytes: 16}}
					if async {
						cfg.Gateway.ImageStreamDataIntervalTimeout = 1
					}
					rec := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(rec)
					if disconnected {
						c.Writer = &imageLimitDisconnectedWriter{ResponseWriter: c.Writer}
					}
					c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
					resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(wire))}
					svc := &OpenAIGatewayService{cfg: cfg}
					usage, count, _, _, err := svc.handleOpenAIImagesStreamingResponse(resp, c, time.Now(), nil)
					require.NoError(t, err)
					if !disconnected {
						require.Equal(t, wire, rec.Body.String())
					}
					require.Equal(t, OpenAIUsage{InputTokens: 1558, OutputTokens: 343}, usage)
					require.Equal(t, 1, count)
				})
			}
		}
	}
}
