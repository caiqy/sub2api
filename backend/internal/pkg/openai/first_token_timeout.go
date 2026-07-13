package openai

import (
	"strings"

	"github.com/tidwall/gjson"
)

type FirstTokenClass string

const (
	FirstTokenClassText  FirstTokenClass = "text"
	FirstTokenClassImage FirstTokenClass = "image"
)

func ResponsesFirstTokenClass(payload []byte) FirstTokenClass {
	if strings.EqualFold(strings.TrimSpace(gjson.GetBytes(payload, "tool_choice.type").String()), "image_generation") {
		return FirstTokenClassImage
	}
	return FirstTokenClassText
}

func ResponsesEventEndsFirstTokenWait(payload []byte) bool {
	eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	return eventType != "" && eventType != "response.created" && eventType != "response.in_progress"
}

func ResponsesEventRecordsFirstToken(payload []byte) bool {
	eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	if eventType == "" || eventType == "error" ||
		strings.HasPrefix(eventType, "response.completed") ||
		strings.HasPrefix(eventType, "response.done") ||
		strings.HasPrefix(eventType, "response.failed") ||
		strings.HasPrefix(eventType, "response.incomplete") ||
		strings.HasPrefix(eventType, "response.cancel") {
		return false
	}
	return eventType != "response.created" && eventType != "response.in_progress"
}
