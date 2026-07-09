package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/tidwall/gjson"
)

type contentModerationInputCollector struct {
	textSegments []string
	textRunes    int
	images       []string
	maxImages    int
	seenImages   map[string]struct{}
}

func (c *contentModerationInputCollector) AddText(text string) {
	text = strings.TrimSpace(text)
	if text == "" || strings.Contains(text, "<system-reminder>") {
		return
	}
	text = normalizeContentModerationText(trimLatestRunesNoAlloc(text, maxModerationInputRunes))
	if text == "" {
		return
	}
	c.textSegments = append(c.textSegments, text)
	c.textRunes += len([]rune(text))
	// ponytail: latest-window audit; stream parsing is the upgrade if body memory itself dominates.
	for len(c.textSegments) > 0 && c.textRunes+len(c.textSegments)-1 > maxModerationInputRunes {
		oldest := c.textSegments[0]
		c.textSegments = c.textSegments[1:]
		c.textRunes -= len([]rune(oldest))
	}
}

func (c *contentModerationInputCollector) AddImageURL(image string) {
	if c.imageLimitReached() {
		return
	}
	image = strings.TrimSpace(image)
	if image == "" {
		return
	}
	if !strings.HasPrefix(image, "data:") && !strings.HasPrefix(image, "http://") && !strings.HasPrefix(image, "https://") {
		return
	}
	if c.seenImages == nil {
		c.seenImages = make(map[string]struct{})
	}
	if _, ok := c.seenImages[image]; ok {
		return
	}
	c.seenImages[image] = struct{}{}
	c.images = append(c.images, image)
}

func (c *contentModerationInputCollector) AddImageData(mimeType string, data string) {
	if c.imageLimitReached() {
		return
	}
	mimeType = strings.TrimSpace(mimeType)
	data = strings.TrimSpace(data)
	if mimeType == "" || data == "" {
		return
	}
	c.AddImageURL(fmt.Sprintf("data:%s;base64,%s", mimeType, data))
}

func (c *contentModerationInputCollector) Result() ContentModerationInput {
	return ContentModerationInput{Text: strings.Join(c.textSegments, " "), Images: normalizeModerationImages(c.images)}
}

func (c *contentModerationInputCollector) imageLimitReached() bool {
	return c.maxImages > 0 && len(c.images) >= c.maxImages
}

func trimLatestRunesNoAlloc(text string, max int) string {
	if max <= 0 {
		return ""
	}
	start := len(text)
	for count := 0; start > 0 && count < max; count++ {
		_, size := utf8.DecodeLastRuneInString(text[:start])
		if size <= 0 {
			break
		}
		start -= size
	}
	return text[start:]
}

func ExtractContentModerationText(protocol string, body []byte) string {
	return ExtractContentModerationInput(protocol, body).Text
}

func ExtractContentModerationInput(protocol string, body []byte) ContentModerationInput {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return ContentModerationInput{}
	}
	collector := contentModerationInputCollector{maxImages: maxContentModerationInputImages}
	switch protocol {
	case ContentModerationProtocolAnthropicMessages:
		collectAllAnthropicMessages(gjson.GetBytes(body, "messages"), &collector)
	case ContentModerationProtocolOpenAIChat:
		collectAllOpenAIChatMessages(gjson.GetBytes(body, "messages"), &collector)
	case ContentModerationProtocolOpenAIResponses:
		collectAllResponsesInput(gjson.GetBytes(body, "input"), &collector)
	case ContentModerationProtocolGemini:
		collectAllGeminiContents(gjson.GetBytes(body, "contents"), &collector)
	case ContentModerationProtocolOpenAIImages:
		collector.AddText(gjson.GetBytes(body, "prompt").String())
		collectContentValue(gjson.GetBytes(body, "images"), &collector)
	default:
		collectAllResponsesInput(gjson.GetBytes(body, "input"), &collector)
		collectAllOpenAIChatMessages(gjson.GetBytes(body, "messages"), &collector)
		collectAllGeminiContents(gjson.GetBytes(body, "contents"), &collector)
	}
	return collector.Result()
}

