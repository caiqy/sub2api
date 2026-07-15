package openai

import (
	"bufio"
	"io"
	"strconv"
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
	class, _ := ResponsesFirstTokenRequestReader(reader)
	return class
}

func ResponsesFirstTokenRequestReader(reader io.Reader) (FirstTokenClass, bool) {
	if reader == nil {
		return FirstTokenClassText, false
	}
	input := bufio.NewReader(reader)
	start, err := readJSONNonSpace(input)
	if err != nil || start != '{' {
		return FirstTokenClassText, false
	}
	class := FirstTokenClassText
	stream := false
	for {
		next, err := readJSONNonSpace(input)
		if err != nil || next == '}' {
			return class, stream
		}
		if next != '"' {
			return class, stream
		}
		key, err := readJSONString(input, true)
		if err != nil {
			return class, stream
		}
		colon, err := readJSONNonSpace(input)
		if err != nil || colon != ':' {
			return class, stream
		}
		switch key {
		case "tool_choice":
			class = readToolChoiceClass(input)
		case "stream":
			valueStart, valueErr := readJSONNonSpace(input)
			if valueErr != nil {
				return class, stream
			}
			stream = readJSONBool(input, valueStart)
		default:
			if skipJSONValue(input) != nil {
				return class, stream
			}
		}
		separator, err := readJSONNonSpace(input)
		if err != nil || separator == '}' {
			return class, stream
		}
		if separator != ',' {
			return class, stream
		}
	}
}

func readToolChoiceClass(input *bufio.Reader) FirstTokenClass {
	class := FirstTokenClassText
	start, err := readJSONNonSpace(input)
	if err != nil {
		return class
	}
	if start != '{' {
		_ = skipJSONValueFromFirst(input, start)
		return FirstTokenClassText
	}
	for {
		next, err := readJSONNonSpace(input)
		if err != nil || next == '}' {
			return class
		}
		if next != '"' {
			return class
		}
		key, err := readJSONString(input, true)
		if err != nil {
			return class
		}
		colon, err := readJSONNonSpace(input)
		if err != nil || colon != ':' {
			return class
		}
		valueStart, err := readJSONNonSpace(input)
		if err != nil {
			return class
		}
		if key == "type" && valueStart == '"' {
			value, err := readJSONString(input, true)
			if err == nil && strings.EqualFold(strings.TrimSpace(value), "image_generation") {
				class = FirstTokenClassImage
			}
		} else if skipJSONValueFromFirst(input, valueStart) != nil {
			return class
		}
		separator, err := readJSONNonSpace(input)
		if err != nil || separator == '}' {
			return class
		}
		if separator != ',' {
			return class
		}
	}
}

func readJSONBool(input *bufio.Reader, first byte) bool {
	want := ""
	if first == 't' {
		want = "rue"
	} else if first == 'f' {
		want = "alse"
	} else {
		_ = skipJSONValueFromFirst(input, first)
		return false
	}
	for i := range len(want) {
		current, err := input.ReadByte()
		if err != nil || current != want[i] {
			return false
		}
	}
	return first == 't'
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
				_ = value.WriteByte(current)
			}
			continue
		}
		if current == '\\' {
			escaped = true
			if capture {
				_ = value.WriteByte(current)
			}
			continue
		}
		if current == '"' {
			if !capture {
				return "", nil
			}
			return strconv.Unquote(`"` + value.String() + `"`)
		}
		if capture {
			_ = value.WriteByte(current)
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
	return responsesEventIsTerminal(eventType) || ResponsesEventRecordsFirstToken(payload)
}

func ResponsesEventRecordsFirstToken(payload []byte) bool {
	eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	if eventType == "" || responsesEventIsTerminal(eventType) {
		return false
	}
	if strings.HasSuffix(eventType, ".delta") {
		return strings.HasPrefix(eventType, "response.")
	}
	for _, prefix := range []string{
		"response.output_",
		"response.content_part.",
		"response.reasoning_",
		"response.function_call_arguments.",
		"response.image_generation_call.",
		"response.code_interpreter_call.",
		"response.file_search_call.",
		"response.web_search_call.",
		"response.mcp_call.",
	} {
		if strings.HasPrefix(eventType, prefix) {
			return true
		}
	}
	return false
}

func responsesEventIsTerminal(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "error", "response.completed", "response.done", "response.failed", "response.incomplete", "response.canceled", "response.cancelled":
		return true
	default:
		return false
	}
}
