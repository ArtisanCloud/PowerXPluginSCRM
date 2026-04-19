package webhooks

import (
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	"github.com/gin-gonic/gin"
)

type AcquisitionGroupChatWebhookRequest struct {
	TenantUUID         string         `json:"tenant_uuid"`
	ChannelAccountUUID string         `json:"channel_account_uuid" binding:"required"`
	ChatID             string         `json:"chat_id" binding:"required"`
	Name               string         `json:"name"`
	OwnerUserID        string         `json:"owner_userid"`
	MemberCount        int            `json:"member_count"`
	SourceConfigID     string         `json:"source_config_id"`
	OccurredAt         string         `json:"occurred_at"`
	Payload            map[string]any `json:"payload"`
}

type AcquisitionGroupChatWebhookHandler struct {
	svc *acqsvc.GroupChatSyncService
}

func NewAcquisitionGroupChatWebhookHandler(svc *acqsvc.GroupChatSyncService) *AcquisitionGroupChatWebhookHandler {
	return &AcquisitionGroupChatWebhookHandler{svc: svc}
}

func (h *AcquisitionGroupChatWebhookHandler) Ingest(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group chat sync service unavailable", nil)
		return
	}
	var req AcquisitionGroupChatWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID := strings.TrimSpace(req.TenantUUID)
	if tenantUUID == "" {
		tenantUUID = strings.TrimSpace(c.GetHeader("X-Tenant-UUID"))
	}
	if tenantUUID == "" {
		tenantUUID = "00000000-0000-0000-0000-000000000001"
	}
	occurredAt := time.Now().UTC()
	if strings.TrimSpace(req.OccurredAt) != "" {
		if t, err := time.Parse(time.RFC3339, req.OccurredAt); err == nil {
			occurredAt = t.UTC()
		}
	}
	err := h.svc.ApplyWebhook(c.Request.Context(), acqsvc.GroupChatWebhookEvent{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: req.ChannelAccountUUID,
		ChatID:             req.ChatID,
		Name:               req.Name,
		OwnerUserID:        req.OwnerUserID,
		MemberCount:        req.MemberCount,
		SourceConfigID:     req.SourceConfigID,
		OccurredAt:         occurredAt,
		Payload:            req.Payload,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"accepted": true, "status": "success"})
}
