package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func inputContextDecision(c *gin.Context, s *service.SettingService, key *service.APIKey, model string, body []byte) (*service.InputContextLimitDecision, error) {
	ctx := service.WithOriginalInputContextGroup(c.Request.Context(), key.GroupID)
	ctx, d, err := s.CheckInputContextLimit(ctx, key.GroupID, model, body)
	c.Request = c.Request.WithContext(ctx)
	if d != nil {
		service.MarkOpsStreamErrorValue(c, service.OpsStreamError{ErrType: "invalid_request_error", Code: "context_length_exceeded", Message: d.Message(), IntendedStatus: 400, RequestScoped: true})
		service.MarkOpsClientBusinessLimited(c, "input_context_limit_exceeded")
		slog.Info("input context limit rejected", "model", model, "group_id", func() int64 {
			if key.GroupID != nil {
				return *key.GroupID
			}
			return 0
		}(), "protocol", c.Request.URL.Path, "tokens", d.Tokens, "limit", d.Limit, "coverage", d.Coverage, "source", d.Source, "version", d.Version)
	}
	return d, err
}
func rejectInputContext(c *gin.Context, s *service.SettingService, key *service.APIKey, model string, body []byte, protocol string) bool {
	if protocol == "responses" && service.IsOpenAIResponsesCompactPath(c) {
		return false
	}
	d, err := inputContextDecision(c, s, key, model, body)
	if err == nil && d == nil {
		return false
	}
	message := "Invalid input: failed to estimate visible input context"
	code := "invalid_request_error"
	if d != nil {
		message = d.Message()
		code = "context_length_exceeded"
	}
	switch protocol {
	case "messages":
		if d != nil {
			message = fmt.Sprintf("prompt is too long: %d tokens > %d maximum (gateway estimated input limit)", d.Tokens, d.Limit)
		}
		rid, _ := c.Request.Context().Value(ctxkey.RequestID).(string)
		c.JSON(400, gin.H{"type": "error", "error": gin.H{"type": "invalid_request_error", "message": message}, "request_id": rid})
	case "gemini":
		if d != nil {
			message = fmt.Sprintf("The input token count (%d) exceeds the maximum number of tokens allowed (%d). (gateway estimated input limit)", d.Tokens, d.Limit)
		}
		c.JSON(400, gin.H{"error": gin.H{"code": 400, "status": "INVALID_ARGUMENT", "message": message}})
	default:
		if protocol == "responses" && d != nil && gjson.GetBytes(body, "stream").Bool() {
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Status(http.StatusOK)
			writeResponsesFailedSSE(c, "invalid_request_error", code, message)
		} else {
			param := "messages"
			if protocol == "responses" {
				param = "input"
			}
			c.JSON(400, gin.H{"error": gin.H{"type": "invalid_request_error", "code": code, "param": param, "message": message}})
		}
	}
	return true
}
func rejectInputContextWS(c *gin.Context, s *service.SettingService, key *service.APIKey, model string, body []byte, conn *coderws.Conn) bool {
	d, err := inputContextDecision(c, s, key, model, body)
	if d == nil && err == nil {
		return false
	}
	message := "Invalid input context"
	code := "invalid_request_error"
	if d != nil {
		message = d.Message()
		code = "context_length_exceeded"
	}
	event := responsesFailedEvent{Type: "response.failed", Response: responsesFailedBody{ID: synthesizeResponseID(c), Object: "response", CreatedAt: time.Now().Unix(), Model: model, Status: "failed", Output: []any{}, Error: responsesFailedError{Code: code, Message: message}}}
	payload := map[string]any{"type": event.Type, "sequence_number": event.SequenceNumber, "response": event.Response}
	if streamID := gjson.GetBytes(body, "stream_id"); streamID.Exists() {
		payload["stream_id"] = streamID.Value()
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Second)
	defer cancel()
	encoded, _ := json.Marshal(payload)
	_ = conn.Write(ctx, coderws.MessageText, encoded)
	return true
}
