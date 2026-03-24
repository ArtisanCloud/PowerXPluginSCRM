package lead_capture

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type WelcomeSyncHandler struct {
	svc   *leadsvc.WelcomeSyncService
	authz *leadsvc.WelcomeSyncAuthz
}

func NewWelcomeSyncHandler(svc *leadsvc.WelcomeSyncService, authz *leadsvc.WelcomeSyncAuthz) *WelcomeSyncHandler {
	return &WelcomeSyncHandler{svc: svc, authz: authz}
}

func (h *WelcomeSyncHandler) TriggerSync(c *gin.Context) {
	if h == nil || h.svc == nil || h.authz == nil {
		contracts.ResponseServiceUnavailable(c, "welcome sync service unavailable", nil)
		return
	}
	codeUUID := strings.TrimSpace(c.Param("code_uuid"))
	if codeUUID == "" {
		contracts.ResponseBadRequest(c, "code_uuid is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	tc, _ := authx.GetTenantContext(c)
	if err := h.authz.EnsureCanPublish(tc); err != nil {
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeForbidden, "permission denied")
		return
	}
	result, err := h.svc.TriggerSync(c.Request.Context(), leadsvc.WelcomeSyncTriggerRequest{
		TenantUUID:    tenantUUID,
		CodeUUID:      codeUUID,
		ActorUserUUID: "",
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrWelcomeSyncConfigNotFound), errors.Is(err, leadsvc.ErrWelcomeSyncChannelCodeAbsent):
			contracts.ResponseNotFound(c, err.Error())
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, result)
}

func (h *WelcomeSyncHandler) GetSyncStatus(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "welcome sync service unavailable", nil)
		return
	}
	codeUUID := strings.TrimSpace(c.Param("code_uuid"))
	if codeUUID == "" {
		contracts.ResponseBadRequest(c, "code_uuid is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	result, err := h.svc.GetStatus(c.Request.Context(), tenantUUID, codeUUID)
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrWelcomeSyncConfigNotFound):
			contracts.ResponseNotFound(c, err.Error())
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, result)
}
