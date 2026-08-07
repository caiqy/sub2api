package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	chatgptCodexAlphaSearchURL   = "https://chatgpt.com/backend-api/codex/alpha/search"
	openAIPlatformAlphaSearchURL = "https://api.openai.com/v1/alpha/search"
)

// ForwardAlphaSearch proxies Codex standalone web search without binding the
// evolving alpha request or response schema.
//
// 返回值约定：仅当上游返回 2xx（一次真实成功的搜索）时返回非 nil 的
// *OpenAIForwardResult（WebSearchCalls=1，供按次计费）；上游错误被原样透传
// 给客户端时返回 (nil, nil)，不产生计费。
func (s *OpenAIGatewayService) ForwardAlphaSearch(ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
	if s == nil || c == nil || account == nil {
		return nil, fmt.Errorf("service, context, and account are required")
	}
	var err error
	body, err = openAIRequestBodyBytes(c, body)
	if err != nil {
		return nil, fmt.Errorf("read alpha search request body: %w", err)
	}
	sourceBody := body
	modelResult := gjson.GetBytes(body, "model")
	requestedModel := strings.Clone(strings.TrimSpace(modelResult.String()))
	if modelResult.Type != gjson.String || requestedModel == "" {
		return nil, fmt.Errorf("model is required")
	}

	upstreamModel := normalizeOpenAIModelForUpstream(account, account.GetMappedModel(requestedModel))
	if upstreamModel != "" && upstreamModel != requestedModel {
		body = ReplaceModelInBody(body, upstreamModel)
	}
	bodyHandle, bodyHandleOwned, err := openAIRequestBodyHandleForContext(c, body)
	if err != nil {
		return nil, fmt.Errorf("create alpha search request body: %w", err)
	}
	if bodyHandleOwned {
		defer CleanupRequestBodyHandle(bodyHandle)
	}

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	if err := s.ensureOpenAIAlphaSearchAuthMetadata(ctx, account, token, proxyURL); err != nil {
		return nil, err
	}
	if account.IsOpenAIPersonalAccessToken() {
		return s.forwardAlphaSearchViaResponsesWebSearch(ctx, c, account, bodyHandle, token, proxyURL, requestedModel, upstreamModel)
	}

	req, err := s.buildOpenAIAlphaSearchRequest(ctx, c, account, sourceBody, body, bodyHandle, token)
	if err != nil {
		return nil, err
	}

	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	closeOpenAIRequestBody(req)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, s.handleOpenAIUpstreamTransportError(ctx, c, account, err, true)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, fmt.Errorf("read alpha search response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		upstreamMessage := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMessage, respBody) ||
			isOpenAIAlphaSearchEndpointUnsupported(account, resp.StatusCode) {
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			// alpha/search 是独立的工具端点，单次 401 不能证明账号的模型调用
			// 凭据全局失效。若沿用通用 401 逻辑，PAT 会因没有 refresh_token
			// 被永久标记为 error；历史导入且缺少 auth_mode 标记的 at- token 也会
			// 漏过 PAT 类型判断。这里仍允许本次请求换号，但不修改任何账号状态；
			// 真正的凭据失效由普通 Responses 请求或 whoami 校验判定。
			shouldDisable := false
			if shouldApplyOpenAIAlphaSearchAccountErrorSideEffects(resp.StatusCode) {
				shouldDisable = s.handleFailoverSideEffects(ctx, resp, account, respBody, upstreamModel)
			}
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: !shouldDisable && account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
			}
		}
	}

	if !account.IsShadow() {
		s.UpdateCodexUsageSnapshotFromHeaders(ctx, account.ID, resp.Header)
	}
	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, respBody)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		// 非 2xx（错误/重定向）已原样透传给客户端：不是一次成功的搜索，不计费。
		return nil, nil
	}
	return &OpenAIForwardResult{
		RequestID:      strings.TrimSpace(resp.Header.Get("x-request-id")),
		Model:          requestedModel,
		UpstreamModel:  upstreamModel,
		Duration:       time.Since(upstreamStart),
		WebSearchCalls: 1,
	}, nil
}

