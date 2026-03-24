package lead_capture

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type ChannelCodeEventsHandler struct {
	svc *leadsvc.ChannelCodeEventService
}

func NewChannelCodeEventsHandler(svc *leadsvc.ChannelCodeEventService) *ChannelCodeEventsHandler {
	return &ChannelCodeEventsHandler{svc: svc}
}

func (h *ChannelCodeEventsHandler) List(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel code event service unavailable", nil)
		return
	}
	codeUUID := strings.TrimSpace(c.Param("code_uuid"))
	if codeUUID == "" {
		contracts.ResponseBadRequest(c, "code_uuid is required")
		return
	}
	var query dto.ChannelCodeEventListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	result, err := h.svc.ListByCodeUUID(c.Request.Context(), tenantUUID, codeUUID, query.Limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, result)
}
