package middleware

import (
	"net/http"
	"strconv"
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TenantUUIDContextKey = "tenant_uuid"

// EnsureTenant ensures a valid tenant exists on the request and propagates it through contexts.
func EnsureTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if tenantUUID, ok := resolveTenantUUID(c); ok && tenantUUID != "" {
			attachTenant(c, tenantUUID)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "tenant context missing",
		})
	}
}

func resolveTenantUUID(c *gin.Context) (string, bool) {
	if id, ok := authx.TenantUUIDFromContext(c.Request.Context()); ok && id != "" {
		return id, true
	}

	if id, ok := tenantUUIDFromGinState(c); ok && id != "" {
		return id, true
	}

	if tc, ok := authx.GetTenantContext(c); ok && strings.TrimSpace(tc.TenantUUID) != "" {
		return strings.TrimSpace(tc.TenantUUID), true
	}

	if id, ok := parseTenantUUID(c.GetHeader("X-Tenant-UUID")); ok {
		return id, true
	}

	if id, ok := parseTenantUUID(c.Query("tenant_uuid")); ok {
		return id, true
	}

	return "", false
}

func tenantUUIDFromGinState(c *gin.Context) (string, bool) {
	if id, ok := c.Get(TenantUUIDContextKey); ok {
		switch v := id.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v), true
			}
		case uint64:
			if v > 0 {
				return strconv.FormatUint(v, 10), true
			}
		case int64:
			if v > 0 {
				return strconv.FormatInt(v, 10), true
			}
		case int:
			if v > 0 {
				return strconv.Itoa(v), true
			}
		}
	}
	return "", false
}

func parseTenantUUID(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if _, err := uuid.Parse(raw); err != nil {
		return "", false
	}
	return strings.ToLower(raw), true
}

func attachTenant(c *gin.Context, tenantUUID string) {
	if strings.TrimSpace(tenantUUID) == "" {
		return
	}

	c.Set(TenantUUIDContextKey, tenantUUID)

	if tc, ok := authx.GetTenantContext(c); ok {
		if strings.TrimSpace(tc.TenantUUID) == "" {
			tc.TenantUUID = tenantUUID
			authx.SetTenantContext(c, tc)
		}
	} else {
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
	}

	ctx := authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID)
	if ctx != nil {
		c.Request = c.Request.WithContext(ctx)
	}
}

// TenantUUIDFromContext returns the resolved tenant ID if present.
func TenantUUIDFromContext(c *gin.Context) (string, bool) {
	if id, ok := authx.TenantUUIDFromContext(c.Request.Context()); ok && id != "" {
		return id, true
	}
	if id, ok := tenantUUIDFromGinState(c); ok && id != "" {
		return id, true
	}
	if tc, ok := authx.GetTenantContext(c); ok && strings.TrimSpace(tc.TenantUUID) != "" {
		return strings.TrimSpace(tc.TenantUUID), true
	}
	return "", false
}

// TenantUUIDString returns tenant id as string if present.
func TenantUUIDString(c *gin.Context) (string, bool) {
	return TenantUUIDFromContext(c)
}

// TenantUuidString is a deprecated helper kept for backward compatibility with handlers
// that still expect a numeric tenant identifier. It now simply returns the resolved
// tenant UUID string; callers should migrate to TenantUUIDString.
func TenantUuidString(c *gin.Context) (string, bool) {
	return TenantUUIDString(c)
}
