package acquisition

import (
	"strconv"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type GroupLiveCodeHandler struct {
	svc *acqsvc.GroupLiveCodeService
}

func NewGroupLiveCodeHandler(svc *acqsvc.GroupLiveCodeService) *GroupLiveCodeHandler {
	return &GroupLiveCodeHandler{svc: svc}
}

func (h *GroupLiveCodeHandler) List(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.svc.List(c.Request.Context(), tenantUUID, limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}
