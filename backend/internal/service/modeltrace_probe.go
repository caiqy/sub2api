package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/google/uuid"
)

// ResolveModelTraceTarget keeps the selected account's routing identity. Shadow
// credentials are resolved separately, so the parent's proxy cannot replace it.
func (s *AccountTestService) ResolveModelTraceTarget(ctx context.Context, id int64, model string) (*Account, string, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if account == nil || !account.IsOpenAI() || account.IsSyntheticUITest() || !account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions) || !account.IsModelSupported(model) {
		return nil, "", errors.New("account does not support ModelTrace model")
	}
	credential, err := resolveCredentialAccount(ctx, s.accountRepo, account)
	if err != nil {
		return nil, "", errors.New("account credentials are unavailable")
	}
	if _, err := modelTraceBearer(credential); err != nil {
		return nil, "", err
	}
	target := model
	if !account.IsOpenAIPassthroughEnabled() {
		target = normalizeOpenAIModelForUpstream(credential, account.GetMappedModel(model))
	}
	if !slices.Contains(modeltrace.Models(), target) {
		return nil, "", errors.New("mapped target has no ModelTrace fingerprint")
	}
	return account, target, nil
}

func modelTraceBearer(account *Account) (string, error) {
	if account == nil || !account.IsOpenAI() {
		return "", errors.New("unsupported account")
	}
	if account.IsOpenAIAgentIdentity() {
		return "", nil // Signed just before sending, using the existing identity lifecycle.
	}
	var token string
	if account.IsOpenAIOAuthLike() {
		token = account.GetOpenAIAccessToken()
	} else if account.Type == AccountTypeAPIKey {
		token = account.GetOpenAIProtocolAPIKey()
	}
	if strings.TrimSpace(token) == "" {
		return "", errors.New("account credentials are unavailable")
	}
	return token, nil
}

// ProbeModelTrace shares transport and identity helpers with account testing,
// but never retries a challenge or changes scheduling state on 401/429.
func (s *AccountTestService) ProbeModelTrace(ctx context.Context, account *Account, target string, challenge modeltrace.Challenge) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if account == nil || !account.IsOpenAI() || !account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions) || s.httpUpstream == nil || target == "" {
		return "", errors.New("invalid ModelTrace probe configuration")
	}
	credential, err := resolveCredentialAccount(ctx, s.accountRepo, account)
	if err != nil {
		return "", errors.New("account credentials are unavailable")
	}
	token, err := modelTraceBearer(credential)
	if err != nil {
		return "", err
	}
	oauth := credential.IsOpenAIOAuthLike()
	chat := !oauth && !openai_compat.ShouldUseResponsesAPI(account.Extra)
	apiURL := chatgptCodexAPIURL
	if !oauth {
		base := credential.GetOpenAIBaseURL()
		if base == "" {
			base = "https://api.openai.com"
		}
		base, err = s.validateUpstreamBaseURL(base)
		if err != nil {
			return "", errors.New("invalid upstream base URL")
		}
		apiURL = buildOpenAIResponsesURLForPlatform(credential.Platform, base)
		if chat {
			apiURL = buildOpenAIChatCompletionsURL(base)
		}
	}
	payload := map[string]any{"model": target, "stream": true}
	if chat {
		messages := []map[string]string{}
		if challenge.System != "" {
			messages = append(messages, map[string]string{"role": "system", "content": challenge.System})
		}
		payload["messages"] = append(messages, map[string]string{"role": "user", "content": challenge.Prompt})
	} else {
		payload["input"] = []map[string]any{{"role": "user", "content": []map[string]string{{"type": "input_text", "text": challenge.Prompt}}}}
		payload["instructions"] = challenge.System
		payload["store"] = false
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(WithHTTPUpstreamRedirectsDisabled(WithHTTPUpstreamProfile(ctx, HTTPUpstreamProfileOpenAI)), http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", errors.New("cannot create probe request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if credential.IsOpenAIAgentIdentity() {
		headers, err := buildAgentIdentityAuthenticationHeaders(ctx, s.accountRepo, s.agentIdentityWS, &s.agentIdentityTaskMu, credential)
		if err != nil {
			return "", errors.New("cannot authenticate Agent Identity")
		}
		for key, values := range headers {
			req.Header[key] = values
		}
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if oauth {
		req.Host = "chatgpt.com"
		req.Header.Set("OpenAI-Beta", "responses=experimental")
		enforceCodexIdentityHeadersWithUA(req.Header, credential.GetOpenAIUserAgent())
		setOpenAIChatGPTAccountHeaders(req.Header, credential)
		session := uuid.NewString()
		req.Header.Set("Session_ID", session)
		req.Header.Set("Conversation_ID", session)
		if ids := resolveCodexFingerprintIDsFromRequest(account, req.Header); ids != nil {
			applyCodexFingerprintHeaders(req.Header, ids)
		}
	} else {
		applyOpenAICodexProbeHeaders(req.Header)
		applyOpenCodeUpstreamUserAgent(credential, apiURL, req.Header)
	}
	credential.ApplyHeaderOverrides(req.Header)
	applyOpenCodeSessionHeader(nil, credential, apiURL, req.Header, body)
	proxy := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxy = account.Proxy.URL()
	}
	// Context was created before the durable execution-lease check by the runner.
	if err := ctx.Err(); err != nil {
		return "", err
	}
	resp, err := s.doOpenAIAccountTestUpstream(req, proxy, account, true)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", modelTraceHTTPError(resp.StatusCode)
	}
	return readModelTraceOutput(resp.Body, chat, strings.Contains(resp.Header.Get("Content-Type"), "application/json"))
}