func collectAllOpenAIChatMessages(messages gjson.Result, collector *contentModerationInputCollector) {
	if !messages.IsArray() {
		return
	}
	array := messages.Array()
	if len(array) == 0 {
		return
	}
	for _, msg := range array {
		role := strings.ToLower(strings.TrimSpace(msg.Get("role").String()))
		switch role {
		case "user":
		case "assistant":
			if msg.Get("tool_calls").Exists() || msg.Get("function_call").Exists() {
				continue
			}
		default:
			continue
		}
		collectContentValue(msg.Get("content"), collector)
	}
}

func collectAnthropicUserContentValue(value gjson.Result, collector *contentModerationInputCollector) {
	switch {
	case !value.Exists():
		return
	case value.Type == gjson.String:
		if !isAnthropicSystemReminderText(value.String()) {
			collector.AddText(value.String())
		}
	case value.IsArray():
		value.ForEach(func(_, item gjson.Result) bool {
			collectAnthropicUserContentValue(item, collector)
			return true
		})
	case value.IsObject():
		typ := strings.ToLower(strings.TrimSpace(value.Get("type").String()))
		switch typ {
		case "", "text", "input_text", "message":
			if value.Get("text").Exists() && !isAnthropicSystemReminderText(value.Get("text").String()) {
				collector.AddText(value.Get("text").String())
			}
			if value.Get("content").Exists() {
				collectAnthropicUserContentValue(value.Get("content"), collector)
			}
		case "image_url", "input_image", "image":
			collectContentValue(value, collector)
		}
	}
}

func collectAnthropicAssistantTextOnly(value gjson.Result, collector *contentModerationInputCollector) {
	switch {
	case !value.Exists():
		return
	case value.Type == gjson.String:
		collector.AddText(value.String())
	case value.IsArray():
		value.ForEach(func(_, item gjson.Result) bool {
			collectAnthropicAssistantTextOnly(item, collector)
			return true
		})
	case value.IsObject():
		typ := strings.ToLower(strings.TrimSpace(value.Get("type").String()))
		if typ == "" || typ == "text" || typ == "output_text" {
			collector.AddText(value.Get("text").String())
		}
	}
}

func collectAllAnthropicMessages(messages gjson.Result, collector *contentModerationInputCollector) {
	if !messages.IsArray() {
		return
	}
	array := messages.Array()
	for _, msg := range array {
		role := strings.ToLower(strings.TrimSpace(msg.Get("role").String()))
		if role != "user" && role != "assistant" {
			continue
		}
		if role == "user" {
			collectAnthropicUserContentValue(msg.Get("content"), collector)
		} else {
			collectAnthropicAssistantTextOnly(msg.Get("content"), collector)
		}
	}
}

func isAnthropicSystemReminderText(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "<system-reminder>")
}

func collectAllResponsesInput(input gjson.Result, collector *contentModerationInputCollector) {
	switch {
	case !input.Exists():
		return
	case input.Type == gjson.String:
		collector.AddText(input.String())
	case input.IsArray():
		for _, item := range input.Array() {
			if !isResponsesAuditableItem(item) {
				continue
			}
			collectContentValue(item.Get("content"), collector)
			if item.Get("type").String() == "input_text" || item.Get("text").Exists() {
				collectContentValue(item, collector)
			}
		}
	case input.IsObject():
		if isResponsesAuditableItem(input) {
			collectContentValue(input.Get("content"), collector)
			if input.Get("type").String() == "input_text" || input.Get("text").Exists() {
				collectContentValue(input, collector)
			}
		}
	}
}

