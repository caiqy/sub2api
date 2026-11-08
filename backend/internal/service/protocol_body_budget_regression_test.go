package service

import (
	"context"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAcceptedResponseBudget(t *testing.T)            { reviewAcceptedResponse(t, true) }
func TestAcceptedResponseWithoutContention(t *testing.T) { reviewAcceptedResponse(t, false) }
func reviewAcceptedResponse(t *testing.T, congested bool) {
	budget := httputil.NewBodyMemoryBudget(8 << 20)
	ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
	defer release()
	competitor, releaseCompetitor := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
	defer releaseCompetitor()
	body := []byte(`{"model":"gpt-5.1","service_tier":"priority","stream":false,"input":"` + strings.Repeat("x", (1<<20)+1) + `"}`)
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{Context: ctx, TempDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer CleanupRequestBodyHandle(handle)
	// Match the handler's ReadRaw and pre-forward canonicalBody materializations.
	if _, err = handle.ReadAll(); err != nil {
		t.Fatal(err)
	}
	body, err = handle.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	upstream := &httpUpstreamSequenceRecorder{responses: []*http.Response{{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"id":"resp_ok","usage":{"input_tokens":1,"output_tokens":1}}`))}}}
	if congested {
		upstream.afterDo = func(int) {
			charged := 0
			for httputil.ReserveRequestBodyMemory(competitor, 1024) == nil {
				charged += 1024
			}
			t.Logf("competing requests acquired %d bytes", charged)
		}
	}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(ctx)
	BindOpenAIRequestBodyHandle(c, handle)
	account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test", "base_url": "http://localhost:12345"}}
	result, err := svc.Forward(ctx, c, account, body)
	if upstream.callCount != 1 {
		t.Fatalf("calls=%d", upstream.callCount)
	}
	if err != nil || result == nil {
		t.Fatalf("accepted response lost: result=%+v err=%v", result, err)
	}
	if result.ServiceTier == nil || *result.ServiceTier != "priority" {
		t.Fatalf("service tier=%v", result.ServiceTier)
	}
	if result.Usage.InputTokens != 1 || result.Usage.OutputTokens != 1 || !c.Writer.Written() {
		t.Fatalf("usage=%+v written=%v", result.Usage, c.Writer.Written())
	}
}

func TestCanonicalHandleRewritePreservesBudget(t *testing.T) {
	ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), httputil.NewBodyMemoryBudget(1))
	defer release()
	if err := httputil.ReserveRequestBodyMemory(ctx, 1); err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(ctx)
	if h, owned, err := openAIRequestBodyHandleForContext(c, []byte(`{"model":"gpt-5.1"}`)); !errors.Is(err, httputil.ErrRequestBodyMemoryBudget) {
		if owned {
			CleanupRequestBodyHandle(h)
		}
		t.Fatal("new canonical handle bypassed exhausted request budget")
	}
	original, err := NewRequestBodyHandleFromBytes([]byte(`{"model":"gpt-5.1"}`), RequestBodyHandleOptions{Context: ctx, SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer CleanupRequestBodyHandle(original)
	if h, owned, err := openAIRequestBodyHandleForBytes(original, []byte(`{"model":"gpt-5.2"}`)); !errors.Is(err, httputil.ErrRequestBodyMemoryBudget) {
		if owned {
			CleanupRequestBodyHandle(h)
		}
		t.Fatal("rewritten canonical handle bypassed exhausted request budget")
	}
}

func TestRequestBodyHandleStorageBudgetLifetime(t *testing.T) {
	for _, mode := range []string{"cleanup", "open-reader", "read-all-copy"} {
		t.Run(mode, func(t *testing.T) {
			const limit = 1024
			budget := httputil.NewBodyMemoryBudget(limit)
			ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
			defer release()
			probe, releaseProbe := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
			defer releaseProbe()
			h, err := NewRequestBodyHandleFromBytes([]byte("hello"), RequestBodyHandleOptions{Context: ctx})
			if err != nil {
				t.Fatal(err)
			}
			defer CleanupRequestBodyHandle(h)
			var reader io.ReadCloser
			var copied []byte
			if mode == "open-reader" {
				reader, err = h.Open()
				if err != nil {
					t.Fatal(err)
				}
			}
			if mode == "read-all-copy" {
				copied, err = h.ReadAll()
				if err != nil {
					t.Fatal(err)
				}
			}
			if err = h.Cleanup(); err != nil {
				t.Fatal(err)
			}
			if mode == "open-reader" {
				if err = httputil.ReserveRequestBodyMemory(probe, limit); !errors.Is(err, httputil.ErrRequestBodyMemoryBudget) {
					t.Fatalf("reader storage released early: %v", err)
				}
				got, e := io.ReadAll(reader)
				if e != nil || string(got) != "hello" {
					t.Fatalf("reader lost bytes: %q %v", got, e)
				}
				if err = reader.Close(); err != nil {
					t.Fatal(err)
				}
				if err = reader.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "read-all-copy" {
				if string(copied) != "hello" {
					t.Fatal("copy lost")
				}
				if err = httputil.ReserveRequestBodyMemory(probe, limit); !errors.Is(err, httputil.ErrRequestBodyMemoryBudget) {
					t.Fatalf("copy charge released early: %v", err)
				}
				if err = httputil.ReserveRequestBodyMemory(probe, limit-int64(len(copied))); err != nil {
					t.Fatalf("storage not returned: %v", err)
				}
				releaseProbe()
				release()
				probe, releaseProbe = httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
				defer releaseProbe()
			}
			if err = httputil.ReserveRequestBodyMemory(probe, limit); err != nil {
				t.Fatalf("storage not returned: %v", err)
			}
		})
	}
}
