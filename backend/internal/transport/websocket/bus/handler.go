package bus

import (
	"net/http"

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
	if !ok || tenantCtx.TenantUUID == "" {
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
