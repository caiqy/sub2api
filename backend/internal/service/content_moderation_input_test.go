package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 当数组末尾不是可审计用户输入时（典型场景：Agent 工具循环结束于 tool/assistant），
// 应直接跳过审计——不再回溯查找历史中的某条用户消息。

func TestExtractContentModerationInput_AnthropicAgentToolLoopSkipsAudit(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"调用一下天气工具"},
			{"role":"assistant","content":[{"type":"tool_use","id":"tool_1","name":"weather","input":{}}]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool_1","content":"晴 25 度"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolAnthropicMessages, body)

	require.Equal(t, "调用一下天气工具", input.Text)
}

func TestExtractContentModerationInput_AnthropicFirstTurnExtractsUser(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"Q1"}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolAnthropicMessages, body)

	require.Equal(t, "Q1", input.Text)
}

func TestExtractContentModerationInput_AnthropicMultiTurnExtractsRecentUniqueUsers(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"Q1"},
			{"role":"assistant","content":"A1"},
			{"role":"user","content":"Q2"},
			{"role":"assistant","content":"A2"},
			{"role":"user","content":"Q2"},
			{"role":"assistant","content":"A3"},
			{"role":"user","content":"Q3"},
			{"role":"assistant","content":"A4"},
			{"role":"user","content":"Q4"},
			{"role":"assistant","content":"A5"},
			{"role":"user","content":"Q5"},
			{"role":"assistant","content":"A6"},
			{"role":"user","content":"Q6"}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolAnthropicMessages, body)

	require.Equal(t, "Q1 A1 Q2 A2 A3 Q3 A4 Q4 A5 Q5 A6 Q6", input.Text)
}

func TestExtractContentModerationInput_AnthropicStreamResendExtractsResend(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"原问题"},
			{"role":"assistant","content":"部分回答……"},
			{"role":"user","content":"重发"}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolAnthropicMessages, body)

	require.Equal(t, "原问题 部分回答…… 重发", input.Text)
}

func TestExtractContentModerationInput_OpenAIChatAgentToolLoopSkipsAudit(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"system","content":"sys"},
			{"role":"user","content":"列出我的订单"},
			{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"orders","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"[]"}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIChat, body)

	require.Equal(t, "列出我的订单", input.Text)
	require.Empty(t, input.Images)
}

func TestExtractContentModerationInput_OpenAIChatMultiTurnExtractsRecentUniqueUsers(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"Q1"},
			{"role":"assistant","content":"A1"},
			{"role":"user","content":"Q2"},
			{"role":"assistant","content":"A2"},
			{"role":"user","content":"Q3"},
			{"role":"assistant","content":"A3"},
			{"role":"user","content":"Q4"},
			{"role":"assistant","content":"A4"},
			{"role":"user","content":"Q5"},
			{"role":"assistant","content":"A5"},
			{"role":"user","content":"Q6"}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIChat, body)

	require.Equal(t, "Q1 A1 Q2 A2 Q3 A3 Q4 A4 Q5 A5 Q6", input.Text)
}

