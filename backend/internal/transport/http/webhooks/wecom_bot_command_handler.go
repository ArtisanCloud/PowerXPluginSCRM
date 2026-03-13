package webhooks

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WeComBotCommandHandler struct {
	svc         *leadsvc.BotCommandService
	accountRepo *socialrepo.AccountRepository
}

func NewWeComBotCommandHandler(svc *leadsvc.BotCommandService, accountRepo *socialrepo.AccountRepository) *WeComBotCommandHandler {
	return &WeComBotCommandHandler{svc: svc, accountRepo: accountRepo}
}

func (h *WeComBotCommandHandler) Ingest(c *gin.Context) {
	if h == nil || h.svc == nil || h.accountRepo == nil {
		contracts.ResponseServiceUnavailable(c, "bot command service unavailable", nil)
		return
	}
	signature := strings.TrimSpace(c.GetHeader("X-WeCom-Signature"))
	timestamp := strings.TrimSpace(c.GetHeader("X-WeCom-Timestamp"))
	nonce := strings.TrimSpace(c.GetHeader("X-WeCom-Nonce"))
	if signature == "" || timestamp == "" || nonce == "" {
		contracts.ResponseBadRequest(c, "invalid signature headers")
		return
	}

	var req dto.WeComBotCommandRequest
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
	occurredAt := time.Now().UTC()
	if raw := strings.TrimSpace(req.OccurredAt); raw != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, raw); parseErr == nil {
			occurredAt = parsed.UTC()
		}
	}
	result, err := h.svc.HandleWeComLeadCreateCommand(c.Request.Context(), leadsvc.WeComBotCommandInput{
		TenantUUID:         strings.TrimSpace(account.TenantUuid),
		ChannelAccountUUID: strings.TrimSpace(req.ChannelAccountUUID),
		ExternalEventID:    strings.TrimSpace(req.ExternalEventID),
		ConversationID:     strings.TrimSpace(req.ConversationID),
		OperatorID:         strings.TrimSpace(req.OperatorID),
		CommandText:        strings.TrimSpace(req.CommandText),
		Permissions:        req.Permissions,
		OccurredAt:         occurredAt,
		RawPayload:         req.RawPayload,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrBotCommandForbidden):
			contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeForbidden, err.Error())
		case errors.Is(err, leadsvc.ErrBotCommandInvalidPayload), errors.Is(err, leadsvc.ErrBotCommandUnsupported):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, err.Error())
		default:
			contracts.ResponseBadRequest(c, err.Error())
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"request_id":     result.RequestID,
		"lead_id":        result.LeadUUID,
		"created":        result.Created,
		"idempotent_hit": !result.Created,
	})
}
