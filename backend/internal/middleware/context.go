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
	TenantID      int64    `json:"tenant_id"`
	UserID        int64    `json:"user_id"`
	UserUUID      string   `json:"user_uuid"`
	MemberID      int64    `json:"member_id"`
	MemberUUID    string   `json:"member_uuid"`
	Roles         []string `json:"roles"`
	Permissions   []string `json:"permissions"`
	PolicyVersion string   `json:"policy_version"`
	PluginID      string   `json:"plugin_id"`
}

const (
	ctxKeyTenant     = "tenant_ctx"
	ctxKeyToken      = "raw_bearer_token"
	ctxKeyUserUUID   = "user_uuid"
	ctxKeyUserID     = "user_id"
	ctxKeyMemberUUID = "member_uuid"
	ctxKeyMemberID   = "member_id"
)

type (
	tenantUUIDContextKey struct{}
	userUUIDContextKey   struct{}
	userIDContextKey     struct{}
	memberUUIDContextKey struct{}
	memberIDContextKey   struct{}
)

var (
	ctxKeyTenantUUIDValue = tenantUUIDContextKey{}
	ctxKeyUserUUIDValue   = userUUIDContextKey{}
	ctxKeyUserIDValue     = userIDContextKey{}
	ctxKeyMemberUUIDValue = memberUUIDContextKey{}
	ctxKeyMemberIDValue   = memberIDContextKey{}
)

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
	return context.WithValue(ctx, ctxKeyTenantUUIDValue, strings.TrimSpace(tenantUUID))
}

// TenantUUIDFromContext extracts tenant UUID from a standard context.
func TenantUUIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if v := ctx.Value(ctxKeyTenantUUIDValue); v != nil {
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

func ContextWithUserUUID(ctx context.Context, userUUID string) context.Context {
	if ctx == nil || strings.TrimSpace(userUUID) == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyUserUUIDValue, strings.TrimSpace(userUUID))
}

func UserUUIDFromContext(ctx context.Context) (string, bool) {
	return stringFromContext(ctx, ctxKeyUserUUIDValue)
}

func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	if ctx == nil || userID <= 0 {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyUserIDValue, userID)
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	return int64FromContext(ctx, ctxKeyUserIDValue)
}

func ContextWithMemberUUID(ctx context.Context, memberUUID string) context.Context {
	if ctx == nil || strings.TrimSpace(memberUUID) == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyMemberUUIDValue, strings.TrimSpace(memberUUID))
}

func MemberUUIDFromContext(ctx context.Context) (string, bool) {
	return stringFromContext(ctx, ctxKeyMemberUUIDValue)
}

func ContextWithMemberID(ctx context.Context, memberID int64) context.Context {
	if ctx == nil || memberID <= 0 {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyMemberIDValue, memberID)
}

func MemberIDFromContext(ctx context.Context) (int64, bool) {
	return int64FromContext(ctx, ctxKeyMemberIDValue)
}

func ContextWithTenantIdentity(ctx context.Context, tc TenantContext) context.Context {
	ctx = ContextWithTenantUUID(ctx, tc.TenantUUID)
	ctx = ContextWithUserUUID(ctx, tc.UserUUID)
	ctx = ContextWithUserID(ctx, tc.UserID)
	ctx = ContextWithMemberUUID(ctx, tc.MemberUUID)
	ctx = ContextWithMemberID(ctx, tc.MemberID)
	return ctx
}

func stringFromContext(ctx context.Context, key any) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if v := ctx.Value(key); v != nil {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s), true
		}
	}
	return "", false
}

func int64FromContext(ctx context.Context, key any) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	if v := ctx.Value(key); v != nil {
		switch id := v.(type) {
		case int64:
			return id, id > 0
		case uint64:
			if id > 0 {
				return int64(id), true
			}
		case int:
			if id > 0 {
				return int64(id), true
			}
		case string:
			parsed, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
			return parsed, err == nil && parsed > 0
		}
	}
	return 0, false
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
