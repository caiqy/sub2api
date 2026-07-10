package handler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGatewayHandler_MessagesRequestBodyUsesCoordinator(t *testing.T) {
	assertGatewayRequestBodyCoordinator(t, "gateway_handler.go", "func (h *GatewayHandler) Messages", true)
}

func TestGatewayHandler_ResponsesRequestBodyUsesCoordinator(t *testing.T) {
	assertGatewayRequestBodyCoordinator(t, "gateway_handler_responses.go", "func (h *GatewayHandler) Responses", false)
}

func assertGatewayRequestBodyCoordinator(t *testing.T, filename, declaration string, messages bool) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(".", filename))
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	source := string(body)
	start := strings.Index(source, declaration)
	if start < 0 {
		t.Fatalf("%s declaration not found", declaration)
	}
	source = source[start:]
	if !strings.Contains(source, "newJSONRequestBody(c.Request)") {
		t.Fatalf("%s must create a JSON request body coordinator", declaration)
	}
	if !strings.Contains(source, "defer coordinator.Cleanup()") {
		t.Fatalf("%s must clean up the request body coordinator", declaration)
	}
	if !strings.Contains(source, "coordinator.Effective()") {
		t.Fatalf("%s must forward the effective request body handle", declaration)
	}
	if messages && !strings.Contains(source, "coordinator.ReadRaw()") {
		t.Fatalf("%s must read raw bytes only during synchronous request processing", declaration)
	}
}
