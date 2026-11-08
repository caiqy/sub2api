package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func trafficBillingStart(settings *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := strings.ToLower(c.Request.URL.Path)
		if c.Request.Method == http.MethodGet || strings.Contains(path, "/batches") || strings.Contains(path, "/batch/") || strings.Contains(path, "/async") || strings.Contains(path, "/videos") || strings.Contains(path, "/tasks") {
			c.Next()
			return
		}
		if key, ok := middleware.GetAPIKeyFromContext(c); ok && key != nil {
			c.Request = c.Request.WithContext(service.CaptureTrafficBillingStart(c.Request.Context(), settings, key.User))
		}
		c.Next()
	}
}
