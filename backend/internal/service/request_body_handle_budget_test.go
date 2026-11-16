package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequestBodyHandleBudgetSharedAcrossMaterializations(t *testing.T) {
	for _, threshold := range []int64{1, 1024} {
		t.Run(string(rune('a'+threshold%2)), func(t *testing.T) {
			budget := httputil.NewBodyMemoryBudget(128)
			ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
			defer release()
			handle, err := NewRequestBodyHandleFromBytes(bytes.Repeat([]byte("x"), 64), RequestBodyHandleOptions{Context: ctx, SpoolThresholdBytes: threshold, TempDir: t.TempDir()})
			require.NoError(t, err)
			defer CleanupRequestBodyHandle(handle)
			_, err = handle.ReadAll()
			require.NoError(t, err)
			if threshold == 1 {
				_, err = handle.ReadAll()
				require.NoError(t, err)
			}
			_, err = handle.ReadAll()
			require.ErrorIs(t, err, httputil.ErrRequestBodyMemoryBudget)
			release()
			other, done := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
			defer done()
			require.NoError(t, httputil.ReserveRequestBodyMemory(other, 128))
		})
	}
}

func bodyBudgetUsed(t *testing.T, budget *httputil.BodyMemoryBudget, limit int64) int64 {
	t.Helper()
	ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
	defer release()
	lo, hi := int64(0), limit
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		if httputil.ReserveRequestBodyMemory(ctx, mid) == nil {
			httputil.ReleaseRequestBodyMemory(ctx, mid)
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return limit - lo
}

func TestRequestBodyHandleScopedCopyLifetime(t *testing.T) {
	for _, threshold := range []int64{1, 1024} {
		t.Run(string(rune('a'+threshold%2)), func(t *testing.T) {
			const limit = 1024
			budget := httputil.NewBodyMemoryBudget(limit)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx, releaseRoot := httputil.WithRequestBodyMemoryBudget(ctx, budget)
			defer releaseRoot()
			h, err := NewRequestBodyHandleFromBytes(bytes.Repeat([]byte("x"), 64), RequestBodyHandleOptions{Context: ctx, SpoolThresholdBytes: threshold, TempDir: t.TempDir()})
			require.NoError(t, err)
			defer CleanupRequestBodyHandle(h)
			storage := bodyBudgetUsed(t, budget, limit)
			first, releaseFirst, err := h.ReadAllScoped()
			require.NoError(t, err)
			defer releaseFirst()
			second, releaseSecond, err := h.ReadAllScoped()
			require.NoError(t, err)
			defer releaseSecond()
			resume := make(chan struct{})
			resumeCopy := sync.OnceFunc(func() { close(resume) })
			defer resumeCopy()
			observed := make(chan byte, 1)
			go func(data []byte) {
				<-resume
				observed <- data[0]
			}(second)
			first[0] = '!'
			require.Equal(t, byte('x'), second[0], "copies must remain independently mutable")
			require.EqualValues(t, storage+128, bodyBudgetUsed(t, budget, limit))
			cancel()
			releaseRoot()
			require.EqualValues(t, 128, bodyBudgetUsed(t, budget, limit), "retained copies outlive cancellation and root release")
			_, failedRelease, err := h.ReadAllScoped()
			failedRelease()
			require.ErrorIs(t, err, httputil.ErrRequestBodyMemoryBudget)
			require.NoError(t, h.Cleanup())
			require.EqualValues(t, 128, bodyBudgetUsed(t, budget, limit), "handle cleanup cannot refund live copies")
			runtime.KeepAlive(first)
			first = nil
			releaseFirst()
			releaseFirst()
			require.EqualValues(t, 64, bodyBudgetUsed(t, budget, limit))
			second = nil
			resumeCopy()
			require.Equal(t, byte('x'), <-observed, "asynchronous owner can still read after root cleanup")
			releaseSecond()
			require.Zero(t, bodyBudgetUsed(t, budget, limit))
		})
	}
}

func TestGatewayRequestScopedParsingDoesNotAccumulate(t *testing.T) {
	body := []byte(`{"model":"claude-test","system":"instructions","messages":[{"role":"user","content":"hello"}]}`)
	limit := int64(4 * len(body))
	budget := httputil.NewBodyMemoryBudget(limit)
	ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
	defer release()
	h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{Context: ctx, SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	defer CleanupRequestBodyHandle(h)
	parsed, err := ParseGatewayRequest(NewRequestBodyRefFromHandle(h), domain.PlatformAnthropic)
	require.NoError(t, err)
	for range 32 {
		clone, err := parsed.CloneForHandle(h)
		require.NoError(t, err)
		_, err = (&GatewayService{}).GenerateSessionHash(clone)
		require.NoError(t, err)
		_, err = BuildAnthropicDigestChain(clone)
		require.NoError(t, err)
		_, err = IsRealUserMessage(clone)
		require.NoError(t, err)
		var messages any
		require.NoError(t, clone.DecodeMessages(&messages))
		_, _, err = clone.SystemValue()
		require.NoError(t, err)
		require.Zero(t, bodyBudgetUsed(t, budget, limit))
	}
	competitor, releaseCompetitor := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
	defer releaseCompetitor()
	require.NoError(t, httputil.ReserveRequestBodyMemory(competitor, limit))
	_, err = parsed.CloneForHandle(h)
	require.ErrorIs(t, err, httputil.ErrRequestBodyMemoryBudget, "real overload must not turn into a successful parse")
	releaseCompetitor()
	require.NoError(t, os.WriteFile(h.spoolPath, []byte("{"), 0600))
	_, releaseFailed, err := h.ReadAllScoped()
	releaseFailed()
	require.ErrorIs(t, err, ErrRequestBodySpool)
	require.Zero(t, bodyBudgetUsed(t, budget, limit), "failed reads must refund their lease")
}

func TestGatewayRequestNormalizationAndRawLeaseRemainCharged(t *testing.T) {
	body := []byte(`{"model":"claude-test[1m]","system":"instructions","messages":[]}`)
	limit := int64(4 * len(body))
	budget := httputil.NewBodyMemoryBudget(limit)
	ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
	defer release()
	h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{Context: ctx, SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	defer CleanupRequestBodyHandle(h)
	parsed, err := ParseGatewayRequest(NewRequestBodyRefFromHandle(h), domain.PlatformAnthropic)
	require.NoError(t, err)
	normalized, err := parsed.Body.ReadAll()
	require.NoError(t, err)
	require.Equal(t, "claude-test", parsed.Model)
	require.EqualValues(t, len(normalized), bodyBudgetUsed(t, budget, limit), "normalization output escapes parsing")
	parsed, err = ParseGatewayRequest(NewRequestBodyRefFromHandle(h), "")
	require.NoError(t, err)
	raw, releaseRaw, err := parsed.rawScoped(parsed.systemRange)
	require.NoError(t, err)
	defer releaseRaw()
	text := extractTextFromSystemRaw(raw) // A string alias can pin the whole body.
	require.EqualValues(t, len(normalized)+len(body), bodyBudgetUsed(t, budget, limit))
	release()
	require.EqualValues(t, len(body), bodyBudgetUsed(t, budget, limit))
	require.Equal(t, "instructions", text)
	runtime.KeepAlive(text)
	text, raw = "", nil
	releaseRaw()
	require.Zero(t, bodyBudgetUsed(t, budget, limit))
}

func TestGatewayRequestNormalizationEncodingGrowth(t *testing.T) {
	for _, model := range []string{"claude-" + strings.Repeat("\u2028\u2029", 64), "claude-" + strings.Repeat("<>&", 64)} {
		body := []byte(`{"model":"` + model + `[1m]","messages":[]}`)
		limit := int64(16 * len(body))
		for _, tight := range []bool{false, true} {
			budgetLimit := limit
			if tight {
				budgetLimit = int64(len(body) + 1)
			}
			budget := httputil.NewBodyMemoryBudget(budgetLimit)
			ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
			h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{Context: ctx, SpoolThresholdBytes: 1, TempDir: t.TempDir()})
			require.NoError(t, err)
			parsed, err := ParseGatewayRequest(NewRequestBodyRefFromHandle(h), domain.PlatformAnthropic)
			if tight {
				require.ErrorIs(t, err, httputil.ErrRequestBodyMemoryBudget)
				require.Zero(t, bodyBudgetUsed(t, budget, budgetLimit), "failed normalization must not leak a temporary lease")
			} else {
				require.NoError(t, err)
				normalized, err := parsed.Body.ReadAll()
				require.NoError(t, err)
				require.Greater(t, len(normalized), len(body), "JSON escaping can grow a shortened model")
				require.EqualValues(t, len(normalized), bodyBudgetUsed(t, budget, budgetLimit), "all output bytes must remain charged")
				require.Equal(t, model, parsed.Model)
			}
			require.NoError(t, h.Cleanup())
			release()
			require.Zero(t, bodyBudgetUsed(t, budget, budgetLimit))
		}
	}
}

func TestImagesScopedReadPreservesBudgetError(t *testing.T) {
	for _, accountType := range []string{AccountTypeAPIKey, AccountTypeOAuth} {
		t.Run(accountType, func(t *testing.T) {
			body := []byte(`{"model":"gpt-image-2","prompt":"draw a cat"}`)
			limit := int64(len(body) + 1)
			if accountType == AccountTypeOAuth {
				limit = 1
			}
			budget := httputil.NewBodyMemoryBudget(limit)
			ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
			defer release()
			h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{Context: ctx, SpoolThresholdBytes: 1, TempDir: t.TempDir()})
			require.NoError(t, err)
			defer CleanupRequestBodyHandle(h)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, openAIImagesGenerationsEndpoint, nil).WithContext(ctx)
			BindOpenAIRequestBodyHandle(c, h)
			upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: 400, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"controlled error"}}`))}}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
			parsed, err := svc.ParseOpenAIImagesRequest(c, body)
			require.NoError(t, err)
			account := &Account{ID: 1, Platform: PlatformOpenAI, Type: accountType, Credentials: map[string]any{"api_key": "test-key", "access_token": "test-token", "base_url": "https://image-upstream.example/v1"}}
			_, err = svc.ForwardImages(ctx, c, account, nil, parsed, "")
			require.ErrorIs(t, err, httputil.ErrRequestBodyMemoryBudget, "handlers must recognize overload and retain 503/Retry-After")
			if accountType == AccountTypeAPIKey {
				require.NotNil(t, upstream.lastReq, "overload is from the second copy after the upstream error")
			} else {
				require.Nil(t, upstream.lastReq)
			}
			require.Zero(t, bodyBudgetUsed(t, budget, limit))
		})
	}
}
