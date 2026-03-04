package lead_capture

import (
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/gin-gonic/gin"
)

type WeComSyncHandler struct {
	svc *leadsvc.WeComSyncService
}

func NewWeComSyncHandler(svc *leadsvc.WeComSyncService) *WeComSyncHandler {
	return &WeComSyncHandler{svc: svc}
}

type TriggerWeComSyncRequest struct {
	ChannelAccountUUID string `json:"channel_account_uuid"`
	TraceID            string `json:"trace_id"`
}

func (h *WeComSyncHandler) TriggerSync(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom sync service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req TriggerWeComSyncRequest
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

func (h *WeComSyncHandler) ListSyncTasks(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom sync service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	channelAccountUUID := strings.TrimSpace(c.Query("channel_account_uuid"))
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.svc.ListSyncTasks(c.Request.Context(), tenantUUID, channelAccountUUID, limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}
