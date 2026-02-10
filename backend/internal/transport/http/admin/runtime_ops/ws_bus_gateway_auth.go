package runtime_ops

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// resolveGatewayBearerToken 统一 ws-bus 出站 token 选择规则：
// - Delegated/宿主模式：透传入站 Bearer（与宿主请求链路保持一致）
// - Local/Standalone 模式：不透传，交由 HostClient 使用 PX_TOOL_TOKEN
func resolveGatewayBearerToken(c *gin.Context, deps *app.Deps) string {
	if c == nil || deps == nil {
		return ""
	}
	if deps.IAMMode != iamservice.IAMModeDelegated {
		return ""
	}
	if raw, ok := middleware.GetRawBearerToken(c); ok {
		return strings.TrimSpace(raw)
	}
	return ""
}

// resolveGatewayTenantUUID 统一 ws-bus 出站 tenant 选择规则：
// 1) 请求体显式 tenant_uuid 优先；
// 2) 入站请求 token/上下文租户（两种 IAM 模式都可用）；
// 3) Local/Standalone 模式回退 PX_TOOL_TOKEN.tid；
// 4) 最后回退 gateway.tenant_uuid（兼容旧配置）。
func resolveGatewayTenantUUID(c *gin.Context, deps *app.Deps, requested string) string {
	tenantUUID := strings.TrimSpace(requested)
	if tenantUUID != "" {
		return tenantUUID
	}

	if c != nil {
		if tc, ok := middleware.GetTenantContext(c); ok {
			tenantUUID = strings.TrimSpace(tc.TenantUUID)
			if tenantUUID != "" {
				return tenantUUID
			}
		}
	}

	if deps != nil && deps.Config != nil && deps.Config.Gateway != nil {
		configuredTenant := strings.TrimSpace(deps.Config.Gateway.TenantUUID)
		if deps.IAMMode == iamservice.IAMModeLocal {
			if tokenTenant := tenantUUIDFromJWT(strings.TrimSpace(deps.Config.Gateway.ToolToken)); tokenTenant != "" {
				if configuredTenant != "" && configuredTenant != tokenTenant {
					logger.WithFields(logger.Fields{
						"component":         "ws_bus_gateway_auth",
						"iam_mode":          deps.IAMMode,
						"configured_tenant": configuredTenant,
						"token_tenant":      tokenTenant,
					}).Warn("gateway.tenant_uuid 与 PX_TOOL_TOKEN.tid 不一致，已优先使用 token 租户")
				}
				return tokenTenant
			}
		}
		return configuredTenant
	}

	return ""
}

func tenantUUIDFromJWT(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	tid, _ := claims["tid"].(string)
	return strings.TrimSpace(tid)
}

// logGatewayAuthSelection 输出 ws-bus 出站鉴权选择，便于联调观察 token 来源。
func logGatewayAuthSelection(c *gin.Context, deps *app.Deps, outboundBearer string, tenantUUID string) {
	if deps == nil || deps.Config == nil || deps.Config.Server == nil || !deps.Config.Server.DevMode {
		return
	}

	inboundBearerPresent := false
	inboundBearerPrefix := ""
	if c != nil {
		if raw, ok := middleware.GetRawBearerToken(c); ok {
			raw = strings.TrimSpace(raw)
			if raw != "" {
				inboundBearerPresent = true
				inboundBearerPrefix = tokenPrefix(raw)
			}
		}
	}

	pxToolToken := ""
	if deps.Config.Gateway != nil {
		pxToolToken = strings.TrimSpace(deps.Config.Gateway.ToolToken)
	}

	outboundSource := "PX_TOOL_TOKEN"
	if strings.TrimSpace(outboundBearer) != "" {
		outboundSource = "request_bearer_passthrough"
	}

	logger.WithFields(logger.Fields{
		"component":               "ws_bus_gateway_auth",
		"iam_mode":                deps.IAMMode,
		"inbound_bearer_present":  inboundBearerPresent,
		"inbound_bearer_prefix":   inboundBearerPrefix,
		"outbound_token_source":   outboundSource,
		"outbound_bearer_prefix":  tokenPrefix(outboundBearer),
		"px_tool_token_present":   pxToolToken != "",
		"px_tool_token_prefix":    tokenPrefix(pxToolToken),
		"resolved_gateway_tenant": strings.TrimSpace(tenantUUID),
	}).Info("WS bus gateway auth resolved")
}

func tokenPrefix(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 16 {
		return token
	}
	return token[:16] + "..."
}
