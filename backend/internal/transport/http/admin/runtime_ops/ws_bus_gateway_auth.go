package runtime_ops

import (
	"os"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// resolveGatewayBearerToken returns request-scoped bearer when delegated IAM provides one.
func resolveGatewayBearerToken(c *gin.Context, deps *app.Deps) string {
	if c == nil || deps == nil {
		return ""
	}
	if deps.Config != nil && deps.Config.Gateway != nil {
		authScheme := strings.ToLower(strings.TrimSpace(deps.Config.Gateway.AuthScheme))
		apiKey := strings.TrimSpace(deps.Config.Gateway.APIKey)
		if authScheme == "apikey" || authScheme == "api_key" || authScheme == "api-key" {
			return ""
		}
		if authScheme == "" && apiKey != "" {
			return ""
		}
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
// 1) proxy 模式：tenant 由宿主按凭证解析，插件侧不透传 tenant_uuid；
// 2) 非 proxy 模式：请求体 tenant_uuid 与入站租户需一致；
// 3) 非 proxy 模式：优先入站租户；请求体为空时回退入站租户。
func resolveGatewayTenantUUID(c *gin.Context, _ *app.Deps, requested string) (tenantUUID string, mismatch bool) {
	if os.Getenv("POWERX_PROXY") == "1" {
		return "", false
	}

	requested = strings.TrimSpace(requested)
	inboundTenant := ""
	if c != nil {
		if tc, ok := middleware.GetTenantContext(c); ok {
			inboundTenant = strings.TrimSpace(tc.TenantUUID)
		}
	}

	if requested != "" && inboundTenant != "" && requested != inboundTenant {
		return "", true
	}
	if requested != "" {
		return requested, false
	}
	if inboundTenant != "" {
		return inboundTenant, false
	}
	return "", false
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

	apiKey := ""
	authScheme := ""
	if deps.Config.Gateway != nil {
		apiKey = strings.TrimSpace(deps.Config.Gateway.APIKey)
		authScheme = strings.TrimSpace(deps.Config.Gateway.AuthScheme)
	}

	outboundSource := "sts"
	if strings.EqualFold(authScheme, "apikey") || strings.EqualFold(authScheme, "api_key") || strings.EqualFold(authScheme, "api-key") {
		outboundSource = "PX_GATEWAY_API_KEY"
	}
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
		"px_gateway_api_key_set":  apiKey != "",
		"gateway_auth_scheme":     authScheme,
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
