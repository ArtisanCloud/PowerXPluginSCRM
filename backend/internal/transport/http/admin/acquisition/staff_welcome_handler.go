package acquisition

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/acquisition"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type StaffWelcomeHandler struct {
	svc *acqsvc.StaffWelcomeService
}

func NewStaffWelcomeHandler(svc *acqsvc.StaffWelcomeService) *StaffWelcomeHandler {
	return &StaffWelcomeHandler{svc: svc}
}

func (h *StaffWelcomeHandler) Save(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff welcome service unavailable", nil)
		return
	}
	staffCodeUUID := strings.TrimSpace(c.Param("staff_code_uuid"))
	if staffCodeUUID == "" {
		contracts.ResponseBadRequest(c, "staff_code_uuid is required")
		return
	}
	var req dto.StaffWelcomeSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.Save(c.Request.Context(), acqsvc.StaffWelcomeSaveRequest{
		TenantUUID:    tenantUUID,
		StaffCodeUUID: staffCodeUUID,
		WelcomeMode:   req.WelcomeMode,
		ContentBlocks: req.ContentBlocks,
	})
	if err != nil {
		switch {
		case errors.Is(err, acqsvc.ErrInvalidStaffWelcomePayload):
			contracts.ResponseError(c, 400, contracts.ErrCodeValidationFailed, "invalid staff welcome payload")
		case errors.Is(err, acqsvc.ErrStaffWelcomeCodeNotFound):
			contracts.ResponseNotFound(c, "staff live code not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *StaffWelcomeHandler) TriggerSync(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff welcome service unavailable", nil)
		return
	}
	staffCodeUUID := strings.TrimSpace(c.Param("staff_code_uuid"))
	if staffCodeUUID == "" {
		contracts.ResponseBadRequest(c, "staff_code_uuid is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	result, err := h.svc.TriggerSync(c.Request.Context(), tenantUUID, staffCodeUUID, "")
	if err != nil {
		switch {
		case errors.Is(err, acqsvc.ErrStaffWelcomeConfigNotFound):
			contracts.ResponseNotFound(c, "staff welcome config not found")
		case errors.Is(err, acqsvc.ErrStaffWelcomeSyncNotImplemented):
			contracts.ResponseError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "channel sync adapter is not implemented")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, result)
}

func (h *StaffWelcomeHandler) GetSyncStatus(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff welcome service unavailable", nil)
		return
	}
	staffCodeUUID := strings.TrimSpace(c.Param("staff_code_uuid"))
	if staffCodeUUID == "" {
		contracts.ResponseBadRequest(c, "staff_code_uuid is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	result, err := h.svc.GetSyncStatus(c.Request.Context(), tenantUUID, staffCodeUUID)
	if err != nil {
		switch {
		case errors.Is(err, acqsvc.ErrStaffWelcomeConfigNotFound):
			contracts.ResponseNotFound(c, "staff welcome config not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, result)
}
