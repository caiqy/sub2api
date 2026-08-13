package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func requireOpenAIRequestBodiesEqualExceptFingerprint(t *testing.T, first, second []byte) {
	t.Helper()
	var firstPayload, secondPayload map[string]any
	require.NoError(t, json.Unmarshal(first, &firstPayload))
	require.NoError(t, json.Unmarshal(second, &secondPayload))
	delete(firstPayload, "client_metadata")
	delete(secondPayload, "client_metadata")
	require.Equal(t, firstPayload, secondPayload)
}

func requireOpenAIFingerprintsDiffer(t *testing.T, first, second []byte) {
	t.Helper()
	fields := []string{
		"x-codex-installation-id",
		"session_id",
		"thread_id",
		"x-codex-window-id",
	}
	different := false
	for _, field := range fields {
		firstValue := gjson.GetBytes(first, "client_metadata."+field)
		secondValue := gjson.GetBytes(second, "client_metadata."+field)
		require.True(t, firstValue.Exists(), "first fingerprint must include %s", field)
		require.True(t, secondValue.Exists(), "second fingerprint must include %s", field)
		require.NotEmpty(t, firstValue.String(), "first fingerprint %s must be non-empty", field)
		require.NotEmpty(t, secondValue.String(), "second fingerprint %s must be non-empty", field)
		different = different || firstValue.String() != secondValue.String()
	}
	require.True(t, different, "different accounts must change an account-derived fingerprint field")
}
