package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	authobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/auth"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RBAC 仅做粗粒度权限判定；在 DelegateToPowerX 模式下只校验令牌来源
func RBAC(cfg *authx.RBACConfig, _ authx.ABACClient, _ func(string, string) (bool, map[string]any)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg == nil || !cfg.Enabled || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		// 内部回调放行（仍建议结合网络/签名防护）
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/internal/") || strings.HasPrefix(c.Request.URL.Path, "/api/v1/agent/") {
			c.Next()
			return
		}
		if isHealthEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		if cfg.DelegateToPowerX {
			if allowPowerXDelegate(c, cfg) {
				c.Next()
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "PowerX delegated authentication required",
					"hint":  "请在宿主 PowerX 登录后再访问插件，或设置 POWERX_PROXY=0 运行 Standalone 模式",
				})
			}
			return
		}

		tc, ok := authx.GetTenantContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Role Authentication required"})
			return
		}
		if authx.IsSuperAdmin(tc.Roles, cfg.SuperAdminRoles) {
			c.Next()
			return
		}

		full := c.FullPath()
		if full == "" {
			full = c.Request.URL.Path
		}
		perm, has := authx.MatchRoute(c.Request.Method, full, cfg.RoutePermissions)
		if has {
			perm = cfg.NormalizePermission(perm)
		} else if inferred, ok := authx.InferPermission(c.Request.Method, c.Request.URL.Path); ok {
			perm = cfg.NormalizePermission(inferred)
			has = true
		}
		passRBAC := (!has && !cfg.DefaultDeny) || (has && authx.HasPerm(tc.Permissions, perm))
		if !passRBAC {
			resource := perm.Resource
			action := perm.Action
			if strings.TrimSpace(resource) == "" {
				resource = "unknown"
			}
			if strings.TrimSpace(action) == "" {
				action = "unknown"
			}
			authobs.RecordRBACDenied(app.PluginID, resource, action)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":             "Insufficient permissions",
				"required_resource": perm.Resource,
				"required_action":   perm.Action,
			})
			return
		}

		c.Next()
	}
}

func allowPowerXDelegate(c *gin.Context, cfg *authx.RBACConfig) bool {
	if cfg == nil {
		return false
	}
	log.Printf("[PLUGIN-RBAC] delegate check: need{iss=%s aud=%s}", cfg.PowerXIssuer, cfg.PowerXAudience)
	if _, ok := authx.GetTenantContext(c); !ok {
		return false
	}
	raw, ok := authx.GetRawBearerToken(c)
	if !ok || strings.TrimSpace(raw) == "" {
		return false
	}
	claims, err := decodeJWTClaims(raw)
	if err != nil {
		return false
	}
	log.Printf("[PLUGIN-RBAC] delegate token: iss=%v aud=%v", claims["iss"], claims["aud"])

	if cfg.PowerXIssuer != "" {
		if iss, _ := claims["iss"].(string); iss != cfg.PowerXIssuer {
			return false
		}
	}
	if cfg.PowerXAudience != "" && !audienceMatches(claims["aud"], cfg.PowerXAudience) {
		return false
	}
	return true
}

func decodeJWTClaims(raw string) (map[string]any, error) {
	parts := strings.Split(raw, ".")
	if len(parts) < 2 {
		return nil, errInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func audienceMatches(aud any, expected string) bool {
	switch v := aud.(type) {
	case string:
		return v == expected
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == expected {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if s == expected {
				return true
			}
		}
	}
	return false
}

var errInvalidToken = errors.New("invalid token")

func isHealthEndpoint(path string) bool {
	p := strings.ToLower(strings.TrimSpace(path))
	if p == "" {
		return false
	}
	if strings.HasPrefix(p, "/healthz") {
		return true
	}
	if strings.HasPrefix(p, "/api/v1/admin/runtime/metrics") {
		return true
	}
	return false
}
