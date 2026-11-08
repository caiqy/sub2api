package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMessagesParseBudgetClassification(t *testing.T) {
	ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), httputil.NewBodyMemoryBudget(1024))
	defer release()
	h, err := service.NewRequestBodyHandleFromBytes([]byte(`{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hello"}]}`), service.RequestBodyHandleOptions{Context: ctx, SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer service.CleanupRequestBodyHandle(h)
	if err := httputil.ReserveRequestBodyMemory(ctx, 1024); err != nil {
		t.Fatal(err)
	}
	_, err = service.ParseGatewayRequest(service.NewRequestBodyRefFromHandle(h), "anthropic")
	if !errors.Is(err, httputil.ErrRequestBodyMemoryBudget) {
		t.Fatalf("err=%v", err)
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil).WithContext(ctx)
	(&GatewayHandler{}).handleMessagesParseError(c, fmt.Errorf("parse: %w", err), false)
	if rec.Code != 503 || rec.Header().Get("Retry-After") != "1" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestCanonicalBodyBudgetFourEntries(t *testing.T) {
	for _, entry := range []string{"claude-count", "openai-messages", "openai-count", "alpha-search"} {
		t.Run(entry, func(t *testing.T) {
			body := `{"model":"gpt-5.1","max_tokens":16,"input":"hello","messages":[{"role":"user","content":"hello"}]}`
			group := &service.Group{ID: 65, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
			upstream := &responsesSpoolTransportUpstream{}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{}, upstream)
			route := "/v1/messages"
			handler := env.handler.Messages
			switch entry {
			case "openai-count":
				route = "/v1/messages/count_tokens"
				handler = env.handler.CountTokens
			case "alpha-search":
				route = "/v1/alpha/search"
				handler = env.handler.AlphaSearch
			}
			router := env.router(route, handler)
			if entry == "claude-count" {
				g := newTerminalGatewayMessagesEnv(t, group, upstream)
				route = "/v1/messages/count_tokens"
				router = g.routerFor(route, g.handler.CountTokens)
			}
			// Ingress owns exactly len(body)+1 bytes, leaving no capacity for canonical storage.
			ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), httputil.NewBodyMemoryBudget(int64(len(body)+1)))
			defer release()
			req := httptest.NewRequest(http.MethodPost, route, strings.NewReader(body)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != 503 || rec.Header().Get("Retry-After") != "1" {
				t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
			}
			if upstream.calls != 0 {
				t.Fatalf("upstream calls=%d", upstream.calls)
			}
		})
	}
}

func TestMessagesParseErrorContracts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		cancel bool
	}{
		{"malformed", errors.New("invalid JSON"), 400, false},
		{"timeout", fmt.Errorf("parse: %w", os.ErrDeadlineExceeded), 408, false},
		{"oversized", fmt.Errorf("parse: %w", &http.MaxBytesError{Limit: 32}), 413, false},
		{"spool", fmt.Errorf("parse: %w", service.ErrRequestBodySpool), 503, false},
		{"canceled", fmt.Errorf("parse: %w", context.Canceled), 499, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest("POST", "/v1/messages", nil).WithContext(ctx)
			(&GatewayHandler{}).handleMessagesParseError(c, tc.err, false)
			if c.Writer.Status() != tc.status {
				t.Fatalf("status=%d body=%s", c.Writer.Status(), rec.Body.String())
			}
			if rec.Header().Get("Retry-After") != "" {
				t.Fatalf("unexpected retry header: %v", rec.Header())
			}
			if tc.cancel && (c.Writer.Written() || rec.Body.Len() != 0) {
				t.Fatalf("canceled response written: %s", rec.Body.String())
			}
		})
	}
	if _, err := service.ParseGatewayRequest(service.NewRequestBodyRef([]byte(`{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hello"}]}`)), "anthropic"); err != nil {
		t.Fatalf("valid control: %v", err)
	}
	// Validate the malformed control through the real parser as well.
	if _, err := service.ParseGatewayRequest(service.NewRequestBodyRef([]byte(`{"model":`)), "anthropic"); err == nil {
		t.Fatal("malformed JSON parsed successfully")
	}
}

func TestParsedMetadataBudgetClassification(t *testing.T) {
	for _, name := range []string{"session-hash", "claude-client-system"} {
		t.Run(name, func(t *testing.T) {
			ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), httputil.NewBodyMemoryBudget(4096))
			defer release()
			body := []byte(`{"model":"claude-sonnet-4-5","system":"system","messages":[{"role":"user","content":"hello"}]}`)
			handle, err := service.NewRequestBodyHandleFromBytes(body, service.RequestBodyHandleOptions{Context: ctx, SpoolThresholdBytes: 1, TempDir: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			defer service.CleanupRequestBodyHandle(handle)
			parsed, err := service.ParseGatewayRequest(service.NewRequestBodyRefFromHandle(handle), "anthropic")
			if err != nil {
				t.Fatal(err)
			}
			// Fill whatever capacity remains after parsing, without guessing its allocation count.
			for httputil.ReserveRequestBodyMemory(ctx, 1) == nil {
			}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest("POST", "/v1/messages", nil).WithContext(ctx)
			if name == "session-hash" {
				_, err = (&service.GatewayService{}).GenerateSessionHash(parsed)
			} else {
				c.Request.Header.Set("User-Agent", "claude-cli/2.1.260 (external, cli)")
				err = SetClaudeCodeClientContext(c, nil, parsed)
			}
			if !errors.Is(err, httputil.ErrRequestBodyMemoryBudget) {
				t.Fatalf("metadata err=%v", err)
			}
			(&GatewayHandler{}).handleMessagesParseError(c, err, false)
			if rec.Code != 503 || rec.Header().Get("Retry-After") != "1" {
				t.Fatalf("status=%d headers=%v", rec.Code, rec.Header())
			}
		})
	}
}
