package runtime_ops

import (
	"context"
	"net/http"
	"os"
	"strings"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type wsBusPublishRequest struct {
	Topic      string `json:"topic"`
	Payload    any    `json:"payload"`
	TenantUUID string `json:"tenant_uuid"`
	TraceID    string `json:"trace_id"`
}

// WSBusPublishHandler provides a debug publish endpoint for standalone mode.
func WSBusPublishHandler(deps *app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.WSBusHub == nil {
			contracts.ResponseServiceUnavailable(c, "ws bus is not configured", nil)
			return
		}
		var req wsBusPublishRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid payload")
			return
		}

		tenantUUID := resolveGatewayTenantUUID(c, deps, req.TenantUUID)
		traceID := strings.TrimSpace(req.TraceID)
		if traceID == "" {
			traceID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
		}

		publisher := fwwsbus.NewAdapter(
			fwwsbus.NewLocalPublisher(deps.WSBusHub, nil),
			"",
			nil,
		)
		outboundBearer := ""
		if os.Getenv("POWERX_PROXY") == "1" && deps.Config != nil && deps.Config.Gateway != nil {
			outboundBearer = resolveGatewayBearerToken(c, deps)
			logGatewayAuthSelection(c, deps, outboundBearer, tenantUUID)

			baseURL := strings.TrimSpace(deps.Config.Gateway.BaseURL)
			if strings.HasSuffix(baseURL, "/api/v1") {
				baseURL = strings.TrimSuffix(baseURL, "/api/v1")
			}
			hostClient, err := fwwsbus.NewHostClient(fwwsbus.HostClientConfig{
				BaseURL:    baseURL,
				Token:      strings.TrimSpace(deps.Config.Gateway.ToolToken),
				TenantUUID: strings.TrimSpace(deps.Config.Gateway.TenantUUID),
				UserAgent:  strings.TrimSpace(deps.Config.Gateway.UserAgent),
				Timeout:    deps.Config.Gateway.Timeout,
			})
			if err == nil {
				publisher = fwwsbus.NewAdapter(hostClient, "", nil)
			}
		}
		result := publisher.Publish(context.Background(), req.Topic, req.Payload, fwwsbus.PublishOptions{
			TenantUUID:  tenantUUID,
			TraceID:     traceID,
			BearerToken: outboundBearer,
		})

		if !result.OK {
			contracts.ResponseError(c, http.StatusBadRequest, result.ErrorCode, result.ErrorMessage)
			return
		}
		contracts.ResponseSuccess(c, gin.H{"ok": true})
	}
}
