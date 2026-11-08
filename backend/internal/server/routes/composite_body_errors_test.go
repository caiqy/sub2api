package routes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type compositeBodyErrorReader struct{ err error }

func (r compositeBodyErrorReader) Read([]byte) (int, error) { return 0, r.err }

func TestCompositeBodyReadErrorsKeepIngressClassification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		err    error
		status int
		cancel bool
	}{
		{"timeout", os.ErrDeadlineExceeded, http.StatusRequestTimeout, false},
		{"budget", pkghttputil.ErrRequestBodyMemoryBudget, http.StatusServiceUnavailable, false},
		{"size", &http.MaxBytesError{Limit: 1}, http.StatusRequestEntityTooLarge, false},
		{"cancelled", context.Canceled, 499, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{Group: &service.Group{Platform: service.PlatformComposite}})
			})
			router.Use(compositeTargetPlatformMiddleware(effectiveCompositeRouteResolverForTest(nil)))
			router.POST("/v1/responses", func(c *gin.Context) { t.Fatal("failed read reached handler") })
			req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			req.Body = io.NopCloser(compositeBodyErrorReader{tc.err})
			if tc.cancel {
				ctx, cancel := context.WithCancel(req.Context())
				cancel()
				req = req.WithContext(ctx)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			require.Equal(t, tc.status, response.Code)
			if tc.name == "budget" {
				require.Equal(t, "1", response.Header().Get("Retry-After"))
			}
			if tc.cancel {
				require.Empty(t, response.Body.String())
			}
		})
	}
}
