package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelTraceHandlerRejectsIDAndPaginationBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &ModelTraceHandler{}
	r := gin.New()
	r.GET("/models", h.Models)
	r.GET("/accounts/:id", h.History)
	r.POST("/accounts/:id", h.Create)
	for _, path := range []string{"/models?account_id=", "/models?account_id=-1", "/models?account_id=1&account_id=2", "/accounts/0", "/accounts/1?page=overflow", "/accounts/1?page_size=nan"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusBadRequest, w.Code, path)
	}
	for _, body := range []string{`{"model":"gpt-6-astra","unknown":1}`, `{"model":"gpt-6-astra"} {}`, strings.Repeat("x", 5000)} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/accounts/1", strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, w.Code)
	}
}
