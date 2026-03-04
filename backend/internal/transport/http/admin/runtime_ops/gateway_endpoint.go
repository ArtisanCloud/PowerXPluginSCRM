package runtime_ops

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
)

func buildGatewayEndpoint(gateway *config.GatewayConfig, routePath string) string {
	if gateway == nil {
		return ""
	}
	base := strings.TrimRight(strings.TrimSpace(gateway.BaseURL), "/")
	route := "/" + strings.TrimLeft(strings.TrimSpace(routePath), "/")
	prefix := normalizeGatewayAPIPrefix(strings.TrimSpace(gateway.APIPrefix))
	if strings.HasSuffix(base, prefix) {
		return base + route
	}
	return base + prefix + route
}

func normalizeGatewayAPIPrefix(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "/api/v1"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = "/" + strings.Trim(strings.TrimSpace(value), "/")
	if value == "/" {
		return "/api/v1"
	}
	return value
}
