package runtime_ops

import (
	"context"
	"net/http"
	"os"
	"strings"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	runtimeswitch "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/runtime/switches"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type wsBusGrantRequest struct {
	Topics     []string `json:"topics"`
	TenantUUID string   `json:"tenant_uuid"`
	TraceID    string   `json:"trace_id"`
}

func WSBusGrantHandler(deps *app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil {
			contracts.ResponseServiceUnavailable(c, "ws bus is not configured", nil)
			return
		}
		var req wsBusGrantRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid payload")
			return
		}
		topics, result := fwwsbus.ExpandTopicsForRegister(req.Topics)
		if !result.OK {
			contracts.ResponseError(c, http.StatusBadRequest, result.ErrorCode, result.ErrorMessage)
			return
		}

		tenantUUID, tenantMismatch := resolveGatewayTenantUUID(c, deps, req.TenantUUID)
		if tenantMismatch {
			contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeTenantMismatch, "tenant mismatch")
			return
		}
		traceID := strings.TrimSpace(req.TraceID)
		if traceID == "" {
			traceID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
		}

		if runtimeswitch.IsHostDriver(runtimeswitch.Resolve(deps.Config).WSBus) && deps.Config != nil && deps.Config.Gateway != nil {
			outboundBearer := resolveGatewayBearerToken(c, deps)
			logGatewayAuthSelection(c, deps, outboundBearer, tenantUUID)

			hostTenantUUID := strings.TrimSpace(deps.Config.Gateway.TenantUUID)
			if strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1" {
				hostTenantUUID = ""
			}
			hostClient, err := fwwsbus.NewHostClient(fwwsbus.HostClientConfig{
				BaseURL:    strings.TrimSpace(deps.Config.Gateway.BaseURL),
				APIPrefix:  strings.TrimSpace(deps.Config.Gateway.APIPrefix),
				AuthScheme: strings.TrimSpace(deps.Config.Gateway.AuthScheme),
				Token:      strings.TrimSpace(deps.Config.Gateway.ToolToken),
				APIKey:     strings.TrimSpace(deps.Config.Gateway.APIKey),
				TenantUUID: hostTenantUUID,
				UserAgent:  strings.TrimSpace(deps.Config.Gateway.UserAgent),
				Timeout:    deps.Config.Gateway.Timeout,
			})
			if err == nil {
				result = hostClient.RegisterTopics(context.Background(), topics, fwwsbus.PublishOptions{
					TenantUUID:  tenantUUID,
					TraceID:     traceID,
					BearerToken: outboundBearer,
				})
				if !result.OK {
					contracts.ResponseError(c, http.StatusBadRequest, result.ErrorCode, result.ErrorMessage)
					return
				}
				contracts.ResponseSuccess(c, gin.H{"ok": true, "topics": topics})
				return
			}
		}

		contracts.ResponseSuccess(c, gin.H{"ok": true, "topics": topics})
	}
}
