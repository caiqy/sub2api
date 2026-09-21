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

const (
	codexRecapFormat = `{"type":"json_schema","name":"codex_output_schema","strict":true,"schema":{"type":"object","properties":{"recap":{"type":"string","minLength":1,"maxLength":320}},"required":["recap"],"additionalProperties":false}}`
	codexTitleFormat = `{"type":"json_schema","name":"codex_output_schema","strict":true,"schema":{"type":"object","properties":{"title":{"type":"string","minLength":1,"maxLength":36}},"required":["title"],"additionalProperties":false}}`
)

type codexStringOutputSpec struct {
	field     string
	maxLength int
	format    string
}

var codexStringOutputSpecs = []codexStringOutputSpec{
	{field: "recap", maxLength: 320, format: codexRecapFormat},
	{field: "title", maxLength: 36, format: codexTitleFormat},
}

// OpenCode Go's DeepSeek rejects json_schema for Codex's small structured outputs.
// Match whole known schemas: unknown constraints must never be silently lost.
func adaptOpenCodeCodexStringOutput(c *gin.Context, targetURL string, req *apicompat.ResponsesRequest, chat *apicompat.ChatCompletionsRequest) *codexStringOutputSpec {
	if req.Text == nil || chat.Model != "deepseek-v4.1-flash" ||
		!openai.IsCodexOfficialClientByHeaders(c.GetHeader("User-Agent"), c.GetHeader("originator")) {
		return nil
	}
	u, err := url.Parse(targetURL)
	if err != nil || u.Scheme != "https" || u.Host != "opencode.ai" || u.Path != "/zen/go/v1/chat/completions" || u.RawQuery != "" {
		return nil
	}
	var actual any
	if json.Unmarshal(req.Text.Format, &actual) != nil {
		return nil
	}
	var spec *codexStringOutputSpec
	for i := range codexStringOutputSpecs {
		var expected any
		_ = json.Unmarshal([]byte(codexStringOutputSpecs[i].format), &expected)
		if reflect.DeepEqual(actual, expected) {
			spec = &codexStringOutputSpecs[i]
			break
		}
	}
	if spec == nil {
		return nil
	}
	// encoding/json keeps the last duplicate key. Check all four objects in
	// this exact schema so conflicting repeated constraints cannot opt in.
	format := gjson.ParseBytes(req.Text.Format)
	for _, object := range []gjson.Result{format, format.Get("schema"), format.Get("schema.properties"), format.Get("schema.properties." + spec.field)} {
		members := 0
		object.ForEach(func(_, _ gjson.Result) bool { members++; return true })
		if members != len(object.Map()) {
			return nil
		}
	}

	prompt, _ := json.Marshal(fmt.Sprintf("Return only a JSON object with exactly one field, %s. Put the requested result in that string, using the user's language and 1 to %d Unicode characters. Do not use markdown fences or extra fields. The required output format is: %s", spec.field, spec.maxLength, spec.format))
	chat.Messages = append(chat.Messages, apicompat.ChatMessage{Role: "system", Content: prompt})
	chat.ResponseFormat = json.RawMessage(`{"type":"json_object"}`)
	// The output is short. Buffer upstream JSON so no unvalidated output reaches
	// the client; its requested Responses streaming format is restored afterward.
	chat.Stream = false
	chat.StreamOptions = nil
	return spec
}

func validateCodexStringOutputResponse(resp *apicompat.ChatCompletionsResponse, spec *codexStringOutputSpec) (string, error) {
	if len(resp.Choices) != 1 || resp.Choices[0].FinishReason != "stop" {
		return "", errors.New("Codex structured output did not complete normally")
	}
	message := resp.Choices[0].Message
	if message.Role != "assistant" || len(message.ToolCalls) != 0 || message.FunctionCall != nil {
		return "", errors.New("Codex structured output is not an assistant text message")
	}
	var content string
	if !utf8.Valid(message.Content) || json.Unmarshal(message.Content, &content) != nil || !json.Valid([]byte(content)) {
		return "", errors.New("Codex structured output is not valid JSON text")
	}
	// Decode exactly one key/value pair, rejecting duplicates as well as extra
	// fields. Length is measured in Unicode code points, as JSON Schema requires.
	d := json.NewDecoder(strings.NewReader(content))
	start, _ := d.Token()
	key, _ := d.Token()
	value, err := d.Token()
	output, ok := value.(string)
	if start != json.Delim('{') || key != spec.field || err != nil || !ok || utf8.RuneCountInString(output) < 1 || utf8.RuneCountInString(output) > spec.maxLength {
		return "", fmt.Errorf("%s must be a string of 1 to %d characters", spec.field, spec.maxLength)
	}
	end, err := d.Token()
	if err != nil || end != json.Delim('}') {
		return "", errors.New("Codex structured output contains extra or duplicate fields")
	}
	if _, err := d.Token(); err != io.EOF {
		return "", errors.New("Codex structured output contains trailing data")
	}
	return content, nil
}

// Reuse the CC bridge's event lifecycle instead of hand-building Responses SSE.
func (s *OpenAIGatewayService) writeCodexStringOutputStream(c *gin.Context, resp *apicompat.ChatCompletionsResponse, model, content string) error {
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
			return fmt.Errorf("encode validated Codex structured output stream: %w", err)
		}
		if _, err := fmt.Fprint(c.Writer, sse); err != nil {
			return err
		}
	}
	c.Writer.Flush()
	return nil
}
