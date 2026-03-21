package runtime_ops

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	runtimeswitch "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/runtime/switches"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

const defaultDebugNotificationTopic = "org_sync.progress"

type wsBusTestNotificationRequest struct {
	Topic      string `json:"topic"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	TenantUUID string `json:"tenant_uuid"`
	TraceID    string `json:"trace_id"`
}

// WSBusTestNotificationHandler publishes a minimal notification event for WS bus E2E verification.
func WSBusTestNotificationHandler(deps *app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.WSBusHub == nil {
			contracts.ResponseServiceUnavailable(c, "ws bus is not configured", nil)
			return
		}

		var req wsBusTestNotificationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			if !errors.Is(err, io.EOF) {
				contracts.ResponseBadRequest(c, "invalid payload")
				return
			}
		}

		tenantUUID, tenantMismatch := resolveGatewayTenantUUID(c, deps, req.TenantUUID)
		if tenantMismatch {
			contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeTenantMismatch, "tenant mismatch")
			return
		}

		topic := strings.TrimSpace(req.Topic)
		if topic == "" {
			topic = defaultDebugNotificationTopic
		}
		traceID := strings.TrimSpace(req.TraceID)
		if traceID == "" {
			traceID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
		}

		now := time.Now().UTC()
		title := strings.TrimSpace(req.Title)
		if title == "" {
			title = "调试通知"
		}
		message := strings.TrimSpace(req.Message)
		if message == "" {
			message = "WS bus 测试通知"
		}

		payload := gin.H{
			"title":       title,
			"message":     message,
			"server_time": now.Format(time.RFC3339),
			"server_unix": now.Unix(),
		}

		publisher := fwwsbus.NewAdapter(
			fwwsbus.NewLocalPublisher(deps.WSBusHub, nil),
			"",
			nil,
		)
		outboundBearer := ""
		if runtimeswitch.IsHostDriver(runtimeswitch.Resolve(deps.Config).WSBus) && deps.Config != nil && deps.Config.Gateway != nil {
			outboundBearer = resolveGatewayBearerToken(c, deps)
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
				publisher = fwwsbus.NewAdapter(hostClient, "", nil)
			}
		}

		result := publisher.Publish(context.Background(), topic, payload, fwwsbus.PublishOptions{
			TenantUUID:  tenantUUID,
			TraceID:     traceID,
			BearerToken: outboundBearer,
		})
		if !result.OK {
			contracts.ResponseError(c, http.StatusBadRequest, result.ErrorCode, result.ErrorMessage)
			return
		}

		contracts.ResponseSuccess(c, gin.H{
			"ok":          true,
			"topic":       topic,
			"tenant_uuid": tenantUUID,
			"payload":     payload,
		})
	}
}
