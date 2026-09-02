package lead_capture

import (
	"encoding/json"
	"errors"
	"fmt"
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
		DisplayName:       req.DisplayName,
		Phone:             req.Phone,
		Email:             req.Email,
		SourceChannel:     req.SourceChannel,
		SourceAppType:     req.SourceAppType,
		SourceAccountUUID: req.SourceAccountUUID,
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

func (h *LeadHandler) UpdateQualification(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	var req dto.LeadQualificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	targetStatus := strings.TrimSpace(req.TargetStatus)
	if targetStatus == "" {
		targetStatus = strings.TrimSpace(req.Status)
	}
	if targetStatus == "" {
		contracts.ResponseBadRequest(c, "target_status is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	updated, err := h.svc.UpdateQualification(c.Request.Context(), tenantUUID, leadUUID, leadsvc.LeadStatusUpdateRequest{
		Status: targetStatus,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidLeadStatus):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid lead qualification target")
		case errors.Is(err, leadsvc.ErrInvalidLeadStatusTransition):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid qualification transition")
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

func (h *LeadHandler) ListTimeline(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "leadCapture.errors.serviceUnavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "leadCapture.errors.invalidLead")
		return
	}
	var query dto.LeadTimelineQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "leadCapture.errors.invalidTimelineQuery")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "leadCapture.errors.tenantContextMissing")
		return
	}
	result, err := h.svc.ListTimeline(c.Request.Context(), tenantUUID, leadUUID, leadsvc.LeadTimelineQuery{
		StageKey: query.StageKey, EventType: query.EventType, Page: query.Page, PageSize: query.PageSize,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidLeadTimelineQuery):
			contracts.ResponseBadRequest(c, "leadCapture.errors.invalidTimelineQuery")
		case errors.Is(err, leadrepo.ErrLeadNotFound):
			contracts.ResponseNotFound(c, "leadCapture.errors.leadNotFound")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "leadCapture.errors.tenantRequired")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, result)
}

func (h *LeadHandler) RecordActivity(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "lead service unavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	if leadUUID == "" {
		contracts.ResponseBadRequest(c, "lead_id is required")
		return
	}
	var req dto.LeadActivityCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.RecordActivity(c.Request.Context(), tenantUUID, leadUUID, leadsvc.LeadActivityCreateRequest{
		Method:         req.Method,
		Subject:        req.Subject,
		Content:        req.Content,
		Result:         req.Result,
		NextStep:       req.NextStep,
		NextFollowUpAt: req.NextFollowUpAt,
		StageKey:       req.StageKey,
		ActionKey:      req.ActionKey,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidLeadPayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid activity payload")
		case errors.Is(err, leadrepo.ErrLeadNotFound):
			contracts.ResponseNotFound(c, "lead not found")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseCreated(c, item)
}

func (h *LeadHandler) UploadActivityAttachment(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "leadCapture.errors.serviceUnavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	activityUUID := strings.TrimSpace(c.Param("activity_id"))
	if leadUUID == "" || activityUUID == "" {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeLeadAttachmentInvalid, "leadCapture.errors.attachmentInvalid")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "leadCapture.errors.tenantContextMissing")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeLeadAttachmentInvalid, "leadCapture.errors.attachmentInvalid")
		return
	}
	opened, err := fileHeader.Open()
	if err != nil {
		contracts.ResponseError(c, http.StatusInternalServerError, contracts.ErrCodeInternalError, "leadCapture.errors.internal")
		return
	}
	defer opened.Close()
	item, err := h.svc.UploadActivityAttachment(c.Request.Context(), tenantUUID, leadUUID, leadsvc.LeadAttachmentUploadRequest{
		ActivityUUID: activityUUID,
		StageKey:     c.PostForm("stage_key"),
		ActionKey:    c.PostForm("action_key"),
		FileName:     filepath.Base(fileHeader.Filename),
		ContentType:  fileHeader.Header.Get("Content-Type"),
		FileSize:     fileHeader.Size,
		Content:      opened,
	})
	if err != nil {
		respondLeadAttachmentError(c, err)
		return
	}
	contracts.ResponseCreated(c, item)
}

