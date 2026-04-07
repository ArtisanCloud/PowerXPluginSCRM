package middleware

import (
	"net/http"
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// EnsureRootRole restricts platform-level operations to root/system-admin identities.
func EnsureRootRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		tc, ok := authx.GetTenantContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
			return
		}
		if isRootRole(tc) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "root role required",
		})
	}
}

func isRootRole(tc authx.TenantContext) bool {
	for _, role := range tc.Roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "root", "superadmin", "admin", "system.admin":
			return true
		}
	}
	for _, perm := range tc.Permissions {
		candidate := strings.ToLower(strings.TrimSpace(perm))
		if candidate == "*" || candidate == "*:*" {
			return true
		}
	}
	return false
}
