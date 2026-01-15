package operations

import (
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	operationsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/operations"
	"github.com/gin-gonic/gin"
)

// SLAHandler exposes admin endpoints for SLA management.
type SLAHandler struct {
	svc *operationsvc.SLAService
}

// NewSLAHandler builds a handler instance.
func NewSLAHandler(svc *operationsvc.SLAService) *SLAHandler {
	return &SLAHandler{svc: svc}
}

type slaTargetsRequest struct {
	PlanType string `json:"planType" binding:"required"`
	Targets  struct {
		UptimeTarget          float64 `json:"uptimeTarget"`
		ResponseTargetMs      int32   `json:"responseTargetMs"`
		SuccessTargetPct      float64 `json:"successTargetPct"`
		SupportFrtTargetHours float64 `json:"supportFrtTargetHours"`
	} `json:"targets" binding:"required"`
}

// GetProfiles handles GET /sla/profiles.
func (h *SLAHandler) GetProfiles(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "sla service unavailable", nil)
		return
	}
	profiles, err := h.svc.ListProfiles(c.Request.Context())
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, profiles)
}

// UpsertProfile handles POST /sla/profiles.
func (h *SLAHandler) UpsertProfile(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "sla service unavailable", nil)
		return
	}
	var req slaTargetsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	saved, err := h.svc.UpsertTargets(c.Request.Context(), operationsvc.ProfileTargets{
		PlanType:              req.PlanType,
		UptimeTarget:          req.Targets.UptimeTarget,
		ResponseTargetMs:      req.Targets.ResponseTargetMs,
		SuccessTargetPct:      req.Targets.SuccessTargetPct,
		SupportFrtTargetHours: req.Targets.SupportFrtTargetHours,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, saved)
}

// Recompute handles POST /sla/profiles/recompute.
func (h *SLAHandler) Recompute(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "sla service unavailable", nil)
		return
	}
	if _, err := h.svc.RecomputeScores(c.Request.Context()); err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

// UpdateActuals handles PATCH updates to actual metrics.
type slaActualsRequest struct {
	PlanType string                     `json:"planType" binding:"required"`
	Actuals  operationsvc.ActualMetrics `json:"actuals" binding:"required"`
}

// UpdateActuals endpoint allows manual actual metrics submission.
func (h *SLAHandler) UpdateActuals(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "sla service unavailable", nil)
		return
	}
	var req slaActualsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	profile, err := h.svc.UpdateActuals(c.Request.Context(), req.PlanType, req.Actuals)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, profile)
}