func (h *LeadHandler) ListActivityAttachments(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "leadCapture.errors.serviceUnavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	activityUUID := strings.TrimSpace(c.Param("activity_id"))
	if leadUUID == "" || activityUUID == "" {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeLeadAttachmentInvalid, "leadCapture.errors.attachmentInvalid")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "leadCapture.errors.tenantContextMissing")
		return
	}
	items, err := h.svc.ListActivityAttachments(c.Request.Context(), tenantUUID, leadUUID, activityUUID)
	if err != nil {
		respondLeadAttachmentError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *LeadHandler) UploadNodeAttachment(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "leadCapture.errors.serviceUnavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	stageKey := strings.TrimSpace(c.PostForm("stage_key"))
	if leadUUID == "" || stageKey == "" {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeLeadAttachmentInvalid, "leadCapture.errors.attachmentInvalid")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "leadCapture.errors.tenantContextMissing")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeLeadAttachmentInvalid, "leadCapture.errors.attachmentInvalid")
		return
	}
	opened, err := fileHeader.Open()
	if err != nil {
		contracts.ResponseError(c, http.StatusInternalServerError, contracts.ErrCodeInternalError, "leadCapture.errors.internal")
		return
	}
	defer opened.Close()
	item, err := h.svc.UploadNodeAttachment(c.Request.Context(), tenantUUID, leadUUID, leadsvc.LeadAttachmentUploadRequest{
		StageKey:    stageKey,
		ActionKey:   c.PostForm("action_key"),
		FileName:    filepath.Base(fileHeader.Filename),
		ContentType: fileHeader.Header.Get("Content-Type"),
		FileSize:    fileHeader.Size,
		Content:     opened,
	})
	if err != nil {
		respondLeadAttachmentError(c, err)
		return
	}
	contracts.ResponseCreated(c, item)
}

func (h *LeadHandler) ListNodeAttachments(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "leadCapture.errors.serviceUnavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	stageKey := strings.TrimSpace(c.Query("stage_key"))
	if leadUUID == "" || stageKey == "" {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeLeadAttachmentInvalid, "leadCapture.errors.attachmentInvalid")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "leadCapture.errors.tenantContextMissing")
		return
	}
	items, err := h.svc.ListNodeAttachments(c.Request.Context(), tenantUUID, leadUUID, stageKey, c.Query("action_key"))
	if err != nil {
		respondLeadAttachmentError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *LeadHandler) DownloadAttachment(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "leadCapture.errors.serviceUnavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	attachmentUUID := strings.TrimSpace(c.Param("attachment_id"))
	if leadUUID == "" || attachmentUUID == "" {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeLeadAttachmentInvalid, "leadCapture.errors.attachmentInvalid")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "leadCapture.errors.tenantContextMissing")
		return
	}
	item, err := h.svc.GetAttachment(c.Request.Context(), tenantUUID, leadUUID, attachmentUUID)
	if err != nil {
		respondLeadAttachmentError(c, err)
		return
	}
	fileName := filepath.Base(strings.TrimSpace(item.FileName))
	if fileName == "" || fileName == "." || fileName == string(filepath.Separator) {
		contracts.ResponseError(c, http.StatusInternalServerError, contracts.ErrCodeInternalError, "leadCapture.errors.internal")
		return
	}
	contentType := strings.TrimSpace(item.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	c.Data(http.StatusOK, contentType, item.Content)
}

func (h *LeadHandler) DeleteAttachment(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "leadCapture.errors.serviceUnavailable", nil)
		return
	}
	leadUUID := strings.TrimSpace(c.Param("lead_id"))
	attachmentUUID := strings.TrimSpace(c.Param("attachment_id"))
	if leadUUID == "" || attachmentUUID == "" {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeLeadAttachmentInvalid, "leadCapture.errors.attachmentInvalid")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "leadCapture.errors.tenantContextMissing")
		return
	}
	item, err := h.svc.DeleteAttachment(c.Request.Context(), tenantUUID, leadUUID, attachmentUUID)
	if err != nil {
		respondLeadAttachmentError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"attachment_uuid": item.AttachmentUUID})
}

func respondLeadAttachmentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, leadsvc.ErrLeadAttachmentTooLarge):
		contracts.ResponseError(c, http.StatusRequestEntityTooLarge, contracts.ErrCodeLeadAttachmentTooLarge, "leadCapture.errors.attachmentTooLarge")
	case errors.Is(err, leadsvc.ErrInvalidLeadPayload), errors.Is(err, leadrepo.ErrLeadActivityNotFound):
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeLeadAttachmentInvalid, "leadCapture.errors.attachmentInvalid")
	case errors.Is(err, leadrepo.ErrLeadNotFound), errors.Is(err, leadrepo.ErrLeadAttachmentNotFound):
		contracts.ResponseError(c, http.StatusNotFound, contracts.ErrCodeLeadAttachmentNotFound, "leadCapture.errors.attachmentNotFound")
	case errors.Is(err, repository.ErrTenantUuidRequired):
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "leadCapture.errors.tenantRequired")
	default:
		contracts.ResponseError(c, http.StatusInternalServerError, contracts.ErrCodeInternalError, "leadCapture.errors.internal")
	}
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
