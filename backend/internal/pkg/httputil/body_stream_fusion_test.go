package httputil

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type stalledFusionBody struct{ first bool }

func (b *stalledFusionBody) Read(p []byte) (int, error) {
	if !b.first {
		b.first = true
		return copy(p, "abc"), nil
	}
	return 0, os.ErrDeadlineExceeded
}
func (*stalledFusionBody) Close() error { return nil }

func TestStreamingBodyPreservesTimeoutDiagnostics(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.Body = &stalledFusionBody{}
	reader, err := NewDecodedRequestBodyReader(req)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	_, err = io.ReadAll(reader)
	var readErr *RequestBodyReadError
	if !errors.As(err, &readErr) || readErr.BytesRead != 3 || !IsRequestBodyReadTimeout(err) {
		t.Fatalf("lost timeout diagnostics: %#v %v", readErr, err)
	}
}

func TestNormalizeSpooledBodyUsesIngressBudget(t *testing.T) {
	budget := NewBodyMemoryBudget(1)
	ctx, release := WithRequestBodyMemoryBudget(context.Background(), budget)
	defer release()
	if _, err := NormalizeLenientJSONRequestBodyWithContext(ctx, []byte("{\"input\":\"a\nb\"}"), 4096); !errors.Is(err, ErrRequestBodyMemoryBudget) {
		t.Fatalf("normalization bypassed ingress budget: %v", err)
	}
}

func TestStreamingZstdWindowUsesIngressBudget(t *testing.T) {
	budget := NewBodyMemoryBudget(maxRequestZstdWindowSize - 1)
	ctx, release := WithRequestBodyMemoryBudget(context.Background(), budget)
	defer release()
	req := httptest.NewRequest("POST", "/", strings.NewReader("unused")).WithContext(ctx)
	req.Header.Set("Content-Encoding", "zstd")
	if _, err := NewDecodedRequestBodyReader(req); !errors.Is(err, ErrRequestBodyMemoryBudget) {
		t.Fatalf("expected overload, got %v", err)
	}
	if used := requestBodyBudgetUsed(budget); used != 0 {
		t.Fatalf("leaked %d bytes", used)
	}
}
