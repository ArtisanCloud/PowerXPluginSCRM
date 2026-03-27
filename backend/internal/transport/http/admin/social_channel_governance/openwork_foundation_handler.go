package social_channel_governance

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/social_channel_governance"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type OpenWorkFoundationHandler struct {
	svc *socialsvc.OpenWorkFoundationService
}

func NewOpenWorkFoundationHandler(svc *socialsvc.OpenWorkFoundationService) *OpenWorkFoundationHandler {
	return &OpenWorkFoundationHandler{svc: svc}
}

func (h *OpenWorkFoundationHandler) IngestEvent(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.OpenWorkEventIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	event, binding, err := h.svc.IngestEvent(c.Request.Context(), socialsvc.OpenWorkEventIngestInput{
		TenantUUID:  tenantUUID,
		SuiteID:     req.SuiteID,
		EventType:   req.EventType,
		SuiteTicket: req.SuiteTicket,
		CorpID:      req.CorpID,
		AgentID:     req.AgentID,
		EventTime:   req.EventTime,
		Payload:     req.Payload,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"event":   event,
		"binding": binding,
	})
}

func (h *OpenWorkFoundationHandler) StartAuthorization(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.OpenWorkAuthorizeStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	result, err := h.svc.StartAuthorization(c.Request.Context(), socialsvc.OpenWorkAuthorizeStartInput{
		TenantUUID:  tenantUUID,
		SuiteID:     req.SuiteID,
		SuiteSecret: req.SuiteSecret,
		SuiteTicket: req.SuiteTicket,
		RedirectURI: req.RedirectURI,
		State:       req.State,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, result)
}

func (h *OpenWorkFoundationHandler) CompleteAuthorization(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.OpenWorkAuthorizeCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	binding, err := h.svc.CompleteAuthorization(c.Request.Context(), socialsvc.OpenWorkAuthorizeCompleteInput{
		TenantUUID:         tenantUUID,
		SuiteID:            req.SuiteID,
		SuiteSecret:        req.SuiteSecret,
		SuiteTicket:        req.SuiteTicket,
		AuthCode:           req.AuthCode,
		ChannelAccountUUID: req.ChannelAccountUUID,
		SetDefault:         req.SetDefault,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, binding)
}

func (h *OpenWorkFoundationHandler) ListBindings(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.svc.ListBindings(c.Request.Context(), tenantUUID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *OpenWorkFoundationHandler) GetAuthorizationStatus(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	suiteID := strings.TrimSpace(c.Query("suite_id"))
	state := strings.TrimSpace(c.Query("state"))
	startedAt, _ := strconv.ParseInt(strings.TrimSpace(c.Query("started_at")), 10, 64)
	data, err := h.svc.AuthorizationStatus(c.Request.Context(), socialsvc.OpenWorkAuthorizeStatusInput{
		TenantUUID: tenantUUID,
		SuiteID:    suiteID,
		State:      state,
		StartedAt:  startedAt,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, data)
}

func (h *OpenWorkFoundationHandler) SetDefaultBinding(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	bindingUUID := strings.TrimSpace(c.Param("binding_uuid"))
	if bindingUUID == "" {
		contracts.ResponseBadRequest(c, "binding_uuid is required")
		return
	}
	var req dto.OpenWorkSetDefaultRequest
	_ = c.ShouldBindJSON(&req)
	binding, err := h.svc.SetDefaultBinding(c.Request.Context(), tenantUUID, bindingUUID, req.ChannelAccountUUID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, binding)
}

func (h *OpenWorkFoundationHandler) CreateSyncJob(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.SyncBaselineJobCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	job, err := h.svc.CreateSyncJob(c.Request.Context(), socialsvc.SyncBaselineJobCreateInput{
		TenantUUID:      tenantUUID,
		BindingUUID:     req.BindingUUID,
		Domain:          req.Domain,
		Mode:            req.Mode,
		IdempotencyKey:  req.IdempotencyKey,
		MaxRetries:      req.MaxRetries,
		WriteBackFields: req.WriteBackFields,
		Context:         req.Context,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, job)
}

func (h *OpenWorkFoundationHandler) ListSyncJobs(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := h.svc.ListSyncJobs(c.Request.Context(), tenantUUID, status, limit)
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *OpenWorkFoundationHandler) ListSyncConflicts(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := h.svc.ListSyncConflicts(c.Request.Context(), tenantUUID, status, limit)
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *OpenWorkFoundationHandler) ReplaySyncConflict(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	conflictUUID := strings.TrimSpace(c.Param("conflict_uuid"))
	if conflictUUID == "" {
		contracts.ResponseBadRequest(c, "conflict_uuid is required")
		return
	}
	var req dto.SyncConflictReplayRequest
	_ = c.ShouldBindJSON(&req)
	record, err := h.svc.ReplayConflict(c.Request.Context(), tenantUUID, conflictUUID, req.Note)
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, record)
}

func (h *OpenWorkFoundationHandler) GetDashboard(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	data, err := h.svc.Dashboard(c.Request.Context(), tenantUUID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	contracts.ResponseSuccess(c, data)
}

func (h *OpenWorkFoundationHandler) GetGoLiveGates(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "openwork foundation service unavailable", nil)
		return
	}
	contracts.ResponseSuccess(c, h.svc.GoLiveGates())
}

func (h *OpenWorkFoundationHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrTenantUuidRequired):
		contracts.ResponseBadRequest(c, "tenant_uuid is required")
	case errors.Is(err, socialrepo.ErrBindingNotFound), errors.Is(err, socialrepo.ErrSyncJobNotFound), errors.Is(err, socialrepo.ErrSyncConflictNotFound):
		contracts.ResponseNotFound(c, err.Error())
	default:
		msg := strings.TrimSpace(err.Error())
		if msg != "" && !strings.Contains(strings.ToLower(msg), "internal") {
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, msg)
			return
		}
		contracts.ResponseInternalError(c, err)
	}
}
