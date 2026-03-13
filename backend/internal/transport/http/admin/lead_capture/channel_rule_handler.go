package lead_capture

import (
	"errors"
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type ChannelRuleHandler struct {
	svc *leadsvc.ChannelRuleService
}

func NewChannelRuleHandler(svc *leadsvc.ChannelRuleService) *ChannelRuleHandler {
	return &ChannelRuleHandler{svc: svc}
}

func (h *ChannelRuleHandler) GetWeComCustomerDMRule(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel rule service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	rule, err := h.svc.GetWeComCustomerDMRule(c.Request.Context(), tenantUUID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantUuidRequired) {
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, rule)
}

func (h *ChannelRuleHandler) UpdateWeComCustomerDMRule(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel rule service unavailable", nil)
		return
	}
	var req dto.UpdateWeComCustomerDMRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	rule, err := h.svc.UpdateWeComCustomerDMRule(c.Request.Context(), tenantUUID, req.Enabled)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		case errors.Is(err, leadsvc.ErrInvalidChannelRulePayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid channel rule payload")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, rule)
}
