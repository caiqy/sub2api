//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestEstimateVisibleInputContext(t *testing.T) {
	base := `{"model":"gpt-4o","messages":[{"role":"user","content":"hello 世界"}]}`
	a, err := EstimateVisibleInputContext("gpt-4o", []byte(base))
	require.NoError(t, err)
	require.Positive(t, a.Tokens)
	b, err := EstimateVisibleInputContext("gpt-4o", []byte(strings.TrimSuffix(base, "}")+`,"max_output_tokens":999999,"cached_tokens":999999}`))
	require.NoError(t, err)
	require.Equal(t, a, b)
	for _, body := range []string{
		`{"tools":[{"type":"function","function":{"parameters":{"properties":{"very_long_business_property_name":{"type":"string"}}}}}]}`,
		`{"messages":[{"content":[{"type":"tool_use","input":{"data":"hello","id":"world","type":"important"}}]}]}`,
		`{"contents":[{"parts":[{"functionCall":{"args":{"data":"hello","id":"world","type":"important"}}}]}]}`,
	} {
		n, e := EstimateVisibleInputContext("gpt-4o", []byte(body))
		require.NoError(t, e)
		require.Greater(t, n.Tokens, 3, body)
	}
	for _, body := range []string{`{"previous_response_id":"resp_a","input":"hello"}`, `{"input":[{"type":"input_image","image_url":"data:image/png;base64,` + strings.Repeat("AAAA", 10000) + `"}]}`, `{"contents":[{"parts":[{"inlineData":{"data":"AAAA","mimeType":"image/png"}}]}]}`} {
		n, e := EstimateVisibleInputContext("gpt-4o", []byte(body))
		require.NoError(t, e)
		require.Equal(t, "partial", n.Coverage)
		require.Less(t, n.Tokens, 20)
	}
	_, err = EstimateVisibleInputContext("gpt-4o", []byte(`{broken`))
	require.Error(t, err)
}
func TestInputContextTypedChatConversion(t *testing.T) {
	payload := strings.Repeat("long payload ", 500)
	quoted, err := json.Marshal(payload)
	require.NoError(t, err)
	for _, extra := range []string{`,"Instructions":` + string(quoted), `,"Inſtructions":` + string(quoted), `,"İnstructions":` + string(quoted), `,"tools":null`} {
		t.Run(extra[:12], func(t *testing.T) {
			body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]` + extra + `}`)
			var req apicompat.ChatCompletionsRequest
			require.NoError(t, json.Unmarshal(body, &req))
			converted, err := apicompat.ChatCompletionsToResponses(&req)
			require.NoError(t, err)
			wire, err := json.Marshal(converted)
			require.NoError(t, err)
			want, err := EstimateVisibleInputContext(req.Model, wire)
			require.NoError(t, err)
			got, err := EstimateVisibleInputContext(req.Model, body)
			require.NoError(t, err)
			require.Equal(t, want.Tokens, got.Tokens)
			if req.Instructions != "" {
				s := &SettingService{settingRepo: newWritableSettingRepoStub(nil)}
				s.inputContextLimitCache.Store(&cachedInputContextLimit{settings: InputContextLimitSettings{Enabled: true, GroupLimits: map[string]int32{"1": 10}}, expires: time.Now().Add(time.Hour)})
				group := int64(1)
				_, decision, err := s.CheckInputContextLimit(context.Background(), &group, req.Model, body)
				require.NoError(t, err)
				require.NotNil(t, decision)
			}
		})
	}
}

func TestInputContextTypedNestedConversion(t *testing.T) {
	payload, err := json.Marshal(strings.Repeat("long payload ", 500))
	require.NoError(t, err)
	for _, fields := range []string{
		`"Messages":[{"Role":"user","Content":[{"Type":"text","Text":` + string(payload) + `}]}]`,
		`"Messages":[{"Role":"assistant","Tool_Calls":[{"Type":"function","Function":{"Name":"lookup","Arguments":` + string(payload) + `}}]}]`,
		`"messages":[{"role":"user","content":"hi"}],"Tools":[{"Type":"function","Function":{"Name":"lookup","Parameters":{"Properties":{"Data":{"Description":` + string(payload) + `}}}}}]`,
	} {
		body := []byte(`{"model":"gpt-4o",` + fields + `}`)
		var req apicompat.ChatCompletionsRequest
		require.NoError(t, json.Unmarshal(body, &req))
		out, err := apicompat.ChatCompletionsToResponses(&req)
		require.NoError(t, err)
		wire, err := json.Marshal(out)
		require.NoError(t, err)
		for _, input := range [][]byte{body, wire} {
			n, err := EstimateVisibleInputContext(req.Model, input)
			require.NoError(t, err)
			require.Greater(t, n.Tokens, 900)
		}
	}
}

func TestEstimateVisibleCaseAndDuplicates(t *testing.T) {
	long := strings.Repeat("long payload ", 500)
	q, err := json.Marshal(long)
	require.NoError(t, err)
	for _, body := range []string{
		`{"instructions":"hi","Instructions":` + string(q) + `}`,
		`{"Instructions":` + string(q) + `,"instructions":"hi"}`,
		`{"instructions":` + string(q) + `,"instructions":"hi"}`,
		`{"instructions":"hi","instructions":` + string(q) + `}`,
		`{"instructions":` + string(q) + `,"Instructions":null}`,
		`{"Messages":[{"Content":` + string(q) + `,"content":"hi"}]}`,
		`{"Messages":[{"content":"hi","Content":` + string(q) + `}]}`,
		`{"Text":{"Format":{"Schema":{"Data":` + string(q) + `}},"format":null}}`,
		`{"MESSAGES":[{"Role":"user","Content":[{"Type":"text","Text":` + string(q) + `}]}]}`,
		`{"Input":[{"Type":"input_text","Text":` + string(q) + `}]}`,
		`{"Meſſageſ":[{"Content":[{"Type":"tool_use","Input":{"ſignature":` + string(q) + `}}]}]}`,
		`{"Messages":[{"Content":[{"Input_ſchema":{"ſignature":` + string(q) + `}}]}]}`,
		`{"Messages":[{"Content":[{"Sıgnature":` + string(q) + `}]}]}`,
		`{"Messages":[{"Content":[{"Sİgnature":` + string(q) + `}]}]}`,
		`{"Input":[{"Type":"function_call","Arguments":` + string(q) + `}]}`,
		`{"Messages":[{"Content":[{"Type":"tool_use","Input":{"Data":` + string(q) + `,"Signature":"business","Content":{}}}]}]}`,
		`{"Contents":[{"Parts":[{"FunctionCall":{"Args":{"Data":` + string(q) + `}}}]}]}`,
		`{"Tools":[{"Type":"function","Function":{"Parameters":{"Data":` + string(q) + `}}}]}`,
		`{"Text":{"Format":{"Schema":{"Properties":{"Signature":{"Description":` + string(q) + `}}}}}}`,
		`{"GENERATIONCONFIG":{"RESPONSEJSONSCHEMA":{"Data":` + string(q) + `}}}`,
	} {
		n, err := EstimateVisibleInputContext("gpt-4o", []byte(body))
		require.NoError(t, err, body)
		require.Greater(t, n.Tokens, 900, body)
		require.Equal(t, "visible", n.Coverage, body)
	}
	for _, body := range []string{
		`{"Input":[{"Type":"input_image","Image_URL":` + string(q) + `}]}`,
		`{"Messages":[{"Content":[{"Type":"thinking","Thinking":"hi","Signature":` + string(q) + `}]}]}`,
		`{"Contents":[{"Parts":[{"InlineData":{"Data":` + string(q) + `}}]}]}`,
	} {
		n, err := EstimateVisibleInputContext("gpt-4o", []byte(body))
		require.NoError(t, err)
		require.Less(t, n.Tokens, 10)
		require.Equal(t, "partial", n.Coverage)
	}
	for _, types := range []string{`"type":"input_image","Type":"tool_use"`, `"Type":"tool_use","type":"input_image"`, `"type":"input_image","type":"tool_use"`, `"type":"tool_use","type":"input_image"`} {
		n, err := EstimateVisibleInputContext("gpt-4o", []byte(`{"input":[{`+types+`,"Input":{"Data":`+string(q)+`}}]}`))
		require.NoError(t, err)
		require.Greater(t, n.Tokens, 900, types)
		require.Equal(t, "partial", n.Coverage)
	}
	for _, field := range []string{"messages", "system", "input", "instructions", "tools", "functions", "contents"} {
		_, err := EstimateVisibleInputContext("gpt-4o", []byte(`{"`+field+`":null}`))
		require.NoError(t, err, field)
		_, err = EstimateVisibleInputContext("gpt-4o", []byte(`{"`+field+`":42}`))
		require.Error(t, err, field)
	}
}

func TestInputContextSettingsAndSnapshots(t *testing.T) {
	ctx := context.Background()
	repo := newWritableSettingRepoStub(nil)
	s := &SettingService{settingRepo: repo, defaultSubGroupReader: &defaultSubGroupReaderStub{byID: map[int64]*Group{1: {ID: 1}, 2: {ID: 2}}}}
	_, d, e := s.CheckInputContextLimit(ctx, nil, "", []byte(`bad json`))
	require.NoError(t, e)
	require.Nil(t, d)
	config := &InputContextLimitSettings{Enabled: true, GroupLimits: map[string]int32{"1": 1, "2": 1000}}
	require.NoError(t, s.SetInputContextLimitSettings(ctx, config))
	config.GroupLimits["1"] = 500
	original, effective := int64(1), int64(2)
	ctx = WithOriginalInputContextGroup(ctx, &original)
	frozen, d, e := s.CheckInputContextLimit(ctx, &effective, "gpt-4o", []byte(`{"input":"hello world more text"}`))
	require.NoError(t, e)
	require.NotNil(t, d)
	require.EqualValues(t, 1, d.Limit)
	require.NoError(t, s.SetInputContextLimitSettings(ctx, &InputContextLimitSettings{Enabled: false}))
	_, d, e = s.CheckInputContextLimit(frozen, &effective, "gpt-4o", []byte(`{"input":"hello world more text"}`))
	require.NoError(t, e)
	require.NotNil(t, d)
	_, d, e = s.CheckInputContextLimit(FreshInputContextLimitTurn(frozen), &effective, "", []byte(`bad`))
	require.NoError(t, e)
	require.Nil(t, d)
	require.Error(t, s.SetInputContextLimitSettings(ctx, &InputContextLimitSettings{GroupLimits: map[string]int32{"3": 1}}))
	for _, raw := range []string{`{"group_limits":{"1":-1}}`, `{"group_limits":{"1":1.5}}`, `{"group_limits":{"1":2147483648}}`} {
		v := InputContextLimitSettings{}
		err := json.Unmarshal([]byte(raw), &v)
		if err == nil {
			err = s.SetInputContextLimitSettings(ctx, &v)
		}
		require.Error(t, err)
	}
	repo.setErr = errors.New("save failed")
	require.Error(t, s.SetInputContextLimitSettings(ctx, config))
	require.False(t, s.inputContextLimitSnapshot(ctx).settings.Enabled)
	prior := s.inputContextLimitSnapshot(ctx)
	s.inputContextLimitCache.Store(&cachedInputContextLimit{settings: prior.settings, expires: time.Now().Add(-time.Second), version: prior.version})
	repo.err = errors.New("db down")
	require.False(t, s.inputContextLimitSnapshot(ctx).settings.Enabled)
}

type inputContextRefreshBarrier struct {
	*writableSettingRepoStub
	started chan struct{}
	release chan struct{}
}

func (r *inputContextRefreshBarrier) GetValue(ctx context.Context, key string) (string, error) {
	value, err := r.writableSettingRepoStub.GetValue(ctx, key)
	if key == SettingKeyInputContextLimit {
		close(r.started)
		<-r.release
	}
	return value, err
}
func TestInputContextRefreshCannotOverwriteSave(t *testing.T) {
	ctx := context.Background()
	repo := &inputContextRefreshBarrier{newWritableSettingRepoStub(map[string]string{SettingKeyInputContextLimit: `{"enabled":true,"group_limits":{}}`}), make(chan struct{}), make(chan struct{})}
	s := &SettingService{settingRepo: repo}
	refreshed := make(chan struct{})
	go func() { s.inputContextLimitSnapshot(ctx); close(refreshed) }()
	<-repo.started
	saved := make(chan error, 1)
	go func() { saved <- s.SetInputContextLimitSettings(ctx, &InputContextLimitSettings{Enabled: false}) }()
	close(repo.release)
	<-refreshed
	require.NoError(t, <-saved)
	require.False(t, s.inputContextLimitSnapshot(ctx).settings.Enabled)
}
func TestInputContextBoundaryAndOriginalGroup(t *testing.T) {
	ctx := context.Background()
	body := []byte(`{"input":"hello world"}`)
	estimate, err := EstimateVisibleInputContext("gpt-4o", body)
	require.NoError(t, err)
	s := &SettingService{settingRepo: newWritableSettingRepoStub(nil)}
	original, effective := int64(1), int64(2)
	for _, limit := range []int32{int32(estimate.Tokens), int32(estimate.Tokens - 1)} {
		s.inputContextLimitCache.Store(&cachedInputContextLimit{settings: InputContextLimitSettings{Enabled: true, GroupLimits: map[string]int32{"1": limit, "2": 1000}}, expires: time.Now().Add(time.Hour)})
		frozen := WithOriginalInputContextGroup(ctx, &original)
		frozen = WithOriginalInputContextGroup(frozen, &effective)
		_, d, e := s.CheckInputContextLimit(frozen, &effective, "gpt-4o", body)
		require.NoError(t, e)
		if int(limit) == estimate.Tokens {
			require.Nil(t, d)
		} else {
			require.NotNil(t, d)
			require.Equal(t, limit, d.Limit)
		}
	}
}

func TestInputContextStructuredOutputSchemas(t *testing.T) {
	paths := [][]string{{"output_format"}, {"output_config", "format"}}
	for _, config := range []string{"generationConfig", "generation_config"} {
		for _, schema := range []string{"responseSchema", "response_schema", "responseJsonSchema", "response_json_schema"} {
			paths = append(paths, []string{config, schema})
		}
	}
	for _, path := range paths {
		t.Run(strings.Join(path, "."), func(t *testing.T) {
			makeBody := func(description string) []byte {
				var value any = map[string]any{"type": "object", "properties": map[string]any{"business_property": map[string]any{"type": "string", "description": description}}}
				if path[0] == "output_format" || path[0] == "output_config" {
					value = map[string]any{"type": "json_schema", "schema": value}
				}
				for i := len(path) - 1; i >= 0; i-- {
					value = map[string]any{path[i]: value}
				}
				root := value.(map[string]any)
				root["input"] = "hello"
				body, err := json.Marshal(root)
				require.NoError(t, err)
				return body
			}
			short, err := EstimateVisibleInputContext("gpt-4o", makeBody("brief"))
			require.NoError(t, err)
			body := makeBody(strings.Repeat("business schema description ", 1000))
			long, err := EstimateVisibleInputContext("gpt-4o", body)
			require.NoError(t, err)
			require.Greater(t, long.Tokens, short.Tokens+1000)
			s := &SettingService{settingRepo: newWritableSettingRepoStub(nil)}
			group := int64(1)
			for _, limit := range []int32{int32(long.Tokens), int32(long.Tokens - 1)} {
				s.inputContextLimitCache.Store(&cachedInputContextLimit{settings: InputContextLimitSettings{Enabled: true, GroupLimits: map[string]int32{"1": limit}}, expires: time.Now().Add(time.Hour)})
				_, decision, err := s.CheckInputContextLimit(context.Background(), &group, "gpt-4o", body)
				require.NoError(t, err)
				require.Equal(t, int(limit) < long.Tokens, decision != nil)
			}
		})
	}
	base, err := EstimateVisibleInputContext("gpt-4o", []byte(`{"input":"hello"}`))
	require.NoError(t, err)
	knobs, err := EstimateVisibleInputContext("gpt-4o", []byte(`{"input":"hello","generationConfig":{"maxOutputTokens":2147483647,"temperature":1},"output_config":{"effort":"high"}}`))
	require.NoError(t, err)
	require.Equal(t, base, knobs)
}

func TestEstimateVisibleSchemaAndSignature(t *testing.T) {
	for _, body := range []string{`{"response_format":{"type":"json_schema","json_schema":{"schema":{"properties":{"long_business_property":{"type":"string"}}}}}}`, `{"text":{"format":{"type":"json_schema","schema":{"properties":{"long_business_property":{"type":"string"}}}}}}`} {
		n, e := EstimateVisibleInputContext("gpt-4o", []byte(body))
		require.NoError(t, e)
		require.Greater(t, n.Tokens, 10)
	}
	n, e := EstimateVisibleInputContext("claude-sonnet", []byte(`{"messages":[{"content":[{"type":"thinking","thinking":"hello","signature":"`+strings.Repeat("opaque", 10000)+`"}]}]}`))
	require.NoError(t, e)
	require.Equal(t, "partial", n.Coverage)
	require.Less(t, n.Tokens, 10)
}
