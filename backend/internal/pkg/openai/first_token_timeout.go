package openai

import (
	"bufio"
	"io"
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

func ResponsesFirstTokenClassReader(reader io.Reader) FirstTokenClass {
	if reader == nil {
		return FirstTokenClassText
	}
	input := bufio.NewReader(reader)
	start, err := readJSONNonSpace(input)
	if err != nil || start != '{' {
		return FirstTokenClassText
	}
	for {
		next, err := readJSONNonSpace(input)
		if err != nil || next == '}' {
			return FirstTokenClassText
		}
		if next != '"' {
			return FirstTokenClassText
		}
		key, err := readJSONString(input, true)
		if err != nil {
			return FirstTokenClassText
		}
		colon, err := readJSONNonSpace(input)
		if err != nil || colon != ':' {
			return FirstTokenClassText
		}
		if key == "tool_choice" {
			return readToolChoiceClass(input)
		}
		if skipJSONValue(input) != nil {
			return FirstTokenClassText
		}
		separator, err := readJSONNonSpace(input)
		if err != nil || separator == '}' {
			return FirstTokenClassText
		}
		if separator != ',' {
			return FirstTokenClassText
		}
	}
}

func readToolChoiceClass(input *bufio.Reader) FirstTokenClass {
	start, err := readJSONNonSpace(input)
	if err != nil {
		return FirstTokenClassText
	}
	if start != '{' {
		_ = skipJSONValueFromFirst(input, start)
		return FirstTokenClassText
	}
	for {
		next, err := readJSONNonSpace(input)
		if err != nil || next == '}' {
			return FirstTokenClassText
		}
		if next != '"' {
			return FirstTokenClassText
		}
		key, err := readJSONString(input, true)
		if err != nil {
			return FirstTokenClassText
		}
		colon, err := readJSONNonSpace(input)
		if err != nil || colon != ':' {
			return FirstTokenClassText
		}
		valueStart, err := readJSONNonSpace(input)
		if err != nil {
			return FirstTokenClassText
		}
		if key == "type" && valueStart == '"' {
			value, err := readJSONString(input, true)
			if err == nil && strings.EqualFold(strings.TrimSpace(value), "image_generation") {
				return FirstTokenClassImage
			}
		} else if skipJSONValueFromFirst(input, valueStart) != nil {
			return FirstTokenClassText
		}
		separator, err := readJSONNonSpace(input)
		if err != nil || separator == '}' {
			return FirstTokenClassText
		}
		if separator != ',' {
			return FirstTokenClassText
		}
	}
}

func readJSONNonSpace(input *bufio.Reader) (byte, error) {
	for {
		value, err := input.ReadByte()
		if err != nil || !strings.ContainsRune(" \t\r\n", rune(value)) {
			return value, err
		}
	}
}

func readJSONString(input *bufio.Reader, capture bool) (string, error) {
	var value strings.Builder
	escaped := false
	for {
		current, err := input.ReadByte()
		if err != nil {
			return "", err
		}
		if escaped {
			escaped = false
			if capture {
				value.WriteByte(current)
			}
			continue
		}
		if current == '\\' {
			escaped = true
			continue
		}
		if current == '"' {
			return value.String(), nil
		}
		if capture {
			value.WriteByte(current)
		}
	}
}

func skipJSONValue(input *bufio.Reader) error {
	first, err := readJSONNonSpace(input)
	if err != nil {
		return err
	}
	return skipJSONValueFromFirst(input, first)
}

func skipJSONValueFromFirst(input *bufio.Reader, first byte) error {
	// Skipped values are scanned byte-by-byte so large base64 strings are not materialized.
	if first == '"' {
		_, err := readJSONString(input, false)
		return err
	}
	if first != '{' && first != '[' {
		for {
			current, err := input.ReadByte()
			if err != nil {
				return err
			}
			if strings.ContainsRune(" \t\r\n,}]", rune(current)) {
				return input.UnreadByte()
			}
		}
	}
	depth := 1
	inString := false
	escaped := false
	for depth > 0 {
		current, err := input.ReadByte()
		if err != nil {
			return err
		}
		if inString {
			if escaped {
				escaped = false
			} else if current == '\\' {
				escaped = true
			} else if current == '"' {
				inString = false
			}
			continue
		}
		if current == '"' {
			inString = true
			continue
		}
		switch current {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		}
	}
	if inString || escaped {
		return io.ErrUnexpectedEOF
	}
	return nil
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