func (s *OpenAIGatewayService) forwardAlphaSearchViaResponsesWebSearch(ctx context.Context, c *gin.Context, account *Account, alphaHandle *RequestBodyHandle, token string, proxyURL string, requestedModel string, upstreamModel string) (*OpenAIForwardResult, error) {
	if upstreamModel == "" {
		upstreamModel = requestedModel
	}
	alphaBody, err := alphaHandle.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read alpha search fallback body: %w", err)
	}
	sessionID := strings.Clone(strings.TrimSpace(gjson.GetBytes(alphaBody, "id").String()))
	responsesBody, err := buildOpenAIAlphaSearchResponsesWebSearchBody(alphaBody, upstreamModel)
	if err != nil {
		return nil, err
	}
	responsesHandle, err := NewRequestBodyHandleFromBytes(responsesBody, openAIRequestBodyHandleOptions())
	if err != nil {
		return nil, fmt.Errorf("create alpha search fallback body: %w", err)
	}
	defer CleanupRequestBodyHandle(responsesHandle)
	req, err := s.buildOpenAIAlphaSearchResponsesWebSearchRequest(ctx, c, account, responsesHandle, token, sessionID)
	if err != nil {
		return nil, err
	}
	SetActualOpenAIUpstreamEndpoint(c, "/v1/responses")

	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	closeOpenAIRequestBody(req)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, s.handleOpenAIUpstreamTransportError(ctx, c, account, err, true)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, fmt.Errorf("read alpha search responses fallback response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		upstreamMessage := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMessage, respBody) {
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			// 仍按 alpha/search 工具请求处理：PAT 的工具链路失败不能直接永久置错。
			shouldDisable := false
			if shouldApplyOpenAIAlphaSearchAccountErrorSideEffects(resp.StatusCode) {
				shouldDisable = s.handleFailoverSideEffects(ctx, resp, account, respBody, upstreamModel)
			}
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: !shouldDisable && account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
			}
		}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/json"
		}
		c.Data(resp.StatusCode, contentType, respBody)
		return nil, nil
	}
	if !account.IsShadow() {
		s.UpdateCodexUsageSnapshotFromHeaders(ctx, account.ID, resp.Header)
	}
	alphaRespBody, err := openAIAlphaSearchResponseFromResponsesSSE(respBody)
	if err != nil {
		return nil, err
	}
	c.Data(http.StatusOK, "application/json", alphaRespBody)
	return &OpenAIForwardResult{
		RequestID:        strings.TrimSpace(resp.Header.Get("x-request-id")),
		Model:            requestedModel,
		UpstreamModel:    upstreamModel,
		UpstreamEndpoint: "/v1/responses",
		ResponseHeaders:  resp.Header.Clone(),
		Duration:         time.Since(upstreamStart),
		WebSearchCalls:   1,
	}, nil
}

func (s *OpenAIGatewayService) buildOpenAIAlphaSearchResponsesWebSearchRequest(ctx context.Context, c *gin.Context, account *Account, bodyHandle *RequestBodyHandle, token string, sessionID string) (*http.Request, error) {
	req, err := openAINewRequestWithBodyHandle(ctx, http.MethodPost, chatgptCodexURL, bodyHandle, false)
	if err != nil {
		return nil, err
	}
	cleanupOnError := true
	defer func() {
		if cleanupOnError {
			closeOpenAIRequestBody(req)
		}
	}()
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	authHeaders, err := s.buildOpenAIAuthenticationHeaders(ctx, account, token)
	if err != nil {
		return nil, fmt.Errorf("build openai authentication headers: %w", err)
	}
	for key, values := range authHeaders {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Host = "chatgpt.com"
	if err := resolveAndSetOpenAIChatGPTAccountHeaders(ctx, s.accountRepo, req.Header, account); err != nil {
		return nil, fmt.Errorf("resolve chatgpt account headers: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	if turnMetadata := openAIAlphaSearchInboundHeader(c, "X-Codex-Turn-Metadata"); turnMetadata != "" {
		req.Header.Set("X-Codex-Turn-Metadata", turnMetadata)
	}
	if version := openAIAlphaSearchInboundHeader(c, "Version"); version != "" {
		req.Header.Set("Version", version)
	} else {
		req.Header.Set("Version", codexCLIVersion)
	}
	if originator := openAIAlphaSearchInboundHeader(c, "Originator"); originator != "" {
		req.Header.Set("Originator", originator)
	} else {
		req.Header.Set("Originator", "codex_cli_rs")
	}
	if userAgent := openAIAlphaSearchInboundHeader(c, "User-Agent"); userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else {
		req.Header.Set("User-Agent", codexCLIUserAgent)
	}
	if s.cfg != nil && s.cfg.Gateway.ForceCodexCLI {
		req.Header.Set("User-Agent", codexCLIUserAgent)
	}
	apiKeyID := getAPIKeyIDFromContext(c)
	if sessionID != "" {
		isolated := isolateOpenAISessionID(apiKeyID, sessionID)
		req.Header.Set("Session_ID", isolated)
		req.Header.Set("Conversation_ID", isolated)
	}
	enforceCodexIdentityHeadersWithUA(req.Header, s.codexIdentityOverrideUA(account))
	account.ApplyHeaderOverrides(req.Header)
	cleanupOnError = false
	return req, nil
}

func buildOpenAIAlphaSearchResponsesWebSearchBody(alphaBody []byte, model string) ([]byte, error) {
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	tool := map[string]any{"type": "web_search"}
	if contextSize := strings.TrimSpace(gjson.GetBytes(alphaBody, "settings.search_context_size").String()); contextSize != "" {
		tool["search_context_size"] = contextSize
	}
	if userLocation := gjson.GetBytes(alphaBody, "settings.user_location"); userLocation.IsObject() {
		var location map[string]any
		if err := json.Unmarshal([]byte(userLocation.Raw), &location); err == nil && len(location) > 0 {
			tool["user_location"] = location
		}
	}
	return json.Marshal(map[string]any{
		"model": model, "stream": true, "store": false,
		"input": []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "input_text", "text": openAIAlphaSearchResponsesWebSearchPrompt(alphaBody)}}}},
		"tools": []any{tool},
	})
}

