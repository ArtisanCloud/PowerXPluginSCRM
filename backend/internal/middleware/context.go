package middleware

// internal/middleware/context.go

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type TenantContext struct {
	TenantUUID    string   `json:"tenant_uuid"`
	UserID        int64    `json:"user_id"`
	Roles         []string `json:"roles"`
	Permissions   []string `json:"permissions"`
	PolicyVersion string   `json:"policy_version"`
	PluginID      string   `json:"plugin_id"`
}

const (
	ctxKeyTenant = "tenant_ctx"
	ctxKeyToken  = "raw_bearer_token"
)

type tenantUUIDContextKey struct{}

var ctxKeyTenantUUID = tenantUUIDContextKey{}

var ErrTenantMissing = errors.New("tenant context missing")

func SetTenantContext(c *gin.Context, tc TenantContext) { c.Set(ctxKeyTenant, tc) }
func GetTenantContext(c *gin.Context) (TenantContext, bool) {
	v, ok := c.Get(ctxKeyTenant)
	if !ok || v == nil {
		return TenantContext{}, false
	}
	tc, ok := v.(TenantContext)
	return tc, ok
}
func SetRawBearerToken(c *gin.Context, token string) {
	if token != "" {
		c.Set(ctxKeyToken, token)
	}
}
func GetRawBearerToken(c *gin.Context) (string, bool) {
	v, ok := c.Get(ctxKeyToken)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

// ContextWithTenantUUID stores tenant UUID into a standard context.
func ContextWithTenantUUID(ctx context.Context, tenantUUID string) context.Context {
	if ctx == nil || strings.TrimSpace(tenantUUID) == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyTenantUUID, strings.TrimSpace(tenantUUID))
}

// TenantUUIDFromContext extracts tenant UUID from a standard context.
func TenantUUIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if v := ctx.Value(ctxKeyTenantUUID); v != nil {
		switch id := v.(type) {
		case string:
			if strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id), true
			}
		case uint64:
			if id > 0 {
				return strconv.FormatUint(id, 10), true
			}
		case int64:
			if id > 0 {
				return strconv.FormatInt(id, 10), true
			}
		case int:
			if id > 0 {
				return strconv.Itoa(id), true
			}
		}
	}
	return "", false
}

// RequireTenantUUID retrieves tenant UUID from context or returns ErrTenantMissing.
func RequireTenantUUID(ctx context.Context) (string, error) {
	if tenantUUID, ok := TenantUUIDFromContext(ctx); ok && tenantUUID != "" {
		return tenantUUID, nil
	}
	return "", ErrTenantMissing
}

// Deprecated compatibility helpers —— convert numeric IDs into UUID strings if possible.
func ContextWithTenantUuid(ctx context.Context, tenantID uint64) context.Context {
	if tenantID == 0 {
		return ctx
	}
	return ContextWithTenantUUID(ctx, strconv.FormatUint(tenantID, 10))
}

func TenantUuidFromContext(ctx context.Context) (uint64, bool) {
	if uuidVal, ok := TenantUUIDFromContext(ctx); ok && uuidVal != "" {
		if num, err := strconv.ParseUint(uuidVal, 10, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}

func RequireTenantUuid(ctx context.Context) (uint64, error) {
	if id, ok := TenantUuidFromContext(ctx); ok && id > 0 {
		return id, nil
	}
	return 0, ErrTenantMissing
}
