//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRecordAlphaSearchUsageIncludesDetailSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 4301, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 4302, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "search-key"},
	}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, &openAIChatCompletionsHTTPUpstreamStub{})

	router := env.router("/alpha/search", func(c *gin.Context) {
		service.SetUsageRequestBody(c, service.RequestBodyPreviewSnapshot(`{"model":"gpt-5.6-sol","query":"release notes"}`, 50))
		c.JSON(http.StatusOK, gin.H{"results": []gin.H{{"title": "Release notes"}}})
		env.handler.recordAlphaSearchUsage(
			c, env.apiKey, account, nil, service.ChannelMappingResult{}, "gpt-5.6-sol", "payload-hash",
			&service.OpenAIForwardResult{RequestID: "search-1", Model: "gpt-5.6-sol", UpstreamModel: "gpt-5.6-sol", WebSearchCalls: 1},
			env.apiKey.UserID,
		)
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/alpha/search", strings.NewReader(`{"model":"gpt-5.6-sol","query":"release notes"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	usageLog := <-env.usageRepo.created
	require.NotNil(t, usageLog.DetailSnapshot)
	require.Contains(t, usageLog.DetailSnapshot.RequestBody, "release notes")
	require.Contains(t, usageLog.DetailSnapshot.ResponseBody, "Release notes")
}
