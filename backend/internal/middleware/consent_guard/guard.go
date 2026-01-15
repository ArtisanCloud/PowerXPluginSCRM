package consent_guard

import (
	"net/http"

	middleware "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	agentsecurity "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/agent/security"
	"github.com/gin-gonic/gin"
)

// AssetExtractor extracts the list of asset keys that the handler intends to
// access. The implementation can inspect the request and context.
type AssetExtractor func(*gin.Context) []string

// Middleware enforces consent against the provided PrivacyGuard using the
// extractor to determine required asset keys.
func Middleware(guard *agentsecurity.PrivacyGuard, extractor AssetExtractor) gin.HandlerFunc {
	return func(c *gin.Context) {
		if guard == nil {
			c.Next()
			return
		}
		tenantID, ok := middleware.TenantUUIDFromContext(c.Request.Context())
		if !ok || tenantID == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "tenant context missing"})
			return
		}
		required := []string(nil)
		if extractor != nil {
			required = extractor(c)
		}
		if err := guard.EnsureConsent(c.Request.Context(), tenantID, required); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.Next()
	}
}
