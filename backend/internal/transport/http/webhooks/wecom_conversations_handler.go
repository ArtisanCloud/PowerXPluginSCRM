package webhooks

import (
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/gin-gonic/gin"
)

type WeComConversationWebhookHandler struct {
	svc *leadsvc.ConversationService
}

func NewWeComConversationWebhookHandler(svc *leadsvc.ConversationService) *WeComConversationWebhookHandler {
	return &WeComConversationWebhookHandler{svc: svc}
}

type WeComConversationWebhookRequest struct {
	TenantUUID         string                 `json:"tenant_uuid"`
	ChannelAccountUUID string                 `json:"channel_account_uuid"`
	ExternalEventID    string                 `json:"external_event_id"`
	ConversationID     string                 `json:"conversation_id"`
	ActorType          string                 `json:"actor_type"`
	ActorID            string                 `json:"actor_id"`
	Direction          string                 `json:"direction"`
	MessageType        string                 `json:"message_type"`
	ContentText        string                 `json:"content_text"`
	OccurredAt         string                 `json:"occurred_at"`
	RawPayload         map[string]any         `json:"raw_payload"`
	Metadata           map[string]interface{} `json:"metadata"`
}

func (h *WeComConversationWebhookHandler) Ingest(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "conversation service unavailable", nil)
		return
	}
	var req WeComConversationWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID := strings.TrimSpace(req.TenantUUID)
	occurredAt := time.Now().UTC()
	if raw := strings.TrimSpace(req.OccurredAt); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			occurredAt = parsed
		}
	}
	eventUUID, created, err := h.svc.IngestWebhook(c.Request.Context(), leadsvc.ConversationWebhookInput{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: strings.TrimSpace(req.ChannelAccountUUID),
		ExternalEventID:    strings.TrimSpace(req.ExternalEventID),
		ConversationID:     strings.TrimSpace(req.ConversationID),
		ActorType:          strings.TrimSpace(req.ActorType),
		ActorID:            strings.TrimSpace(req.ActorID),
		Direction:          strings.TrimSpace(req.Direction),
		MessageType:        strings.TrimSpace(req.MessageType),
		ContentText:        strings.TrimSpace(req.ContentText),
		OccurredAt:         occurredAt,
		RawPayload:         req.RawPayload,
	})
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "event_uuid": eventUUID, "created": created})
}
