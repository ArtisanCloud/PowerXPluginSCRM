package bus

import (
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto"
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
	if !ok || strings.TrimSpace(tenantCtx.TenantUUID) == "" {
		// 兼容本地开发场景：当 JWT/DevSwitch 未注入 tenant_ctx 时，允许从 query 读取 tenant_uuid。
		if qTenant := strings.TrimSpace(c.Query("tenant_uuid")); qTenant != "" {
			tenantCtx = authx.TenantContext{TenantUUID: qTenant}
		} else {
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "tenant_uuid required")
			return
		}
	}
	if strings.TrimSpace(tenantCtx.TenantUUID) == "" {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "tenant_uuid required")
		return
	}

	conn, err := busUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := NewClient(c.Request.Context(), conn, h.hub, h.authorizer)
	client.TenantUUID = tenantCtx.TenantUUID
	if tenantCtx.UserID > 0 {
		client.UserID = uint64(tenantCtx.UserID)
	}
	client.IsRoot = false

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
