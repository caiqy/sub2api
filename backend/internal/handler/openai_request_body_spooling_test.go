package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayHandler_ChatAndEmbeddingsRequestBodyReplay(t *testing.T) {
	body := []byte(`{"model":"mapped-model","input":"` + strings.Repeat("x", 12<<20) + `"}`)
	tempDir := t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: tempDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	coordinator, err := newJSONRequestBody(httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(body)))
	require.NoError(t, err)
	require.NoError(t, coordinator.SetEffectiveBytes(body))

	for range 2 {
		reader, err := coordinator.Effective().Open()
		require.NoError(t, err)
		got, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		require.Equal(t, body, got)
	}
	coordinator.Cleanup()
	entries, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}
