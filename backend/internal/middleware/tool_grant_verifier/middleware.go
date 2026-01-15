package tool_grant_verifier

import (
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	toolgrantservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/agent/tool_grant"
	"github.com/gin-gonic/gin"
)

// Middleware validates ToolGrant tokens on protected routes.
func Middleware(service *toolgrantservice.Service, extractToken func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.Next()
			return
		}
		token := ""
		if extractToken != nil {
			token = extractToken(c)
		} else {
			token = c.GetHeader("X-ToolGrant")
		}
		tenantID, ok := middleware.TenantUUIDFromContext(c.Request.Context())
		if !ok || tenantID == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "tenant context missing"})
			return
		}
		if _, err := service.Validate(c.Request.Context(), tenantID, token); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error(), "code": "TOOLGRANT_INVALID"})
			return
		}
		c.Next()
	}
}
