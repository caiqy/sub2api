package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/tidwall/gjson"
)

type originalInputGroupKey struct{}

func WithOriginalInputContextGroup(ctx context.Context, id *int64) context.Context {
	if _, ok := ctx.Value(originalInputGroupKey{}).(int64); ok {
		return ctx
	}
	var n int64
	if id != nil {
		n = *id
	}
	return context.WithValue(ctx, originalInputGroupKey{}, n)
}

type InputContextEstimate struct {
	Tokens   int
	Coverage string
	Source   string
}
type InputContextLimitDecision struct {
	InputContextEstimate
	Limit   int32
	Version int64
}

func (d *InputContextLimitDecision) Message() string {
	return fmt.Sprintf("Your input exceeds the configured maximum input context length of %d tokens (estimated input: %d tokens). Please reduce your input and try again.", d.Limit, d.Tokens)
}

// CheckInputContextLimit freezes configuration per request/turn; callers reuse snapshot
// through the returned context when reevaluating a fallback route.
type inputContextSnapshotKey struct{}

func (s *SettingService) CheckInputContextLimit(ctx context.Context, groupID *int64, model string, body []byte) (context.Context, *InputContextLimitDecision, error) {
	snap, ok := ctx.Value(inputContextSnapshotKey{}).(*cachedInputContextLimit)
	if !ok || snap == nil {
		snap = s.inputContextLimitSnapshot(ctx)
		ctx = context.WithValue(ctx, inputContextSnapshotKey{}, snap)
	}
	if !snap.settings.Enabled {
		return ctx, nil, nil
	}
	original, _ := ctx.Value(originalInputGroupKey{}).(int64)
	ids := []int64{original}
	if groupID != nil {
		ids = append(ids, *groupID)
	}
	var limit int32
	for _, id := range ids {
		n := snap.settings.GroupLimits[strconv.FormatInt(id, 10)]
		if n > 0 && (limit == 0 || n < limit) {
			limit = n
		}
	}
	if limit == 0 {
		return ctx, nil, nil
	}
	estimate, err := EstimateVisibleInputContext(model, body)
	if err != nil {
		return ctx, nil, err
	}
	if estimate.Tokens <= int(limit) {
		return ctx, nil, nil
	}
	return ctx, &InputContextLimitDecision{estimate, limit, snap.version}, nil
}

