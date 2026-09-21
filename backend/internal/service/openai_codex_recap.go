package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const codexRecapFormat = `{"type":"json_schema","name":"codex_output_schema","strict":true,"schema":{"type":"object","properties":{"recap":{"type":"string","minLength":1,"maxLength":320}},"required":["recap"],"additionalProperties":false}}`

// OpenCode Go's DeepSeek rejects json_schema even for Codex's short catch-up.
// Match the whole known schema: unknown constraints must never be silently lost.
func adaptOpenCodeCodexRecap(c *gin.Context, targetURL string, req *apicompat.ResponsesRequest, chat *apicompat.ChatCompletionsRequest) bool {
	if req.Text == nil || chat.Model != "deepseek-v4.1-flash" ||
		!openai.IsCodexOfficialClientByHeaders(c.GetHeader("User-Agent"), c.GetHeader("originator")) {
		return false
	}
	u, err := url.Parse(targetURL)
	if err != nil || u.Scheme != "https" || u.Host != "opencode.ai" || u.Path != "/zen/go/v1/chat/completions" || u.RawQuery != "" {
		return false
	}
	var actual, expected any
	if json.Unmarshal(req.Text.Format, &actual) != nil {
		return false
	}
	_ = json.Unmarshal([]byte(codexRecapFormat), &expected)
	if !reflect.DeepEqual(actual, expected) {
		return false
	}
	// encoding/json keeps the last duplicate key. Check all four objects in
	// this exact schema so conflicting repeated constraints cannot opt in.
	format := gjson.ParseBytes(req.Text.Format)
	for _, object := range []gjson.Result{format, format.Get("schema"), format.Get("schema.properties"), format.Get("schema.properties.recap")} {
		members := 0
		object.ForEach(func(_, _ gjson.Result) bool { members++; return true })
		if members != len(object.Map()) {
			return false
		}
	}

	prompt, _ := json.Marshal("Return only a JSON object with exactly one field, recap. Put the requested catch-up in that string, using the user's language and 1 to 320 Unicode characters. Do not use markdown fences or extra fields. The required output format is: " + codexRecapFormat)
	chat.Messages = append(chat.Messages, apicompat.ChatMessage{Role: "system", Content: prompt})
	chat.ResponseFormat = json.RawMessage(`{"type":"json_object"}`)
	// The recap is short. Buffer upstream JSON so no unvalidated output reaches
	// the client; its requested Responses streaming format is restored afterward.
	chat.Stream = false
	chat.StreamOptions = nil
	return true
}

func validateCodexRecapResponse(resp *apicompat.ChatCompletionsResponse) (string, error) {
	if len(resp.Choices) != 1 || resp.Choices[0].FinishReason != "stop" {
		return "", errors.New("recap response did not complete normally")
	}
	message := resp.Choices[0].Message
	if message.Role != "assistant" || len(message.ToolCalls) != 0 || message.FunctionCall != nil {
		return "", errors.New("recap response is not an assistant text message")
	}
	var content string
	if !utf8.Valid(message.Content) || json.Unmarshal(message.Content, &content) != nil || !json.Valid([]byte(content)) {
		return "", errors.New("recap response is not valid JSON text")
	}
	// Decode exactly one key/value pair, rejecting duplicates as well as extra
	// fields. Length is measured in Unicode code points, as JSON Schema requires.
	d := json.NewDecoder(strings.NewReader(content))
	start, _ := d.Token()
	key, _ := d.Token()
	value, err := d.Token()
	recap, ok := value.(string)
	if start != json.Delim('{') || key != "recap" || err != nil || !ok || utf8.RuneCountInString(recap) < 1 || utf8.RuneCountInString(recap) > 320 {
		return "", errors.New("recap must be a string of 1 to 320 characters")
	}
	end, err := d.Token()
	if err != nil || end != json.Delim('}') {
		return "", errors.New("recap response contains extra or duplicate fields")
	}
	if _, err := d.Token(); err != io.EOF {
		return "", errors.New("recap response contains trailing data")
	}
	return content, nil
}

// Reuse the CC bridge's event lifecycle instead of hand-building Responses SSE.
func (s *OpenAIGatewayService) writeCodexRecapStream(c *gin.Context, resp *apicompat.ChatCompletionsResponse, model, content string) error {
	choice := resp.Choices[0]
	reasoning := choice.Message.ReasoningContent
	if reasoning == "" {
		reasoning = choice.Message.Reasoning
	}
	state := apicompat.NewChatCompletionsToResponsesStreamState(model)
	events := apicompat.ChatCompletionsChunkToResponsesEvents(&apicompat.ChatCompletionsChunk{
		ID: resp.ID, Model: model, Created: resp.Created, Usage: resp.Usage, ServiceTier: resp.ServiceTier,
		Choices: []apicompat.ChatChunkChoice{{
			Index: 0, FinishReason: &choice.FinishReason,
			Delta: apicompat.ChatDelta{Role: "assistant", Content: &content, ReasoningContent: &reasoning},
		}},
	}, state)
	events = append(events, apicompat.FinalizeChatCompletionsResponsesStream(state)...)
	s.cacheReasoningItemsFromEvents(events)
	for _, event := range events {
		sse, err := apicompat.ResponsesEventToSSE(event)
		if err != nil {
			return fmt.Errorf("encode validated recap stream: %w", err)
		}
		if _, err := fmt.Fprint(c.Writer, sse); err != nil {
			return err
		}
	}
	c.Writer.Flush()
	return nil
}
