package lead_capture

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type ChannelCodeHandler struct {
	svc *leadsvc.ChannelCodeService
}

func NewChannelCodeHandler(svc *leadsvc.ChannelCodeService) *ChannelCodeHandler {
	return &ChannelCodeHandler{svc: svc}
}

func (h *ChannelCodeHandler) Create(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel code service unavailable", nil)
		return
	}
	var req dto.ChannelCodeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.Create(c.Request.Context(), leadsvc.ChannelCodeCreateRequest{
		TenantUUID:         tenantUUID,
		Channel:            req.Channel,
		AppType:            req.AppType,
		ChannelAccountUUID: req.ChannelAccountUUID,
		CodeKey:            req.CodeKey,
		DisplayName:        req.DisplayName,
		TargetType:         req.TargetType,
		TargetID:           req.TargetID,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidChannelCodePayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid channel code payload")
		case errors.Is(err, leadsvc.ErrChannelCodeAlreadyExists):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, "channel code already exists")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseCreated(c, item)
}

func (h *ChannelCodeHandler) List(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel code service unavailable", nil)
		return
	}
	var req dto.ChannelCodeListQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.svc.List(c.Request.Context(), leadsvc.ChannelCodeListRequest{
		TenantUUID:         tenantUUID,
		Channel:            req.Channel,
		AppType:            req.AppType,
		ChannelAccountUUID: req.ChannelAccountUUID,
		Status:             req.Status,
		Limit:              req.Limit,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *ChannelCodeHandler) UpdateStatus(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel code service unavailable", nil)
		return
	}
	codeUUID := strings.TrimSpace(c.Param("code_uuid"))
	if codeUUID == "" {
		contracts.ResponseBadRequest(c, "code_uuid is required")
		return
	}
	var req dto.ChannelCodeStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.UpdateStatus(c.Request.Context(), leadsvc.ChannelCodeStatusUpdateRequest{
		TenantUUID: tenantUUID,
		CodeUUID:   codeUUID,
		Status:     req.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrChannelCodeStatusInvalid):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid status")
		case errors.Is(err, leadrepo.ErrRecordNotFound):
			contracts.ResponseNotFound(c, "channel code not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, item)
}
