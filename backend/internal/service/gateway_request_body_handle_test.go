package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
)

func TestRequestBodyRefHandleReadErrorPropagates(t *testing.T) {
	handle, err := NewRequestBodyHandleFromReader(bytes.NewReader([]byte(`{"model":"claude-test","messages":[]}`)), RequestBodyHandleOptions{
		SpoolThresholdBytes: 1,
		TempDir:             t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create handle: %v", err)
	}
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
	if err := os.Remove(handle.spoolPath); err != nil {
		t.Fatalf("remove spool file: %v", err)
	}

	ref := NewRequestBodyRefFromHandle(handle)
	if _, err := ref.ReadAll(); !errors.Is(err, ErrRequestBodySpool) {
		t.Fatalf("ReadAll error = %v, want ErrRequestBodySpool", err)
	}
	if _, err := ParseGatewayRequest(ref, ""); !errors.Is(err, ErrRequestBodySpool) {
		t.Fatalf("ParseGatewayRequest error = %v, want ErrRequestBodySpool", err)
	}
	parsed := &ParsedRequest{Body: ref}
	if _, err := parsed.CloneForHandle(handle); !errors.Is(err, ErrRequestBodySpool) {
		t.Fatalf("CloneForHandle error = %v, want ErrRequestBodySpool", err)
	}
}

func TestAntigravityForwardHandleAcceptsReopenableRequestBodyHandle(t *testing.T) {
	var _ func(*AntigravityGatewayService, context.Context, *gin.Context, *Account, *RequestBodyHandle, bool, ...ForwardGeminiOption) (*ForwardResult, error) = (*AntigravityGatewayService).ForwardHandle
}

func TestAntigravityRetryLoopReopensGeminiPayloadHandleForRetry(t *testing.T) {
	payload := []byte(`{"request":{"contents":[{"parts":[{"text":"retry payload"}]}]}}`)
	handle, err := NewRequestBodyHandleFromReader(bytes.NewReader(payload), RequestBodyHandleOptions{
		SpoolThresholdBytes: 1,
		TempDir:             t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create payload handle: %v", err)
	}
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	upstream := &retryPayloadHandleUpstream{responses: []int{http.StatusInternalServerError, http.StatusOK}}
	account := &Account{ID: 1, Name: "retry", Platform: PlatformAntigravity, Type: AccountTypeAPIKey, Concurrency: 1}
	result, err := (&AntigravityGatewayService{}).antigravityRetryLoop(antigravityRetryLoopParams{
		ctx:           context.Background(),
		prefix:        "[test]",
		account:       account,
		accessToken:   "token",
		action:        "generateContent",
		payloadHandle: handle,
		httpUpstream:  upstream,
		handleError: func(context.Context, string, *Account, int, http.Header, []byte, string, int64, string, bool) *handleModelRateLimitResult {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("retry loop: %v", err)
	}
	if result == nil || result.resp == nil || result.resp.StatusCode != http.StatusOK {
		t.Fatalf("result = %#v, want 200 response", result)
	}
	_ = result.resp.Body.Close()
	if len(upstream.bodies) != 2 {
		t.Fatalf("attempts = %d, want 2", len(upstream.bodies))
	}
	for i, body := range upstream.bodies {
		if !bytes.Equal(body, payload) {
			t.Fatalf("attempt %d body = %q, want %q", i+1, body, payload)
		}
	}
}

func TestAntigravityCreditsRetryReturnsPayloadReadError(t *testing.T) {
	handle, err := NewRequestBodyHandleFromReader(bytes.NewReader([]byte(`{"request":{"contents":[]}}`)), RequestBodyHandleOptions{
		SpoolThresholdBytes: 1,
		TempDir:             t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create payload handle: %v", err)
	}
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
	if err := os.Remove(handle.spoolPath); err != nil {
		t.Fatalf("remove payload spool: %v", err)
	}

	result := (&AntigravityGatewayService{}).attemptCreditsOveragesRetry(antigravityRetryLoopParams{
		ctx:           context.Background(),
		account:       &Account{ID: 1, Platform: PlatformAntigravity},
		payloadHandle: handle,
	}, "https://ag.test", "gemini-test", 0, http.StatusTooManyRequests, nil)
	if result.err == nil || !errors.Is(result.err, ErrRequestBodySpool) {
		t.Fatalf("credits retry error = %v, want wrapped ErrRequestBodySpool", result.err)
	}
}

func TestForwardAsResponsesHandle_SpoolTransportErrorPreservesSentinel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	handle, err := NewRequestBodyHandleFromBytes([]byte(`{"model":"claude-opus-4-6","input":"hello"}`), RequestBodyHandleOptions{})
	if err != nil {
		t.Fatalf("create handle: %v", err)
	}
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}},
			Gateway:  config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
		},
		httpUpstream:        responsesSpoolTransportUpstream{},
		rateLimitService:    &RateLimitService{},
		tlsFPProfileService: &TLSFingerprintProfileService{},
	}
	account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "token", "base_url": "https://api.anthropic.com"}}

	_, err = svc.ForwardAsResponsesHandle(context.Background(), c, account, handle, nil)
	if !errors.Is(err, ErrRequestBodySpool) {
		t.Fatalf("ForwardAsResponsesHandle error = %v, want ErrRequestBodySpool", err)
	}
	if c.Writer.Written() {
		t.Fatalf("transport spool failure wrote response: %s", recorder.Body.String())
	}
}

type retryPayloadHandleUpstream struct {
	responses []int
	bodies    [][]byte
}

type responsesSpoolTransportUpstream struct{ HTTPUpstream }

func (responsesSpoolTransportUpstream) DoWithTLS(*http.Request, string, int64, int, *tlsfingerprint.Profile) (*http.Response, error) {
	return nil, fmt.Errorf("read request body: %w", ErrRequestBodySpool)
}

func (u *retryPayloadHandleUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	if req.ContentLength <= 0 || req.GetBody == nil {
		return nil, errors.New("request is not handle-backed")
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	reopened, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reopened.Close() }()
	copyBody, err := io.ReadAll(reopened)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(body, copyBody) {
		return nil, errors.New("GetBody changed payload")
	}
	u.bodies = append(u.bodies, body)
	status := u.responses[len(u.bodies)-1]
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(nil))}, nil
}

func (u *retryPayloadHandleUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func TestParsedRequestCloneForHandlePreservesAttemptState(t *testing.T) {
	handle, err := NewRequestBodyHandleFromBytes([]byte(`{"model":"mapped-model","messages":[]}`), RequestBodyHandleOptions{})
	if err != nil {
		t.Fatalf("create handle: %v", err)
	}
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	parsed, err := ParseGatewayRequest(NewRequestBodyRef([]byte(`{"model":"original-model","messages":[]}`)), "")
	if err != nil {
		t.Fatalf("parse request: %v", err)
	}
	accepted := false
	parsed.Model = "mapped-model"
	parsed.OnUpstreamAccepted = func() { accepted = true }

	clone, err := parsed.CloneForHandle(handle)
	if err != nil {
		t.Fatalf("clone request: %v", err)
	}
	if clone.Model != "mapped-model" {
		t.Fatalf("model = %q, want mapped-model", clone.Model)
	}
	if clone.OnUpstreamAccepted == nil {
		t.Fatal("OnUpstreamAccepted was cleared during handle rebind")
	}
	clone.OnUpstreamAccepted()
	if !accepted {
		t.Fatal("OnUpstreamAccepted was not preserved during handle rebind")
	}
}
