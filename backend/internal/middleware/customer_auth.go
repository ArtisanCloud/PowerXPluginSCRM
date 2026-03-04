package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	customermw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware/customer"
	customerobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/customer"
	customersvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/customer"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CustomerAuth enforces customer authentication for /mini-app routes.
// 租户优先从上下文或 query 解析；认证成功后会把解析出的 tenant_uuid 注入上下文，
// 以便后续 EnsureTenant() 正确识别租户。
func CustomerAuth(authenticator customersvc.Authenticator, audit *customerobs.AuditLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		requestTenantUUID, _ := TenantUUIDFromContext(c.Request.Context())
		requestTenantUUID = strings.ToLower(strings.TrimSpace(requestTenantUUID))
		if requestTenantUUID == "" {
			if raw := strings.TrimSpace(c.Query("tenant_uuid")); raw != "" {
				if _, err := uuid.Parse(raw); err == nil {
					requestTenantUUID = strings.ToLower(raw)
				}
			}
		}

		token, ok := extractCustomerToken(c)
		if !ok {
			contracts.ResponseUnauthorized(c, "customer token missing")
			c.Abort()
			return
		}

		start := time.Now()
		cc, err := authenticator.Authenticate(c.Request.Context(), requestTenantUUID, token)
		latency := time.Since(start)

		if err != nil {
			customerobs.RecordValidation("", "unknown", "error", latency)
			if audit != nil {
				audit.LogValidation(requestTenantUUID, "", "", false, latency, err)
			}

			switch {
			case errors.Is(err, customersvc.ErrCustomerAuthNotImplemented):
				contracts.ResponseServiceUnavailable(c, "customer auth not implemented", map[string]any{
					"mode": "unknown",
				})
			case errors.Is(err, customersvc.ErrCustomerDelegateUnavailable):
				contracts.ResponseServiceUnavailable(c, "customer auth delegate unavailable", nil)
			default:
				contracts.ResponseUnauthorized(c, "customer token invalid")
			}
			c.Abort()
			return
		}

		if cc == nil || !cc.Authenticated {
			customerobs.RecordValidation("", "unknown", "unauthorized", latency)
			if audit != nil {
				audit.LogValidation(requestTenantUUID, "", "", false, latency, errors.New("customer not authenticated"))
			}
			contracts.ResponseUnauthorized(c, "customer token invalid")
			c.Abort()
			return
		}

		resolvedTenantUUID := strings.ToLower(strings.TrimSpace(cc.TenantUUID))
		if requestTenantUUID != "" && resolvedTenantUUID != "" && resolvedTenantUUID != requestTenantUUID {
			customerobs.RecordValidation("", string(cc.SourceMode), "tenant_mismatch", latency)
			if audit != nil {
				audit.LogValidation(requestTenantUUID, cc.CustomerUUID, string(cc.SourceMode), false, latency, errors.New("tenant mismatch"))
			}
			contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeTenantMismatch, "customer tenant mismatch")
			c.Abort()
			return
		}
		if resolvedTenantUUID == "" && requestTenantUUID != "" {
			resolvedTenantUUID = requestTenantUUID
			cc.TenantUUID = requestTenantUUID
		}

		// ✅ 把 tenant_uuid 注入到 request context + gin state，供 EnsureTenant() 与后续 repo/service 使用。
		if resolvedTenantUUID != "" {
			ctx := ContextWithTenantUUID(c.Request.Context(), resolvedTenantUUID)
			c.Request = c.Request.WithContext(ctx)
			c.Set("tenant_uuid", resolvedTenantUUID)
		}

		customermw.SetCustomerContext(c, cc)

		customerobs.RecordValidation("", string(cc.SourceMode), "ok", latency)
		if audit != nil {
			audit.LogValidation(resolvedTenantUUID, cc.CustomerUUID, string(cc.SourceMode), true, latency, nil)
		}
		c.Next()
	}
}

func extractCustomerToken(c *gin.Context) (string, bool) {
	raw := strings.TrimSpace(c.GetHeader("Authorization"))
	if raw != "" {
		lower := strings.ToLower(raw)
		if strings.HasPrefix(lower, "bearer ") {
			tok := strings.TrimSpace(raw[len("Bearer "):])
			if tok != "" {
				return tok, true
			}
		}
	}
	if tok := strings.TrimSpace(c.GetHeader("X-Customer-Token")); tok != "" {
		return tok, true
	}
	return "", false
}