// EstimateVisibleInputContext counts only input-bearing fields. Opaque carriers
// are excluded from tokenization and explicitly lower coverage.
func EstimateVisibleInputContext(model string, body []byte) (InputContextEstimate, error) {
	result := InputContextEstimate{Coverage: "visible", Source: "local_tokenizer_estimate"}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return result, fmt.Errorf("invalid input JSON: %w", err)
	}
	codec, err := openAIInputTokensCodecForModel(model)
	if err != nil {
		return result, err
	}
	// Match encoding/json's SimpleFold equivalence (including ſ/S and K/K),
	// only for protocol field names, never for payload values or schema keys.
	fieldName := func(key gjson.Result) string {
		return strings.Map(func(r rune) rune {
			for {
				next := unicode.SimpleFold(r)
				if next <= r {
					if next >= 'A' && next <= 'Z' {
						return next + ('a' - 'A')
					}
					return next
				}
				r = next
			}
		}, key.String())
	}
	var texts []string
	var malformed error
	// Keep raw JSON keys in schemas and tool payloads: business keys such as
	// "data" and "signature" are visible here, not opaque protocol carriers.
	appendJSON := func(value gjson.Result) {
		if value.Type != gjson.Null {
			// gjson.Raw is valid JSON; compact without decoding away duplicate keys.
			var buf bytes.Buffer
			_ = json.Compact(&buf, []byte(value.Raw))
			texts = append(texts, buf.String())
		}
	}
	var visit func(gjson.Result)
	visit = func(x gjson.Result) {
		if x.Type == gjson.String {
			texts = append(texts, x.String())
			return
		}
		if x.IsArray() {
			x.ForEach(func(_, value gjson.Result) bool { visit(value); return true })
			return
		}
		if !x.IsObject() {
			return
		}
		var opaque, visibleType, toolUse bool
		x.ForEach(func(key, value gjson.Result) bool {
			if strings.EqualFold(key.String(), "type") {
				typ := value.String()
				toolUse = toolUse || typ == "tool_use"
				isOpaque := strings.Contains(typ, "image") || strings.Contains(typ, "audio") || strings.Contains(typ, "file") || typ == "document" || typ == "compaction" || typ == "reasoning"
				opaque = opaque || isOpaque
				visibleType = visibleType || !isOpaque
			}
			return true
		})
		if opaque {
			result.Coverage = "partial"
		}
		x.ForEach(func(key, value gjson.Result) bool {
			k := fieldName(key)
			if opaque && !visibleType && k != "text" && k != "summary" && k != "caption" {
				return true
			}
			switch k {
			case "image_url", "image", "inlinedata", "inline_data", "filedata", "file_data", "file_id", "file_url", "encrypted_content", "signature", "thoughtsignature", "thought_signature", "data", "audio", "videometadata":
				result.Coverage = "partial"
			case "parameters", "input_schema", "args", "arguments", "response", "schema":
				appendJSON(value)
			case "input":
				if toolUse {
					appendJSON(value)
				} else {
					visit(value)
				}
			case "content", "text", "thinking":
				if value.Type == gjson.String || value.Type == gjson.Null || value.IsArray() {
					visit(value)
				} else {
					malformed = fmt.Errorf("%s must contain text or content blocks", key.String())
				}
			case "id", "call_id", "tool_call_id", "role", "type", "status":
			default:
				visit(value)
			}
			return true
		})
	}
	// Like requestmodel candidates, retain every occurrence: gjson readers can
	// bind the first exact key while typed JSON binds later case variants.
	// ponytail: ambiguous duplicates are conservatively summed; protocol-specific
	// binding views would be needed to avoid overestimating those requests.
	gjson.ParseBytes(body).ForEach(func(key, value gjson.Result) bool {
		k := fieldName(key)
		if value.Type == gjson.Null {
			return true
		}
		switch k {
		case "messages", "contents", "tools", "functions":
			if !value.IsArray() {
				malformed = fmt.Errorf("%s must be an array", key.String())
				return true
			}
		case "input", "system":
			if value.Type != gjson.String && !value.IsArray() {
				malformed = fmt.Errorf("%s must be text or an array", key.String())
				return true
			}
		case "instructions":
			if value.Type != gjson.String {
				malformed = fmt.Errorf("instructions must be text")
				return true
			}
		case "tool_choice", "systeminstruction", "system_instruction", "response_format", "output_format":
		default:
			return true
		}
		if k == "tools" || k == "functions" || k == "response_format" || k == "output_format" {
			appendJSON(value)
		} else {
			visit(value)
		}
		return true
	})
	// Output schemas are model input; output budgets and sampling controls are not.
	gjson.ParseBytes(body).ForEach(func(key, value gjson.Result) bool {
		config := fieldName(key)
		switch config {
		case "text", "output_config", "generationconfig", "generation_config":
			if value.Type == gjson.Null {
				return true
			}
			if !value.IsObject() {
				malformed = fmt.Errorf("%s must be an object", key.String())
				return true
			}
			value.ForEach(func(field, schema gjson.Result) bool {
				f := fieldName(field)
				if (config == "text" || config == "output_config") && f == "format" ||
					(config == "generationconfig" || config == "generation_config") && (f == "responseschema" || f == "response_schema" || f == "responsejsonschema" || f == "response_json_schema") {
					appendJSON(schema)
				}
				return true
			})
		case "previous_response_id", "conversation", "cachedcontent", "cached_content":
			if value.Type != gjson.Null && value.Raw != `""` {
				result.Coverage = "partial"
			}
		}
		return true
	})
	if malformed != nil {
		return result, malformed
	}
	for _, text := range texts {
		n, e := codec.Count(text)
		if e != nil {
			return result, e
		}
		result.Tokens += n
	}
	return result, nil
}

func (s *OpenAIGatewayService) InputContextLimitSettingsService() *SettingService {
	if s == nil {
		return nil
	}
	return s.settingService
}
func FreshInputContextLimitTurn(ctx context.Context) context.Context {
	return context.WithValue(ctx, inputContextSnapshotKey{}, (*cachedInputContextLimit)(nil))
}