func isResponsesAuditableItem(item gjson.Result) bool {
	typ := strings.ToLower(strings.TrimSpace(item.Get("type").String()))
	role := strings.ToLower(strings.TrimSpace(item.Get("role").String()))
	if typ == "function_call" || typ == "function_call_output" || typ == "item_reference" {
		return false
	}
	if role == "user" || typ == "input_text" {
		return true
	}
	if role == "assistant" && (typ == "" || typ == "message") {
		return true
	}
	if role == "" && (typ == "message" || typ == "input_text") {
		return true
	}
	return false
}

func collectAllGeminiContents(contents gjson.Result, collector *contentModerationInputCollector) {
	if !contents.IsArray() {
		return
	}
	for _, content := range contents.Array() {
		role := strings.ToLower(strings.TrimSpace(content.Get("role").String()))
		if role != "" && role != "user" && role != "model" {
			continue
		}
		if arr := content.Get("parts"); arr.IsArray() {
			arr.ForEach(func(_, part gjson.Result) bool {
				if part.Get("functionCall").Exists() || part.Get("function_call").Exists() ||
					part.Get("functionResponse").Exists() || part.Get("function_response").Exists() {
					return true
				}
				collector.AddText(part.Get("text").String())
				addGeminiModerationImage(collector, part)
				return true
			})
		}
	}
}

func collectContentValue(value gjson.Result, collector *contentModerationInputCollector) {
	switch {
	case !value.Exists():
		return
	case value.Type == gjson.String:
		collector.AddText(value.String())
	case value.IsArray():
		value.ForEach(func(_, item gjson.Result) bool {
			collectContentValue(item, collector)
			return true
		})
	case value.IsObject():
		typ := strings.ToLower(strings.TrimSpace(value.Get("type").String()))
		collector.AddImageURL(value.Get("image_url.url").String())
		collector.AddImageURL(value.Get("image_url").String())
		collector.AddImageURL(value.Get("url").String())
		collector.AddImageData(value.Get("source.media_type").String(), value.Get("source.data").String())
		collector.AddImageData(value.Get("source.mediaType").String(), value.Get("source.data").String())
		collector.AddImageData(value.Get("media_type").String(), value.Get("data").String())
		collector.AddImageData(value.Get("mime_type").String(), value.Get("data").String())
		collector.AddImageData(value.Get("mimeType").String(), value.Get("data").String())
		collector.AddImageURL(value.Get("source.data").String())
		collector.AddImageURL(value.Get("data").String())
		collector.AddImageURL(value.Get("base64").String())
		switch typ {
		case "", "text", "input_text", "output_text", "message":
			if value.Get("text").Exists() {
				collector.AddText(value.Get("text").String())
			}
			if value.Get("content").Exists() {
				collectContentValue(value.Get("content"), collector)
			}
		case "image_url", "input_image", "image":
		}
	}
}

func addGeminiModerationImage(collector *contentModerationInputCollector, part gjson.Result) {
	if inlineData := part.Get("inline_data"); inlineData.IsObject() {
		collector.AddImageData(inlineData.Get("mime_type").String(), inlineData.Get("data").String())
	}
	if inlineData := part.Get("inlineData"); inlineData.IsObject() {
		collector.AddImageData(inlineData.Get("mimeType").String(), inlineData.Get("data").String())
	}
	collector.AddImageURL(part.Get("file_data.file_uri").String())
	collector.AddImageURL(part.Get("fileData.fileUri").String())
}

func normalizeModerationImages(images []string) []string {
	out := make([]string, 0, len(images))
	seen := make(map[string]struct{}, len(images))
	for _, image := range images {
		image = strings.TrimSpace(image)
		if image == "" {
			continue
		}
		if _, ok := seen[image]; ok {
			continue
		}
		seen[image] = struct{}{}
		out = append(out, image)
	}
	return out
}

func limitContentModerationImages(images []string) []string {
	if len(images) <= maxContentModerationInputImages {
		return images
	}
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(images))))
	if err != nil {
		return images[:maxContentModerationInputImages]
	}
	return []string{images[int(idx.Int64())]}
}

func normalizeContentModerationText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}
