package lead_capture

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type WeComSyncHandler struct {
	svc *leadsvc.WeComSyncService
}

func NewWeComSyncHandler(svc *leadsvc.WeComSyncService) *WeComSyncHandler {
	return &WeComSyncHandler{svc: svc}
}

func (h *WeComSyncHandler) TriggerSync(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom sync service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.TriggerWeComSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	task, err := h.svc.TriggerSync(c.Request.Context(), leadsvc.TriggerSyncRequest{
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: strings.TrimSpace(req.ChannelAccountUUID),
		TraceID:            strings.TrimSpace(req.TraceID),
		TriggerType:        "manual",
	})
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"task_uuid":              task.TaskUUID,
		"external_task_id":       task.ExternalTaskID,
		"channel_account_uuid":   task.ChannelAccountUUID,
		"account_resolve_source": task.AccountResolveSource,
		"task_provider":          task.TaskProvider,
		"status":                 task.Status,
	})
}
