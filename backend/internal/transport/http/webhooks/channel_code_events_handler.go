package webhooks

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/gin-gonic/gin"
)

type ChannelCodeEventsWebhookHandler struct {
	svc *leadsvc.ChannelCodeEventService
}

func NewChannelCodeEventsWebhookHandler(svc *leadsvc.ChannelCodeEventService) *ChannelCodeEventsWebhookHandler {
	return &ChannelCodeEventsWebhookHandler{svc: svc}
}

func (h *ChannelCodeEventsWebhookHandler) Ingest(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel code event service unavailable", nil)
		return
	}
	channel := strings.ToLower(strings.TrimSpace(c.Param("channel")))
	if channel == "" {
		contracts.ResponseBadRequest(c, "channel is required")
		return
	}
	var req dto.ChannelCodeEventWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	result, err := h.svc.IngestWebhook(c.Request.Context(), leadsvc.ChannelCodeEventIngestRequest{
		Channel:            channel,
		ChannelAccountUUID: req.ChannelAccountUUID,
		CodeKey:            req.CodeKey,
		ExternalEventID:    req.ExternalEventID,
		EventType:          req.EventType,
		OccurredAt:         req.OccurredAt,
		Payload:            req.Payload,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidChannelCodeEventPayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid webhook payload")
		case errors.Is(err, leadsvc.ErrChannelCodeNotFound):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "channel code not found")
		case errors.Is(err, leadsvc.ErrChannelCodeDisabled):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "channel code disabled")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"ok":             true,
		"event_uuid":     result.Event.EventUUID,
		"created":        result.Created,
		"idempotent_hit": result.IdempotentHit,
		"lead_uuid":      result.LeadUUID,
		"is_primary":     result.IsPrimary,
	})
}
