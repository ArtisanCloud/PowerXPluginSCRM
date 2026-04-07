package social_channel_governance

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type SyncJobHandler struct {
	jobSvc        *socialsvc.SyncJobService
	orchestrator  *socialsvc.SyncOrchestrator
	capabilitySvc *socialsvc.CapabilityService
}

func NewSyncJobHandler(jobSvc *socialsvc.SyncJobService, orchestrator *socialsvc.SyncOrchestrator, capabilitySvc *socialsvc.CapabilityService) *SyncJobHandler {
	return &SyncJobHandler{jobSvc: jobSvc, orchestrator: orchestrator, capabilitySvc: capabilitySvc}
}

func (h *SyncJobHandler) Create(c *gin.Context) {
	if h == nil || h.orchestrator == nil {
		contracts.ResponseServiceUnavailable(c, "sync orchestrator unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req struct {
		Channel   string         `json:"channel"`
		AppType   string         `json:"app_type"`
		Domain    string         `json:"domain"`
		Direction string         `json:"direction"`
		Mode      string         `json:"mode"`
		Payload   map[string]any `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	jobUUID, err := h.orchestrator.Submit(c.Request.Context(), socialsvc.OrchestrateInput{
		TenantUUID: tenantUUID,
		Channel:    req.Channel,
		AppType:    req.AppType,
		Domain:     req.Domain,
		Direction:  req.Direction,
		Mode:       req.Mode,
		Payload:    req.Payload,
	})
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"job_uuid": jobUUID})
}

func (h *SyncJobHandler) List(c *gin.Context) {
	if h == nil || h.jobSvc == nil {
		contracts.ResponseServiceUnavailable(c, "sync job service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := h.jobSvc.List(c.Request.Context(), tenantUUID, c.Query("domain"), c.Query("status"), limit)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *SyncJobHandler) Capabilities(c *gin.Context) {
	if h == nil || h.capabilitySvc == nil {
		contracts.ResponseServiceUnavailable(c, "capability service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	matrix := h.capabilitySvc.Matrix(c.Request.Context(), tenantUUID, c.Query("channel"), c.Query("app_type"))
	contracts.ResponseSuccess(c, gin.H{"capability_status": matrix})
}
