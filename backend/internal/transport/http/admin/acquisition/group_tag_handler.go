package acquisition

import (
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/acquisition"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type GroupTagHandler struct {
	svc *acqsvc.GroupTagService
}

func NewGroupTagHandler(svc *acqsvc.GroupTagService) *GroupTagHandler {
	return &GroupTagHandler{svc: svc}
}

func (h *GroupTagHandler) Create(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group tag service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.GroupTagCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), acqsvc.GroupTagCreateRequest{
		TenantUUID:  tenantUUID,
		TagName:     req.TagName,
		Color:       req.Color,
		RuleMode:    req.RuleMode,
		RulePayload: req.RulePayload,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *GroupTagHandler) List(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group tag service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := h.svc.List(c.Request.Context(), tenantUUID, limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *GroupTagHandler) Bind(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group tag service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.GroupTagBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	bound, err := h.svc.Bind(c.Request.Context(), acqsvc.GroupTagBindRequest{
		TenantUUID:   tenantUUID,
		GroupTagUUID: c.Param("group_tag_uuid"),
		ChatIDs:      req.ChatIDs,
		BindSource:   req.BindSource,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"bound": bound})
}

func (h *GroupTagHandler) ListBindings(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group tag service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := h.svc.ListBindings(c.Request.Context(), tenantUUID, c.Param("group_tag_uuid"), limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *GroupTagHandler) ReplayRule(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group tag service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.GroupTagReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	run, err := h.svc.ReplayRule(c.Request.Context(), acqsvc.GroupTagRuleReplayRequest{
		TenantUUID:    tenantUUID,
		GroupTagUUID:  c.Param("group_tag_uuid"),
		TriggerSource: req.TriggerSource,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, run)
}
