package acquisition

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/acquisition"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type StaffLiveCodeHandler struct {
	svc *acqsvc.StaffLiveCodeService
}

func NewStaffLiveCodeHandler(svc *acqsvc.StaffLiveCodeService) *StaffLiveCodeHandler {
	return &StaffLiveCodeHandler{svc: svc}
}

func (h *StaffLiveCodeHandler) Create(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff live code service unavailable", nil)
		return
	}
	var req dto.StaffLiveCodeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.Create(c.Request.Context(), acqsvc.StaffLiveCodeCreateRequest{
		TenantUUID:              tenantUUID,
		Channel:                 req.Channel,
		AppType:                 req.AppType,
		ChannelAccountUUID:      req.ChannelAccountUUID,
		ActivityName:            req.ActivityName,
		CodeKey:                 req.CodeKey,
		MemberUUIDs:             req.MemberUUIDs,
		CorpTagIDs:              req.CorpTagIDs,
		NewCustomerRemarkEnable: req.NewCustomerRemarkEnable,
	})
	if err != nil {
		switch {
		case errors.Is(err, acqsvc.ErrInvalidStaffLiveCodePayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid staff live code payload")
		case errors.Is(err, acqsvc.ErrDefaultChannelAccountNotFound):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "default channel account not found")
		case errors.Is(err, acqsvc.ErrStaffLiveCodeAlreadyExists):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, "staff live code already exists")
		case errors.Is(err, acqsvc.ErrStaffMemberBindingNotConfirmed):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "member bindings must be confirmed")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *StaffLiveCodeHandler) List(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff live code service unavailable", nil)
		return
	}
	var req dto.StaffLiveCodeListQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.svc.List(c.Request.Context(), acqsvc.StaffLiveCodeListRequest{
		TenantUUID:   tenantUUID,
		ActivityName: req.ActivityName,
		Status:       req.Status,
		Limit:        req.Limit,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *StaffLiveCodeHandler) CheckCodeKeyAvailable(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff live code service unavailable", nil)
		return
	}
	codeKey := strings.TrimSpace(c.Query("code_key"))
	if codeKey == "" {
		contracts.ResponseBadRequest(c, "code_key is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	available, err := h.svc.IsCodeKeyAvailable(c.Request.Context(), tenantUUID, codeKey)
	if err != nil {
		switch {
		case errors.Is(err, acqsvc.ErrInvalidStaffLiveCodePayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid code_key")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"code_key":  codeKey,
		"available": available,
	})
}

func (h *StaffLiveCodeHandler) UpdateStatus(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff live code service unavailable", nil)
		return
	}
	staffCodeUUID := strings.TrimSpace(c.Param("staff_code_uuid"))
	if staffCodeUUID == "" {
		contracts.ResponseBadRequest(c, "staff_code_uuid is required")
		return
	}
	var req dto.StaffLiveCodeStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.UpdateStatus(c.Request.Context(), acqsvc.StaffLiveCodeStatusUpdateRequest{
		TenantUUID:    tenantUUID,
		StaffCodeUUID: staffCodeUUID,
		Status:        req.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, acqsvc.ErrStaffLiveCodeStatusInvalid):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid status")
		case errors.Is(err, acqrepo.ErrRecordNotFound):
			contracts.ResponseNotFound(c, "staff live code not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, item)
}
