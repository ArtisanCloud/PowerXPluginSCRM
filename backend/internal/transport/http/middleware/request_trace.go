package middleware

import (
	"log"
	"os"
	"strings"
	"time"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RequestTrace 输出请求关键信息，辅助排查网关/本地两种模式的差异。
func RequestTrace() gin.HandlerFunc {
	if !traceEnabled() {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	mode := requestMode()
	iamMode := iamModeFromEnv()
	return func(c *gin.Context) {
		start := time.Now()

		authMode, authPreview := detectAuth(c)
		userAgent := shorten(c.GetHeader("User-Agent"), 80)
		traceID := traceIdentifier(c)
		tenantCtx, _ := authx.GetTenantContext(c)

		log.Printf("[PLUGIN-REQ-TRACE] stage=begin mode=%s iam_mode=%s method=%s path=%s auth=%s auth.head=%s tenant_uuid=%s user_id=%d trace=%s ip=%s ua=%s",
			mode,
			iamMode,
			c.Request.Method,
			c.Request.URL.Path,
			authMode,
			authPreview,
			tenantCtx.TenantUUID,
			tenantCtx.UserID,
			traceID,
			c.ClientIP(),
			userAgent,
		)

		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)
		if raw, ok := authx.GetRawBearerToken(c); ok && raw != "" {
			authPreview = shorten(raw, 40)
			authMode = "bearer(validated)"
		}

		log.Printf("[PLUGIN-REQ-TRACE] stage=end mode=%s iam_mode=%s status=%d latency=%s auth=%s auth.head=%s tenant_uuid=%s user_id=%d trace=%s",
			mode,
			iamMode,
			status,
			latency,
			authMode,
			authPreview,
			tenantCtx.TenantUUID,
			tenantCtx.UserID,
			traceID,
		)
	}
}

func traceEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("POWERX_DEBUG_TRAFFIC")))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	// 默认：PowerX 宿主关闭，独立模式开启
	return os.Getenv("POWERX_PROXY") != "1"
}

func requestMode() string {
	if os.Getenv("POWERX_PROXY") == "1" {
		return "powerx-proxy"
	}
	return "standalone"
}

func detectAuth(c *gin.Context) (mode, preview string) {
	auth := c.GetHeader("Authorization")
	if auth != "" {
		return "bearer", shorten(auth, 40)
	}
	if ctx := c.GetHeader("X-PowerX-CTX"); ctx != "" {
		return "signed_ctx", shorten(ctx, 40)
	}
	return "none", ""
}

func iamModeFromEnv() string {
	if truthy(os.Getenv("POWERX_RBAC_DELEGATE")) || strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1" {
		return "delegated"
	}
	return "local"
}

func traceIdentifier(c *gin.Context) string {
	if id := strings.TrimSpace(c.GetHeader("X-Request-ID")); id != "" {
		return id
	}
	if id := strings.TrimSpace(c.GetHeader("Request-ID")); id != "" {
		return id
	}
	if v := strings.TrimSpace(c.GetString("request_id")); v != "" {
		return v
	}
	return ""
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func shorten(raw string, keep int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) <= keep {
		return raw
	}
	return raw[:keep] + "..."
}
