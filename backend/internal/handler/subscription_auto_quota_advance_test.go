//go:build unit

package handler

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/require"
)

func TestUpdateAutoQuotaAdvanceRequestRequiresExplicitBoolean(t *testing.T) {
	for _, body := range []string{`{}`, `{"auto_quota_advance":null}`, `{"auto_quota_advance":"false"}`} {
		t.Run(body, func(t *testing.T) {
			var request UpdateAutoQuotaAdvanceRequest
			require.Error(t, binding.JSON.BindBody([]byte(body), &request))
		})
	}
	for _, body := range []string{`{"auto_quota_advance":true}`, `{"auto_quota_advance":false}`} {
		t.Run(body, func(t *testing.T) {
			var request UpdateAutoQuotaAdvanceRequest
			require.NoError(t, binding.JSON.BindBody([]byte(body), &request))
		})
	}
}
