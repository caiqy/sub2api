package service

import (
	"bytes"
	"errors"
	"os"
	"testing"
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