func TestExtractContentModerationInput_GeminiAgentToolLoopSkipsAudit(t *testing.T) {
	body := []byte(`{
		"contents": [
			{"role":"user","parts":[{"text":"查询天气"}]},
			{"role":"model","parts":[{"functionCall":{"name":"weather","args":{}}}]},
			{"role":"user","parts":[{"functionResponse":{"name":"weather","response":{"temp":25}}}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolGemini, body)

	require.Empty(t, input.Text)
	require.Empty(t, input.Images)
}

func TestExtractContentModerationInput_GeminiFirstTurnExtractsUser(t *testing.T) {
	body := []byte(`{
		"contents": [
			{"role":"user","parts":[{"text":"你好"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolGemini, body)

	require.Equal(t, "你好", input.Text)
}

func TestExtractContentModerationInput_GeminiMultiTurnExtractsRecentUniqueUsers(t *testing.T) {
	body := []byte(`{
		"contents": [
			{"role":"user","parts":[{"text":"Q1"}]},
			{"role":"model","parts":[{"text":"A1"}]},
			{"role":"user","parts":[{"text":"Q2"}]},
			{"role":"model","parts":[{"text":"A2"}]},
			{"role":"user","parts":[{"text":"Q3"}]},
			{"role":"model","parts":[{"text":"A3"}]},
			{"role":"user","parts":[{"text":"Q4"}]},
			{"role":"model","parts":[{"text":"A4"}]},
			{"role":"user","parts":[{"text":"Q5"}]},
			{"role":"model","parts":[{"text":"A5"}]},
			{"role":"user","parts":[{"text":"Q6"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolGemini, body)

	require.Equal(t, "Q2 Q3 Q4 Q5 Q6", input.Text)
}

func TestExtractContentModerationInput_ResponsesAgentToolLoopSkipsAudit(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"运行测试"}]},
			{"type":"function_call","call_id":"call_1","name":"run_tests","arguments":"{}"},
			{"type":"function_call_output","call_id":"call_1","output":"all passed"}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

	require.Equal(t, "运行测试", input.Text)
	require.Empty(t, input.Images)
}

func TestExtractContentModerationInput_ResponsesToolOutputWithContentSkipsAudit(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"运行测试"}]},
			{"type":"function_call_output","call_id":"call_1","content":[{"type":"input_text","text":"tool text"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

	require.Equal(t, "运行测试", input.Text)
}

func TestExtractContentModerationInput_ResponsesRolelessUntypedContentSkipsAudit(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"运行测试"}]},
			{"content":[{"type":"input_text","text":"untyped text"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

	require.Equal(t, "运行测试", input.Text)
}

func TestExtractContentModerationInput_ResponsesRecentUniqueUserMessagesExtracted(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"first"}]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"answer"}]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"second"}]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"answer"}]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"third"}]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"answer"}]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"fourth"}]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"answer"}]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"fifth"}]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"answer"}]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"sixth"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

	require.Equal(t, "first answer second third fourth fifth sixth", input.Text)
}

func TestExtractContentModerationInput_ResponsesLastIsAssistantSkipped(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"q1"}]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"a1"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

	require.Equal(t, "q1 a1", input.Text)
}

func TestExtractContentModerationInput_OpenAIChatUserAndAssistantExtracted(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"Q1"},
			{"role":"assistant","content":"A1"},
			{"role":"user","content":"Q2"},
			{"role":"assistant","content":"A2"}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIChat, body)

	require.Equal(t, "Q1 A1 Q2 A2", input.Text)
}

func TestExtractContentModerationInput_OpenAIChatToolCallsAssistantSkipped(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"Q1"},
			{"role":"assistant","content":"safe text"},
			{"role":"assistant","content":"tool msg","tool_calls":[{"id":"call_1"}]},
			{"role":"tool","tool_call_id":"call_1","content":"result"},
			{"role":"user","content":"Q2"}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIChat, body)

	require.Equal(t, "Q1 safe text Q2", input.Text)
}

func TestExtractContentModerationInput_OpenAIChatToolAtEndStillAudits(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"Q1"},
			{"role":"assistant","content":null,"tool_calls":[{"id":"call_1"}]},
			{"role":"tool","tool_call_id":"call_1","content":"result"}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIChat, body)

	require.Equal(t, "Q1", input.Text)
}

func TestExtractContentModerationInput_AnthropicUserAndAssistantExtracted(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"Q1"},
			{"role":"assistant","content":"A1"},
			{"role":"user","content":"Q2"},
			{"role":"assistant","content":[{"type":"text","text":"A2"}]}
		]
	}`)
	input := ExtractContentModerationInput(ContentModerationProtocolAnthropicMessages, body)
	require.Equal(t, "Q1 A1 Q2 A2", input.Text)
}

func TestExtractContentModerationInput_AnthropicToolUseAndResultSkipped(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role":"user","content":"Q1"},
			{"role":"assistant","content":[{"type":"tool_use","id":"tool_1","name":"x","input":{}}]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool_1","content":"result"}]},
			{"role":"assistant","content":[{"type":"text","text":"A2"}]}
		]
	}`)
	input := ExtractContentModerationInput(ContentModerationProtocolAnthropicMessages, body)
	require.Equal(t, "Q1 A2", input.Text)
}

func TestExtractContentModerationInput_ResponsesUserAndAssistantExtracted(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"Q1"}]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"A1"}]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"Q2"}]}
		]
	}`)
	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)
	require.Equal(t, "Q1 A1 Q2", input.Text)
}

func TestExtractContentModerationInput_ResponsesFunctionCallItemSkipped(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"Q1"}]},
			{"type":"function_call","call_id":"c1","name":"tool","arguments":"{}"},
			{"type":"function_call_output","call_id":"c1","output":"result"},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"A1"}]}
		]
	}`)
	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)
	require.Equal(t, "Q1 A1", input.Text)
}
