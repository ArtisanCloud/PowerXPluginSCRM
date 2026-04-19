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
	jobSvc               *socialsvc.SyncJobService
	tagRecordSvc         *socialsvc.TagRecordService
	customerTagBindingSv *socialsvc.CustomerTagBindingService
	orchestrator         *socialsvc.SyncOrchestrator
	capabilitySvc        *socialsvc.CapabilityService
	conflictSvc          *socialsvc.ConflictResolutionService
}

func NewSyncJobHandler(
	jobSvc *socialsvc.SyncJobService,
	orchestrator *socialsvc.SyncOrchestrator,
	capabilitySvc *socialsvc.CapabilityService,
	conflictSvc *socialsvc.ConflictResolutionService,
	tagRecordSvc *socialsvc.TagRecordService,
	customerTagBindingSvc *socialsvc.CustomerTagBindingService,
) *SyncJobHandler {
	return &SyncJobHandler{
		jobSvc:               jobSvc,
		tagRecordSvc:         tagRecordSvc,
		customerTagBindingSv: customerTagBindingSvc,
		orchestrator:         orchestrator,
		capabilitySvc:        capabilitySvc,
		conflictSvc:          conflictSvc,
	}
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

func (h *SyncJobHandler) ListTags(c *gin.Context) {
	if h == nil || h.tagRecordSvc == nil {
		contracts.ResponseServiceUnavailable(c, "tag record service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := h.tagRecordSvc.List(
		c.Request.Context(),
		tenantUUID,
		strings.TrimSpace(c.Query("channel_account_uuid")),
		limit,
	)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *SyncJobHandler) ListCustomerTagBindings(c *gin.Context) {
	if h == nil || h.customerTagBindingSv == nil {
		contracts.ResponseServiceUnavailable(c, "customer tag binding service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	channelAccountUUID := strings.TrimSpace(c.Query("channel_account_uuid"))
	if channelAccountUUID == "" {
		contracts.ResponseBadRequest(c, "channel_account_uuid is required")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := h.customerTagBindingSv.ListByChannel(
		c.Request.Context(),
		tenantUUID,
		channelAccountUUID,
		limit,
	)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *SyncJobHandler) ClearTerminal(c *gin.Context) {
	if h == nil || h.jobSvc == nil {
		contracts.ResponseServiceUnavailable(c, "sync job service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	domain := strings.TrimSpace(c.Query("domain"))
	if domain == "" {
		contracts.ResponseBadRequest(c, "domain is required")
		return
	}
	includeInFlight := strings.EqualFold(strings.TrimSpace(c.Query("include_inflight")), "true") ||
		strings.TrimSpace(c.Query("include_inflight")) == "1"
	affected, err := h.jobSvc.ClearJobs(c.Request.Context(), tenantUUID, domain, includeInFlight)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"domain":           domain,
		"include_inflight": includeInFlight,
		"deleted_count":    affected,
	})
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

func (h *SyncJobHandler) Overview(c *gin.Context) {
	if h == nil || h.jobSvc == nil {
		contracts.ResponseServiceUnavailable(c, "sync job service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	jobs, err := h.jobSvc.List(c.Request.Context(), tenantUUID, "", "", 200)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// T038: org/tags observability summary for dashboard cards.
	perDomain := map[string]map[string]int{
		"tags": {},
		"org":  {},
	}
	for _, job := range jobs {
		domain := strings.TrimSpace(job.Domain)
		if domain != "tags" && domain != "org" {
			continue
		}
		if perDomain[domain] == nil {
			perDomain[domain] = map[string]int{}
		}
		perDomain[domain][job.Status]++
	}
	openConflicts := map[string]int{
		"tags": 0,
		"org":  0,
	}
	if h.conflictSvc != nil {
		if items, e := h.conflictSvc.List(c.Request.Context(), tenantUUID, "tags", "open", 200); e == nil {
			openConflicts["tags"] = len(items)
		}
		if items, e := h.conflictSvc.List(c.Request.Context(), tenantUUID, "org", "open", 200); e == nil {
			openConflicts["org"] = len(items)
		}
	}
	contracts.ResponseSuccess(c, gin.H{
		"jobs":           perDomain,
		"open_conflicts": openConflicts,
	})
}
