package webhooks

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	"github.com/gin-gonic/gin"
)

// AcquisitionCodeEventRequest is the v2 webhook skeleton payload for staff/group code events.
type AcquisitionCodeEventRequest struct {
	ChannelAccountUUID string         `json:"channel_account_uuid" binding:"required"`
	CodeKey            string         `json:"code_key" binding:"required"`
	ExternalEventID    string         `json:"external_event_id" binding:"required"`
	EventType          string         `json:"event_type" binding:"required"`
	OccurredAt         string         `json:"occurred_at" binding:"required"`
	Payload            map[string]any `json:"payload"`
}

type AcquisitionCodeEventsWebhookHandler struct{}

func NewAcquisitionCodeEventsWebhookHandler() *AcquisitionCodeEventsWebhookHandler {
	return &AcquisitionCodeEventsWebhookHandler{}
}

func (h *AcquisitionCodeEventsWebhookHandler) IngestStaff(c *gin.Context) {
	h.ingest(c, "staff_code")
}

func (h *AcquisitionCodeEventsWebhookHandler) IngestGroup(c *gin.Context) {
	h.ingest(c, "group_code")
}

func (h *AcquisitionCodeEventsWebhookHandler) ingest(c *gin.Context, scope string) {
	var req AcquisitionCodeEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	channel := strings.ToLower(strings.TrimSpace(c.Param("channel")))
	if channel == "" {
		contracts.ResponseBadRequest(c, "channel is required")
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"ok":          true,
		"accepted":    false,
		"scope":       scope,
		"channel":     channel,
		"status":      "not_implemented",
		"external_id": strings.TrimSpace(req.ExternalEventID),
	})
}
