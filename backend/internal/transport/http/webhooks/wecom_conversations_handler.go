package webhooks

import (
	"errors"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WeComConversationWebhookHandler struct {
	svc         *leadsvc.ConversationService
	accountRepo *socialrepo.AccountRepository
}

func NewWeComConversationWebhookHandler(svc *leadsvc.ConversationService, accountRepo *socialrepo.AccountRepository) *WeComConversationWebhookHandler {
	return &WeComConversationWebhookHandler{svc: svc, accountRepo: accountRepo}
}

type WeComConversationWebhookRequest struct {
	ChannelAccountUUID string         `json:"channel_account_uuid" binding:"required,uuid4"`
	ExternalEventID    string         `json:"external_event_id" binding:"required"`
	ConversationID     string         `json:"conversation_id" binding:"required"`
	ActorType          string         `json:"actor_type" binding:"required,oneof=staff app bot customer system"`
	ActorID            string         `json:"actor_id" binding:"required"`
	Direction          string         `json:"direction" binding:"omitempty,oneof=inbound outbound"`
	MessageType        string         `json:"message_type" binding:"required,oneof=text image file other"`
	ContentText        string         `json:"content_text"`
	OccurredAt         string         `json:"occurred_at" binding:"required"`
	RawPayload         map[string]any `json:"raw_payload"`
}

func (h *WeComConversationWebhookHandler) Ingest(c *gin.Context) {
	if h == nil || h.svc == nil || h.accountRepo == nil {
		contracts.ResponseServiceUnavailable(c, "conversation service unavailable", nil)
		return
	}
	signature := strings.TrimSpace(c.GetHeader("X-WeCom-Signature"))
	timestamp := strings.TrimSpace(c.GetHeader("X-WeCom-Timestamp"))
	nonce := strings.TrimSpace(c.GetHeader("X-WeCom-Nonce"))
	if signature == "" || timestamp == "" || nonce == "" {
		contracts.ResponseBadRequest(c, "invalid signature headers")
		return
	}
	var req WeComConversationWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	account, err := h.accountRepo.FindByUUID(c.Request.Context(), strings.TrimSpace(req.ChannelAccountUUID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			contracts.ResponseBadRequest(c, "channel account not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	tenantUUID := strings.TrimSpace(account.TenantUuid)
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
	contracts.ResponseSuccess(c, gin.H{"ok": true, "event_uuid": eventUUID, "created": created})
}