func openAIAlphaSearchResponsesWebSearchPrompt(alphaBody []byte) string {
	var b strings.Builder
	_, _ = b.WriteString("Execute this Codex standalone web.run request for another model.\nUse the hosted web_search tool when web/current information is needed.\nReturn concise source-backed results. Include titles, URLs, dates, and direct answers when available.\n")
	for _, field := range []string{"commands", "settings", "input"} {
		if value := strings.TrimSpace(gjson.GetBytes(alphaBody, field).Raw); value != "" {
			_, _ = fmt.Fprintf(&b, "\n%s JSON:\n%s", field, value)
		}
	}
	return b.String()
}

func openAIAlphaSearchInboundHeader(c *gin.Context, key string) string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.GetHeader(key))
}

func (s *OpenAIGatewayService) ensureOpenAIAlphaSearchAuthMetadata(ctx context.Context, account *Account, token string, proxyURL string) error {
	if s == nil || account == nil || !account.IsOpenAIPersonalAccessToken() || strings.TrimSpace(account.GetChatGPTAccountID()) != "" || s.openAITokenProvider == nil || s.openAITokenProvider.openAIOAuthService == nil {
		return nil
	}
	oauthService := s.openAITokenProvider.openAIOAuthService
	tokenInfo, err := oauthService.ValidateCodexPersonalAccessToken(ctx, token, proxyURL)
	if err != nil {
		return fmt.Errorf("validate Codex PAT metadata for alpha/search: %w", err)
	}
	credentials := shallowCopyMap(account.Credentials)
	for key, value := range oauthService.BuildAccountCredentials(tokenInfo) {
		credentials[key] = value
	}
	credentials = NormalizeOpenAIPersonalAccessTokenCredentials(account, tokenInfo, credentials)
	account.Credentials = shallowCopyMap(credentials)
	if s.accountRepo != nil {
		if err := persistAccountCredentials(ctx, s.accountRepo, account, credentials); err != nil {
			return fmt.Errorf("persist Codex PAT metadata for alpha/search: %w", err)
		}
	}
	return nil
}

// isOpenAIAlphaSearchEndpointUnsupported 识别「API key 上游没有实现
// /v1/alpha/search 端点」的响应。404/405 不在通用 failover 状态集里（模型
// 调用中的 404 通常是用户请求问题），但对这个独立工具端点而言，它几乎只
// 意味着所选上游（官方平台或第三方中转）不提供该端点——应换号重试，而
// 不是把 404 透传给客户端，否则混合分组里 OAuth 账号明明可以承接搜索，
// 请求却可能死在先被选中的 API key 账号上。
func isOpenAIAlphaSearchEndpointUnsupported(account *Account, statusCode int) bool {
	if account == nil || account.Type != AccountTypeAPIKey {
		return false
	}
	return statusCode == http.StatusNotFound || statusCode == http.StatusMethodNotAllowed
}

func shouldApplyOpenAIAlphaSearchAccountErrorSideEffects(statusCode int) bool {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
		// 401：工具端点的 access enforcement 不代表凭据全局失效；
		// 404/405：端点不存在只说明该上游不支持独立搜索，账号本身健康。
		// 两类都只换号，不写账号错误状态。
		return false
	default:
		return true
	}
}

func openAIAlphaSearchResponseFromResponsesSSE(body []byte) ([]byte, error) {
	output, results := parseOpenAIResponsesSSEForAlphaSearch(body)
	response := map[string]any{"output": output}
	if len(results) > 0 {
		response["results"] = results
	}
	return json.Marshal(response)
}

