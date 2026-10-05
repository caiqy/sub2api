package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 分组自助提前重置开关：未传时保持 nil（由 service 默认开启），显式 false 必须原样传给 service。
func TestGroupHandlerQuotaAdvanceFieldPassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newStubAdminService()
	h := NewGroupHandler(svc, nil, nil)
	r := gin.New()
	r.POST("/groups", h.Create)

	for _, tc := range []struct {
		name     string
		body     string
		expected *bool
	}{
		{name: "absent keeps nil so the service default applies", body: `{"name":"g1","platform":"anthropic"}`, expected: nil},
		{name: "explicit false is forwarded", body: `{"name":"g2","platform":"anthropic","allow_quota_advance":false}`, expected: boolPtr(false)},
		{name: "explicit true is forwarded", body: `{"name":"g3","platform":"anthropic","allow_quota_advance":true}`, expected: boolPtr(true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := len(svc.createdGroups)
			req := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)
			require.Equal(t, http.StatusOK, res.Code)
			require.Len(t, svc.createdGroups, before+1)
			got := svc.createdGroups[before].AllowQuotaAdvance
			if tc.expected == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, *tc.expected, *got)
		})
	}
}
