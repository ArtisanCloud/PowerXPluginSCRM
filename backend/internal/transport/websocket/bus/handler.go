package bus

import (
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handler struct {
	hub        *Hub
	authorizer Authorizer
}

func NewHandler() *Handler {
	return &Handler{
		hub:        DefaultHub,
		authorizer: NewDefaultAuthorizer(),
	}
}

var busUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) ServeWS(c *gin.Context) {
	tenantCtx, ok := authx.GetTenantContext(c)
	qTenant := strings.TrimSpace(c.Query("tenant_uuid"))
	authSource := strings.TrimSpace(c.GetString("ws_auth_source"))
	authQueryPresent := strings.TrimSpace(c.Query("authorization")) != ""
	protocolHeaderPresent := strings.TrimSpace(c.GetHeader("Sec-WebSocket-Protocol")) != ""
	if qTenant != "" {
		// 与 HTTP EnsureTenant 行为保持一致：允许显式 tenant_uuid 覆盖默认上下文（常见于本地 DevSwitch 场景）。
		tenantCtx.TenantUUID = qTenant
		ok = true
	}
	if !ok || strings.TrimSpace(tenantCtx.TenantUUID) == "" {
		// 兼容本地开发场景：当 JWT/DevSwitch 未注入 tenant_ctx 时，允许从 query 读取 tenant_uuid。
		if qTenant != "" {
			tenantCtx = authx.TenantContext{TenantUUID: qTenant}
		} else {
			logger.WithFields(logger.Fields{
				"component":               "ws_bus",
				"path":                    c.Request.URL.Path,
				"ws_auth_source":          authSource,
				"query_authorization_set": authQueryPresent,
				"protocol_header_set":     protocolHeaderPresent,
			}).Warn("ws handshake rejected: tenant_uuid required")
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "tenant_uuid required")
			return
		}
	}
	if strings.TrimSpace(tenantCtx.TenantUUID) == "" {
		logger.WithFields(logger.Fields{
			"component":               "ws_bus",
			"path":                    c.Request.URL.Path,
			"ws_auth_source":          authSource,
			"query_authorization_set": authQueryPresent,
			"protocol_header_set":     protocolHeaderPresent,
		}).Warn("ws handshake rejected: empty tenant_uuid")
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "tenant_uuid required")
		return
	}

	conn, err := busUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.WithFields(logger.Fields{
			"component":               "ws_bus",
			"tenant_uuid":             tenantCtx.TenantUUID,
			"path":                    c.Request.URL.Path,
			"ws_auth_source":          authSource,
			"query_authorization_set": authQueryPresent,
			"protocol_header_set":     protocolHeaderPresent,
		}).WithError(err).Warn("ws upgrade failed")
		return
	}

	client := NewClient(c.Request.Context(), conn, h.hub, h.authorizer)
	client.TenantUUID = tenantCtx.TenantUUID
	if tenantCtx.UserID > 0 {
		client.UserID = uint64(tenantCtx.UserID)
	}
	client.IsRoot = false
	logger.WithFields(logger.Fields{
		"component":               "ws_bus",
		"client_id":               client.ID,
		"tenant_uuid":             client.TenantUUID,
		"user_id":                 client.UserID,
		"path":                    c.Request.URL.Path,
		"query_tenant":            qTenant,
		"ws_auth_source":          authSource,
		"query_authorization_set": authQueryPresent,
		"protocol_header_set":     protocolHeaderPresent,
	}).Info("ws client connected")

	h.hub.Register(client)
	_ = sendWelcome(client)
	client.Run()
}

func sendWelcome(client *Client) error {
	if client == nil {
		return nil
	}
	env, err := dto.NewWSBusEnvelope(dto.WSBusTypeWelcome, "", dto.WSBusWelcomePayload{
		Protocol:     "px.ws.bus.v1",
		Server:       "powerx-plugin-ws-bus",
		HeartbeatSec: 25,
	}, "")
	if err != nil {
		return err
	}
	client.sendEnvelope(env)
	return nil
}