func parseOpenAIResponsesSSEForAlphaSearch(body []byte) (string, []any) {
	var output strings.Builder
	results := make([]any, 0)
	seenURLs := make(map[string]struct{})
	for _, block := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n\n") {
		data := openAIAlphaSearchSSEData(block)
		if data == "" || data == "[DONE]" {
			continue
		}
		var event map[string]any
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}
		if event["type"] == "response.output_text.delta" {
			if delta, _ := event["delta"].(string); delta != "" {
				_, _ = output.WriteString(delta)
			}
		}
		collectOpenAIAlphaSearchURLCitations(event, &results, seenURLs)
	}
	return output.String(), results
}

func openAIAlphaSearchSSEData(block string) string {
	var lines []string
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, "data:") {
			lines = append(lines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func collectOpenAIAlphaSearchURLCitations(value any, results *[]any, seen map[string]struct{}) {
	switch value := value.(type) {
	case map[string]any:
		if value["type"] == "url_citation" {
			if urlValue, _ := value["url"].(string); strings.TrimSpace(urlValue) != "" {
				urlValue = strings.TrimSpace(urlValue)
				if _, exists := seen[urlValue]; !exists {
					seen[urlValue] = struct{}{}
					result := map[string]any{"type": "text_result", "ref_id": fmt.Sprintf("turn0search%d", len(*results)), "url": urlValue}
					if title, _ := value["title"].(string); strings.TrimSpace(title) != "" {
						result["title"] = strings.TrimSpace(title)
					}
					*results = append(*results, result)
				}
			}
		}
		for _, child := range value {
			collectOpenAIAlphaSearchURLCitations(child, results, seen)
		}
	case []any:
		for _, child := range value {
			collectOpenAIAlphaSearchURLCitations(child, results, seen)
		}
	}
}

func (s *OpenAIGatewayService) buildOpenAIAlphaSearchRequest(ctx context.Context, c *gin.Context, account *Account, sourceBody []byte, body []byte, bodyHandle *RequestBodyHandle, token string) (*http.Request, error) {
	clientBeta := ""
	if c != nil {
		clientBeta = c.GetHeader("OpenAI-Beta")
	}
	req, err := s.buildUpstreamRequestOpenAIPassthrough(ctx, c, account, sourceBody, body, openAIMatchedRequestBodyHandle{handle: bodyHandle}, token)
	if err != nil {
		return nil, err
	}

	targetURL, err := s.openAIAlphaSearchURL(account)
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("parse alpha search URL: %w", err)
	}
	if c != nil && c.Request != nil && c.Request.URL != nil {
		query := parsedURL.Query()
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				query.Add(key, value)
			}
		}
		parsedURL.RawQuery = query.Encode()
	}
	req.URL = parsedURL
	req.Header.Set("Accept", "application/json")
	if clientBeta == "" {
		req.Header.Del("OpenAI-Beta")
	}
	if version := strings.TrimSpace(c.GetHeader("Version")); version != "" {
		req.Header.Set("Version", version)
	} else if account.Type == AccountTypeOAuth {
		req.Header.Set("Version", codexCLIVersion)
	}
	stripOpenAIAlphaSearchResponsesHeaders(req.Header)
	if account.Type == AccountTypeOAuth {
		enforceCodexIdentityHeadersWithUA(req.Header, s.codexIdentityOverrideUA(account))
	}
	return req, nil
}

func stripOpenAIAlphaSearchResponsesHeaders(headers http.Header) {
	if headers == nil {
		return
	}
	for _, key := range []string{
		"OpenAI-Beta",
		"Session_ID",
		"Conversation_ID",
		"X-Codex-Beta-Features",
		"X-Codex-Turn-State",
		responsesLiteHeaderKey,
	} {
		headers.Del(key)
	}
}

func (s *OpenAIGatewayService) openAIAlphaSearchURL(account *Account) (string, error) {
	if account == nil {
		return "", fmt.Errorf("account is required")
	}
	switch account.Type {
	case AccountTypeOAuth:
		return chatgptCodexAlphaSearchURL, nil
	case AccountTypeAPIKey:
		baseURL := account.GetOpenAIBaseURL()
		if baseURL == "" {
			return openAIPlatformAlphaSearchURL, nil
		}
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		return buildOpenAIEndpointURL(validatedURL, "/v1/alpha/search"), nil
	default:
		return "", fmt.Errorf("unsupported OpenAI account type: %s", account.Type)
	}
}
