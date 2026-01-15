package console

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	consolesvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// JobHandler exposes endpoints for job history and safe operations.
type JobHandler struct {
	jobs    *consolesvc.JobService
	safeOps *consolesvc.SafeOpsService
}

// NewJobHandler constructs a JobHandler when dependencies are available.
func NewJobHandler(deps *app.Deps) *JobHandler {
	if deps == nil || deps.DB == nil {
		return &JobHandler{}
	}
	svc := consolesvc.NewService(deps)
	return &JobHandler{
		jobs:    svc.Jobs(),
		safeOps: svc.SafeOps(),
	}
}

func respondAccepted(c *gin.Context, data any) {
	resp := contracts.APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
		RequestID: c.GetString("request_id"),
	}
	c.JSON(http.StatusAccepted, resp)
}

type listRunsQuery struct {
	TenantUuid string `form:"tenant_uuid"`
	JobType    string `form:"job_type"`
	Status     string `form:"status"`
	Cursor     string `form:"cursor"`
	Limit      int    `form:"limit"`
}

// ListRuns returns recent job runs filtered by tenant/job type.
func (h *JobHandler) ListRuns(c *gin.Context) {
	if h.jobs == nil {
		contracts.ResponseServiceUnavailable(c, "job service unavailable", nil)
		return
	}
	var query listRunsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	var tenantPtr *string
	if strings.TrimSpace(query.TenantUuid) != "" {
		clean := strings.TrimSpace(query.TenantUuid)
		tenantPtr = &clean
	}
	runs, next, err := h.jobs.ListRuns(
		c.Request.Context(),
		tenantPtr,
		strings.TrimSpace(query.JobType),
		strings.TrimSpace(query.Status),
		strings.TrimSpace(query.Cursor),
		query.Limit,
	)
	if err != nil {
		if errors.Is(err, consolesvc.ErrJobServiceUnavailable) {
			contracts.ResponseServiceUnavailable(c, err.Error(), nil)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"runs":        runs,
		"next_cursor": next,
	})
}

type retryRunQuery struct {
	TenantUuid string `form:"tenant_uuid"`
}

// RetryRun triggers a retry for a failed job run when eligible.
func (h *JobHandler) RetryRun(c *gin.Context) {
	if h.jobs == nil {
		contracts.ResponseServiceUnavailable(c, "job service unavailable", nil)
		return
	}
	runID := strings.TrimSpace(c.Param("runId"))
	if runID == "" {
		contracts.ResponseBadRequest(c, "run id is required")
		return
	}
	var query retryRunQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	var tenantPtr *string
	if strings.TrimSpace(query.TenantUuid) != "" {
		clean := strings.TrimSpace(query.TenantUuid)
		tenantPtr = &clean
	}
	actor := resolveActor(c)
	actor.PermissionCode = "operations.plugin.ops"
	result, err := h.jobs.RetryRun(c.Request.Context(), consolesvc.RetryRunInput{
		RunID:      runID,
		TenantUuid: tenantPtr,
		Actor:      actor,
	})
	if err != nil {
		switch {
		case errors.Is(err, consolesvc.ErrJobServiceUnavailable):
			contracts.ResponseServiceUnavailable(c, err.Error(), nil)
		case errors.Is(err, consolesvc.ErrJobRunNotFound):
			contracts.ResponseNotFound(c, err.Error())
		case errors.Is(err, consolesvc.ErrRetryNotAllowed):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		case errors.Is(err, consolesvc.ErrOperationInProgress):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, err.Error())
		default:
			if field, msg, ok := consolesvc.IsValidationError(err); ok {
				contracts.ResponseErrorWithDetails(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, msg, gin.H{"field": field})
			} else {
				contracts.ResponseInternalError(c, err)
			}
		}
		return
	}
	respondAccepted(c, result)
}

type safeOpBody struct {
	TenantUuid  string `json:"tenant_uuid"`
	Environment string `json:"environment"`
	Action      string `json:"action" binding:"required"`
	ScopeType   string `json:"scope_type" binding:"required"`
	ScopeRef    string `json:"scope_ref" binding:"required"`
	TargetID    string `json:"target_id"`
	Reason      string `json:"reason"`
	DryRun      bool   `json:"dry_run"`
}

// ExecuteSafeOp accepts a safe operation request and schedules execution.
func (h *JobHandler) ExecuteSafeOp(c *gin.Context) {
	if h.safeOps == nil {
		contracts.ResponseServiceUnavailable(c, "safe-ops service unavailable", nil)
		return
	}
	var body safeOpBody
	if err := c.ShouldBindJSON(&body); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}
	actor := resolveActor(c)
	actor.PermissionCode = "operations.plugin.ops"
	var tenantPtr *string
	if strings.TrimSpace(body.TenantUuid) != "" {
		clean := strings.TrimSpace(body.TenantUuid)
		tenantPtr = &clean
	}
	request := consolesvc.SafeOpRequest{
		TenantUuid:  tenantPtr,
		Environment: strings.TrimSpace(body.Environment),
		Action:      consolesvc.SafeOpAction(strings.ToLower(strings.TrimSpace(body.Action))),
		ScopeType:   consolesvc.SafeOpScope(strings.ToLower(strings.TrimSpace(body.ScopeType))),
		ScopeRef:    strings.TrimSpace(body.ScopeRef),
		TargetID:    strings.TrimSpace(body.TargetID),
		Reason:      strings.TrimSpace(body.Reason),
		DryRun:      body.DryRun,
		Actor:       actor,
	}
	result, err := h.safeOps.Execute(c.Request.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, consolesvc.ErrJobServiceUnavailable):
			contracts.ResponseServiceUnavailable(c, err.Error(), nil)
		case errors.Is(err, consolesvc.ErrOperationInProgress):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, err.Error())
		default:
			if field, msg, ok := consolesvc.IsValidationError(err); ok {
				contracts.ResponseErrorWithDetails(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, msg, gin.H{"field": field})
			} else {
				contracts.ResponseInternalError(c, err)
			}
		}
		return
	}
	respondAccepted(c, result)
}
