package capability

import (
	"errors"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	lifecycle "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/capability_lifecycle"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// LifecycleHandler 提供生命周期计划相关接口。
type LifecycleHandler struct {
	service *lifecycle.PlanService
}

// NewLifecycleHandler 构造 Handler。
func NewLifecycleHandler(deps *app.Deps) *LifecycleHandler {
	if deps == nil {
		return nil
	}
	return &LifecycleHandler{
		service: lifecycle.NewPlanService(deps),
	}
}

// GetTemplate 返回变更类型与通知选项。
func (h *LifecycleHandler) GetTemplate(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability lifecycle service not available", nil)
		return
	}
	contracts.ResponseSuccess(c, h.service.Template())
}

// List 返回指定能力的生命周期计划。
func (h *LifecycleHandler) List(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability lifecycle service not available", nil)
		return
	}
	capID := strings.TrimSpace(c.Query("capability_id"))
	plans := h.service.ListPlans(capID)
	contracts.ResponseSuccess(c, gin.H{
		"capability_id": capID,
		"plans":         plans,
	})
}

// Create 创建生命周期计划。
func (h *LifecycleHandler) Create(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability lifecycle service not available", nil)
		return
	}
	var payload lifecycle.PlanInput
	if err := c.ShouldBindJSON(&payload); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload: "+err.Error())
		return
	}
	payload.Actor = actorFromContext(c)
	plan, err := h.service.CreatePlan(c.Request.Context(), &payload)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, plan)
}

// UpdateStatus 更新计划状态。
func (h *LifecycleHandler) UpdateStatus(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability lifecycle service not available", nil)
		return
	}
	var payload lifecycle.StatusInput
	if err := c.ShouldBindJSON(&payload); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload: "+err.Error())
		return
	}
	if payload.PlanID == "" {
		payload.PlanID = strings.TrimSpace(c.Param("planID"))
	}
	payload.Actor = actorFromContext(c)
	plan, err := h.service.UpdateStatus(c.Request.Context(), &payload)
	if err != nil {
		if errors.Is(err, lifecycle.ErrPlanNotFound) {
			contracts.ResponseNotFound(c, "lifecycle plan not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, plan)
}
