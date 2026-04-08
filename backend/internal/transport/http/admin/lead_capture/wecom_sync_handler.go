package lead_capture

import (
	"net/http"
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
	writebacks := make([]leadsvc.LeadWritebackRecord, 0, len(req.LeadWriteback))
	for _, item := range req.LeadWriteback {
		writebacks = append(writebacks, leadsvc.LeadWritebackRecord{
			LeadUUID:        strings.TrimSpace(item.LeadUUID),
			ExternalUserID:  strings.TrimSpace(item.ExternalUserID),
			CorpID:          strings.TrimSpace(item.CorpID),
			Phone:           strings.TrimSpace(item.Phone),
			Fields:          item.Fields,
			OrderVersion:    item.OrderVersion,
			IdempotencyHint: strings.TrimSpace(item.IdempotencyHint),
		})
	}
	task, err := h.svc.TriggerSyncAsync(c.Request.Context(), leadsvc.TriggerSyncRequest{
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: strings.TrimSpace(req.ChannelAccountUUID),
		TraceID:            strings.TrimSpace(req.TraceID),
		TriggerType:        "manual",
		Domain:             strings.TrimSpace(req.Domain),
		Direction:          strings.TrimSpace(req.Direction),
		Mode:               strings.TrimSpace(req.Mode),
		CheckpointCursor:   strings.TrimSpace(req.CheckpointCursor),
		LeadWriteback:      writebacks,
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

func (h *WeComSyncHandler) GetWritebackPolicy(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom sync service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	channel := strings.TrimSpace(c.DefaultQuery("channel", "wechat"))
	appType := strings.TrimSpace(c.DefaultQuery("app_type", "wecom"))
	policy, err := h.svc.GetLeadWritebackPolicy(c.Request.Context(), tenantUUID, channel, appType)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	contracts.ResponseSuccess(c, dto.WeComWritebackPolicyResponse{
		Domain:           policy.Domain,
		CapabilityStatus: policy.CapabilityStatus,
		Enabled:          policy.Enabled,
		OverwriteMode:    policy.OverwriteMode,
		MappingRules:     map[string]any(policy.MappingRules),
		ProtectedFields:  map[string]any(policy.ProtectedFields),
	})
}

func (h *WeComSyncHandler) UpdateWritebackPolicy(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom sync service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.UpdateWeComWritebackPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	channel := strings.TrimSpace(c.DefaultQuery("channel", "wechat"))
	appType := strings.TrimSpace(c.DefaultQuery("app_type", "wecom"))
	policy, err := h.svc.UpdateLeadWritebackPolicy(
		c.Request.Context(),
		tenantUUID,
		channel,
		appType,
		req.MappingRules,
		req.ProtectedFields,
		req.OverwriteMode,
		req.Enabled,
	)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	contracts.ResponseSuccess(c, dto.WeComWritebackPolicyResponse{
		Domain:           policy.Domain,
		CapabilityStatus: policy.CapabilityStatus,
		Enabled:          policy.Enabled,
		OverwriteMode:    policy.OverwriteMode,
		MappingRules:     map[string]any(policy.MappingRules),
		ProtectedFields:  map[string]any(policy.ProtectedFields),
	})
}

func (h *WeComSyncHandler) ListWritebackDeadLetters(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom sync service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.svc.ListLeadWritebackDeadLetters(c.Request.Context(), tenantUUID, 50)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *WeComSyncHandler) ReplayWritebackDeadLetter(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom sync service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	deadLetterUUID := strings.TrimSpace(c.Param("dead_letter_uuid"))
	if deadLetterUUID == "" {
		contracts.ResponseBadRequest(c, "dead_letter_uuid is required")
		return
	}
	item, err := h.svc.ReplayLeadWritebackDeadLetter(c.Request.Context(), tenantUUID, deadLetterUUID)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	contracts.ResponseSuccess(c, item)
}
