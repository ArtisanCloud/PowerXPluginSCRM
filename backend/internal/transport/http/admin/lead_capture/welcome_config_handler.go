package lead_capture

import (
	"errors"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type WelcomeConfigHandler struct {
	svc *leadsvc.WelcomeConfigService
}

func NewWelcomeConfigHandler(svc *leadsvc.WelcomeConfigService) *WelcomeConfigHandler {
	return &WelcomeConfigHandler{svc: svc}
}

func (h *WelcomeConfigHandler) Save(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "welcome config service unavailable", nil)
		return
	}
	codeUUID := strings.TrimSpace(c.Param("code_uuid"))
	if codeUUID == "" {
		contracts.ResponseBadRequest(c, "code_uuid is required")
		return
	}
	var req dto.WelcomeConfigSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.Save(c.Request.Context(), leadsvc.WelcomeConfigSaveRequest{
		TenantUUID:     tenantUUID,
		CodeUUID:       codeUUID,
		WelcomeEnabled: req.WelcomeEnabled,
		MessageContent: req.MessageContent,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidWelcomeConfigPayload):
			contracts.ResponseError(c, 400, contracts.ErrCodeValidationFailed, "invalid welcome config payload")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *WelcomeConfigHandler) ListHistory(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "welcome config service unavailable", nil)
		return
	}
	codeUUID := strings.TrimSpace(c.Param("code_uuid"))
	if codeUUID == "" {
		contracts.ResponseBadRequest(c, "code_uuid is required")
		return
	}
	var query dto.WelcomeConfigHistoryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.svc.ListChangeLogs(c.Request.Context(), tenantUUID, codeUUID, query.Limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}