type modelTraceHTTPError int

func (e modelTraceHTTPError) Error() string {
	return fmt.Sprintf("upstream HTTP %d (%s)", int(e), http.StatusText(int(e)))
}

type modelTraceResponse struct {
	Type     string              `json:"type"`
	Status   string              `json:"status"`
	Delta    string              `json:"delta"`
	Error    json.RawMessage     `json:"error"`
	Response *modelTraceResponse `json:"response"`
	Output   json.RawMessage     `json:"output"`
	Item     *struct {
		Type   string `json:"type"`
		Status string `json:"status"`
	} `json:"item"`
	OutputIndex *int `json:"output_index"`
	Choices     []struct {
		Index int `json:"index"`
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type modelTraceOutputItem struct {
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func (r *modelTraceResponse) text() string {
	var items []modelTraceOutputItem
	if json.Unmarshal(r.Output, &items) != nil {
		return ""
	}
	var b strings.Builder
	for _, item := range items {
		if item.Type != "message" {
			continue
		}
		var message strings.Builder
		for _, part := range item.Content {
			if part.Type == "output_text" {
				message.WriteString(part.Text)
			}
		}
		if message.Len() > 0 {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(message.String())
		}
	}
	return b.String()
}

// Require the protocol's successful terminal event; enough numeric tokens in a
// truncated, refused, or failed response must never become a valid fingerprint.
func readModelTraceOutput(body io.Reader, chat, unary bool) (string, error) {
	const maxBody = 8 << 20
	limited := &io.LimitedReader{R: body, N: maxBody + 1}
	invalid := errors.New("incomplete or invalid probe response")
	if unary {
		data, err := io.ReadAll(limited)
		if err != nil {
			return "", err
		}
		if len(data) > maxBody {
			return "", invalid
		}
		var r modelTraceResponse
		if err := json.Unmarshal(data, &r); err != nil || (len(r.Error) > 0 && string(r.Error) != "null") {
			return "", invalid
		}
		if chat {
			if len(r.Choices) == 1 && r.Choices[0].FinishReason == "stop" && r.Choices[0].Message.Content != "" && len(r.Choices[0].Message.Content) <= 1<<20 {
				return r.Choices[0].Message.Content, nil
			}
		} else if r.Status == "completed" {
			snapshot := r.text()
			if snapshot != "" && len(snapshot) <= 1<<20 {
				return snapshot, nil
			}
		}
		return "", invalid
	}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var output strings.Builder
	var data []string
	finished := false
	doneItems := newResponsesStreamOutputItems()
	unfinished := make(map[int]bool)
	invalidItems := false
	consume := func() (bool, error) {
		if len(data) == 0 {
			return false, nil
		}
		encoded := strings.Join(data, "\n")
		data = data[:0]
		if encoded == "[DONE]" {
			if !chat || !finished || output.Len() == 0 {
				return false, invalid
			}
			return true, nil
		}
		var r modelTraceResponse
		if json.Unmarshal([]byte(encoded), &r) != nil || (len(r.Error) > 0 && string(r.Error) != "null") {
			return false, invalid
		}
		if chat {
			for _, choice := range r.Choices {
				if choice.Index != 0 {
					return false, invalid
				}
				output.WriteString(choice.Delta.Content)
				if choice.FinishReason != "" {
					if choice.FinishReason != "stop" {
						return false, invalid
					}
					finished = true
				}
			}
		} else {
			switch r.Type {
			case "response.output_item.added":
				if r.OutputIndex == nil || r.Item == nil || r.Item.Type == "" {
					invalidItems = true
				} else if r.Item.Type == "message" {
					unfinished[*r.OutputIndex] = true
				}
			case "response.output_text.delta":
				output.WriteString(r.Delta)
				if r.OutputIndex == nil {
					invalidItems = true
				} else {
					unfinished[*r.OutputIndex] = true
				}
			case "response.output_item.done":
				if r.OutputIndex == nil || r.Item == nil || r.Item.Type == "" ||
					(r.Item.Status != "" && r.Item.Status != "completed") {
					invalidItems = true
				} else {
					if r.Item.Type == "message" {
						delete(unfinished, *r.OutputIndex)
					} else if unfinished[*r.OutputIndex] {
						invalidItems = true
					}
					doneItems.Observe([]byte(encoded))
				}
			case "response.failed", "response.incomplete", "error":
				return false, invalid
			case "response.completed", "response.done":
				if r.Response == nil || r.Response.Status != "completed" || (len(r.Response.Error) > 0 && string(r.Response.Error) != "null") {
					return false, invalid
				}
				terminalOutput := bytes.TrimSpace(r.Response.Output)
				if len(terminalOutput) == 0 || bytes.Equal(terminalOutput, []byte("null")) || bytes.Equal(terminalOutput, []byte("[]")) {
					if invalidItems || len(unfinished) > 0 {
						return false, invalid
					}
					var ok bool
					r.Response.Output, ok = doneItems.BuildOutput()
					if !ok {
						return false, invalid
					}
				}
				snapshot := r.Response.text()
				if snapshot == "" || len(snapshot) > 1<<20 {
					return false, invalid
				}
				output.Reset()
				output.WriteString(snapshot)
				return true, nil
			}
		}
		if output.Len() > 1<<20 {
			return false, invalid
		}
		return false, nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			done, err := consume()
			if err != nil {
				return "", err
			}
			if done {
				return output.String(), nil
			}
		} else if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if limited.N <= 0 {
		return "", invalid
	}
	done, err := consume()
	if err != nil {
		return "", err
	}
	if (done || (chat && finished)) && output.Len() > 0 {
		return output.String(), nil
	}
	return "", invalid
}
