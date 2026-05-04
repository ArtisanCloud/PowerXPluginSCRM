package lead_capture

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/lead_capture"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type LeadHandler struct {
	svc *leadsvc.LeadService
}

func NewLeadHandler(svc *leadsvc.LeadService) *LeadHandler {
	return &LeadHandler{svc: svc}
}

func (h *LeadHandler) Create(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	var req dto.LeadCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	created, err := h.svc.Create(c.Request.Context(), tenantUUID, leadsvc.LeadCreateRequest{
		DisplayName:       req.DisplayName,
		Phone:             req.Phone,
		Email:             req.Email,
		SourceChannel:     req.SourceChannel,
		SourceAppType:     req.SourceAppType,
		SourceAccountUUID: req.SourceAccountUUID,
		OwnerUserUUID:     req.OwnerUserUUID,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidLeadPayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid lead payload")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseCreated(c, created)
}

func (h *LeadHandler) List(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.svc.List(c.Request.Context(), tenantUUID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *LeadHandler) Get(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), tenantUUID, leadUUID)
	if err != nil {
		switch {
		case errors.Is(err, leadrepo.ErrLeadNotFound):
			contracts.ResponseNotFound(c, "lead not found")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *LeadHandler) Update(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	var req dto.LeadUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	updated, err := h.svc.Update(c.Request.Context(), tenantUUID, leadUUID, leadsvc.LeadUpdateRequest{
		DisplayName: req.DisplayName,
		Phone:       req.Phone,
		Email:       req.Email,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidLeadPayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid lead payload")
		case errors.Is(err, leadrepo.ErrLeadNotFound):
			contracts.ResponseNotFound(c, "lead not found")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, updated)
}

func (h *LeadHandler) Import(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		contracts.ResponseBadRequest(c, "file is required")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".csv" {
		contracts.ResponseBadRequest(c, "only .csv is supported")
		return
	}
	opened, err := file.Open()
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	defer opened.Close()
	result, err := h.svc.ImportCSV(c.Request.Context(), tenantUUID, opened)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	resp := dto.LeadImportResult{
		Total:   result.Total,
		Success: result.Success,
		Failed:  result.Failed,
		Errors:  make([]dto.LeadImportError, 0, len(result.Errors)),
	}
	for _, item := range result.Errors {
		resp.Errors = append(resp.Errors, dto.LeadImportError{
			Row:    item.Row,
			Reason: item.Reason,
		})
	}
	contracts.ResponseSuccess(c, resp)
}

func (h *LeadHandler) ImportPreview(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		contracts.ResponseBadRequest(c, "file is required")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".csv" {
		contracts.ResponseBadRequest(c, "only .csv is supported")
		return
	}
	opened, err := file.Open()
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	defer opened.Close()
	preview, err := h.svc.PreviewImportCSV(c.Request.Context(), tenantUUID, opened)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, dto.LeadImportPreviewResponse{
		Headers:           preview.Headers,
		SampleRows:        preview.SampleRows,
		SuggestedMappings: preview.SuggestedMappings,
		RequiredFields:    preview.RequiredFields,
		AllFields:         preview.AllFields,
	})
}

func (h *LeadHandler) ImportConfirm(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		contracts.ResponseBadRequest(c, "file is required")
		return
	}
	rawMapping := strings.TrimSpace(c.PostForm("mapping"))
	if rawMapping == "" {
		contracts.ResponseBadRequest(c, "mapping is required")
		return
	}
	var mapping map[string]int
	if err := json.Unmarshal([]byte(rawMapping), &mapping); err != nil {
		contracts.ResponseBadRequest(c, "invalid mapping format")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".csv" {
		contracts.ResponseBadRequest(c, "only .csv is supported")
		return
	}
	opened, err := file.Open()
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	defer opened.Close()
	result, err := h.svc.ImportCSVWithMapping(c.Request.Context(), tenantUUID, opened, mapping)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	resp := dto.LeadImportResult{
		Total:   result.Total,
		Success: result.Success,
		Failed:  result.Failed,
		Errors:  make([]dto.LeadImportError, 0, len(result.Errors)),
	}
	for _, item := range result.Errors {
		resp.Errors = append(resp.Errors, dto.LeadImportError{
			Row:    item.Row,
			Reason: item.Reason,
		})
	}
	contracts.ResponseSuccess(c, resp)
}

func (h *LeadHandler) Assign(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	var req dto.LeadAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	updated, err := h.svc.Assign(c.Request.Context(), tenantUUID, leadUUID, leadsvc.LeadAssignRequest{
		OwnerUserUUID: req.OwnerUserUUID,
		Reason:        req.Reason,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidAssignee):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid assignee")
		case errors.Is(err, leadsvc.ErrAssigneeNotFound):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "assignee not found")
		case errors.Is(err, leadsvc.ErrAssigneeNotBound):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "assignee not bound to source member")
		case errors.Is(err, leadsvc.ErrAssigneeTransferFailed):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, strings.TrimSpace(err.Error()))
		case errors.Is(err, leadrepo.ErrLeadNotFound):
			contracts.ResponseNotFound(c, "lead not found")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, updated)
}

func (h *LeadHandler) BatchAssign(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	var req dto.LeadBatchAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	result, err := h.svc.BatchAssign(c.Request.Context(), tenantUUID, leadsvc.LeadBatchAssignRequest{
		LeadUUIDs:     req.LeadUUIDs,
		OwnerUserUUID: req.OwnerUserUUID,
		Reason:        req.Reason,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidAssignee):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid assignee")
		case errors.Is(err, leadsvc.ErrAssigneeNotFound):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "assignee not found")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, result)
}

func (h *LeadHandler) UpdateStatus(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	var req dto.LeadStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	updated, err := h.svc.UpdateStatus(c.Request.Context(), tenantUUID, leadUUID, leadsvc.LeadStatusUpdateRequest{
		Status: req.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidLeadStatus):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid lead status")
		case errors.Is(err, leadsvc.ErrInvalidLeadStatusTransition):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid status transition")
		case errors.Is(err, leadrepo.ErrLeadNotFound):
			contracts.ResponseNotFound(c, "lead not found")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, updated)
}

func (h *LeadHandler) ListAssignments(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.svc.ListAssignments(c.Request.Context(), tenantUUID, leadUUID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *LeadHandler) ListStatusHistory(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.svc.ListStatusHistory(c.Request.Context(), tenantUUID, leadUUID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *LeadHandler) ListActivities(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.svc.ListActivities(c.Request.Context(), tenantUUID, leadUUID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *LeadHandler) ListSourceEvents(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.svc.ListSourceEvents(c.Request.Context(), tenantUUID, leadUUID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}
