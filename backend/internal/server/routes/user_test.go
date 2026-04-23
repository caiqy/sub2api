package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterUserRoutesRegistersImageHistoryEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterUserRoutes(
		v1,
		&handler.Handlers{
			ImageHistory: handler.NewImageHistoryHandler(nil, nil),
		},
		servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
		nil,
	)

	for _, path := range []string{"/api/v1/images/history", "/api/v1/images/history/123"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code, "path=%s should be registered and reach handler", path)
	}
}
