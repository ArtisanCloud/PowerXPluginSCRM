package lead_capture

import (
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type ConversationHandler struct {
	svc *leadsvc.ConversationService
}

func NewConversationHandler(svc *leadsvc.ConversationService) *ConversationHandler {
	return &ConversationHandler{svc: svc}
}

type BindConversationRequest struct {
	ConversationID     string `json:"conversation_id" binding:"required"`
	ChannelAccountUUID string `json:"channel_account_uuid" binding:"required"`
}

func (h *ConversationHandler) ListLeadConversations(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "conversation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	items, err := h.svc.ListLeadConversations(c.Request.Context(), tenantUUID, leadUUID)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"lead_id": leadUUID, "conversations": items})
}

func (h *ConversationHandler) ListConversationEvents(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "conversation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	conversationID := strings.TrimSpace(c.Param("conversation_id"))
	if conversationID == "" {
		contracts.ResponseBadRequest(c, "conversation_id is required")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.svc.ListConversationEvents(c.Request.Context(), tenantUUID, conversationID, limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"conversation_id": conversationID, "events": items})
}

func (h *ConversationHandler) BindConversation(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "conversation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	var req BindConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	actorUserUUID := strings.TrimSpace(c.GetString("user_uuid"))
	if err := h.svc.BindConversation(c.Request.Context(), tenantUUID, leadUUID, req.ConversationID, req.ChannelAccountUUID, actorUserUUID); err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}
