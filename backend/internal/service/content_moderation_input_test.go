package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractContentModerationInput_AnthropicAgentToolLoopFiltersTools(t *testing.T) {
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

func TestExtractContentModerationInput_AnthropicMultiTurnExtractsAllAuditableContent(t *testing.T) {
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

func TestExtractContentModerationInput_OpenAIChatAgentToolLoopFiltersTools(t *testing.T) {
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

func TestExtractContentModerationInput_OpenAIChatMultiTurnExtractsAllAuditableContent(t *testing.T) {
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

func TestExtractContentModerationInput_GeminiAgentToolLoopFiltersTools(t *testing.T) {
	body := []byte(`{
		"contents": [
			{"role":"user","parts":[{"text":"查询天气"}]},
			{"role":"model","parts":[{"functionCall":{"name":"weather","args":{}}}]},
			{"role":"user","parts":[{"functionResponse":{"name":"weather","response":{"temp":25}}}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolGemini, body)

	require.Equal(t, "查询天气", input.Text)
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

func TestExtractContentModerationInput_GeminiMultiTurnExtractsAllAuditableContent(t *testing.T) {
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

	require.Equal(t, "Q1 A1 Q2 A2 Q3 A3 Q4 A4 Q5 A5 Q6", input.Text)
}

func TestExtractContentModerationInput_ResponsesAgentToolLoopFiltersTools(t *testing.T) {
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

func TestExtractContentModerationInput_ResponsesToolOutputItemFiltered(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"运行测试"}]},
			{"type":"function_call_output","call_id":"call_1","content":[{"type":"input_text","text":"tool text"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

	require.Equal(t, "运行测试", input.Text)
}

func TestExtractContentModerationInput_ResponsesAllAuditableContentExtracted(t *testing.T) {
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

func TestExtractContentModerationInput_ResponsesUserAndAssistantBothExtracted(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"q1"}]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"a1"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

	require.Equal(t, "q1 a1", input.Text)
}

func TestExtractContentModerationInput_ResponsesAssistantWithoutTypeExtracted(t *testing.T) {
	body := []byte(`{
		"input":[
			{"role":"user","content":[{"type":"input_text","text":"q1"}]},
			{"role":"assistant","content":[{"type":"output_text","text":"assistant answer"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

	require.Equal(t, "q1 assistant answer", input.Text)
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

func TestExtractContentModerationInput_GeminiUserAndModelExtracted(t *testing.T) {
	body := []byte(`{
		"contents": [
			{"role":"user","parts":[{"text":"Q1"}]},
			{"role":"model","parts":[{"text":"A1"}]},
			{"role":"user","parts":[{"text":"Q2"}]}
		]
	}`)
	input := ExtractContentModerationInput(ContentModerationProtocolGemini, body)
	require.Equal(t, "Q1 A1 Q2", input.Text)
}

func TestExtractContentModerationInput_GeminiFunctionCallPartSkipped(t *testing.T) {
	body := []byte(`{
		"contents": [
			{"role":"user","parts":[{"text":"Q1"}]},
			{"role":"model","parts":[{"functionCall":{"name":"x","args":{}}}]},
			{"role":"user","parts":[{"functionResponse":{"name":"x","response":{"r":1}}}]},
			{"role":"model","parts":[{"text":"A1"}]}
		]
	}`)
	input := ExtractContentModerationInput(ContentModerationProtocolGemini, body)
	require.Equal(t, "Q1 A1", input.Text)
}

func TestExtractContentModerationInput_ResponsesRolelessTypedMessageExtracted(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"message","role":"","content":[{"type":"input_text","text":"roleless msg"}]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"user msg"}]}
		]
	}`)
	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)
	require.Equal(t, "roleless msg user msg", input.Text)
}

func TestExtractContentModerationInput_ResponsesTranscriptItemsExtracted(t *testing.T) {
	body := []byte(`{
		"input":[
			{"role":"developer","content":"developer secret"},
			{"role":"user","content":[{"type":"input_text","text":"user prompt"}]},
			{"type":"reasoning","encrypted_content":"gAAAA"},
			{"role":"assistant","content":[{"type":"output_text","text":"assistant answer"}]},
			{"type":"function_call_output","content":[{"type":"input_text","text":"tool result"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

	require.Equal(t, "user prompt assistant answer", input.Text)
}

func TestExtractContentModerationInput_ResponsesCodexSkipsInstructionsAndDeveloperContext(t *testing.T) {
	body := []byte(`{
		"model":"gpt-5.5",
		"instructions":"You are Codex, a coding agent based on GPT-5.",
		"input":[
			{"type":"message","role":"developer","content":[
				{"type":"input_text","text":"<permissions instructions> Filesystem sandboxing"},
				{"type":"input_text","text":"<app-context> Codex desktop context"}
			]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"用户真实问题"}]}
		]
	}`)

	input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

	require.Equal(t, "用户真实问题", input.Text)
}
