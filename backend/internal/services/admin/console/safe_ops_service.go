package console

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	consolerepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/admin_console"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SafeOpRequest represents the user request to execute a safe operation.
type SafeOpRequest struct {
	TenantUuid  *string
	Environment string
	Action      SafeOpAction
	ScopeType   SafeOpScope
	ScopeRef    string
	TargetID    string
	Reason      string
	DryRun      bool
	Actor       Actor
}

// SafeOpsService orchestrates execution of safe operations.
type SafeOpsService struct {
	cfg     *config.Config
	jobs    *JobService
	audits  *consolerepo.AuditRepository
	jobRepo *consolerepo.JobRunRepository
	metrics *adminmetrics.Metrics
	nowFunc func() time.Time
}

// NewSafeOpsService constructs the service using shared dependencies.
func NewSafeOpsService(deps *app.Deps, jobs *JobService) *SafeOpsService {
	if jobs == nil && deps != nil {
		jobs = NewJobService(deps)
	}
	var metrics *adminmetrics.Metrics
	var cfg *config.Config
	var db = (*gorm.DB)(nil)
	if deps != nil {
		metrics = deps.AdminConsoleMetrics
		cfg = deps.Config
		db = deps.DB
	}
	if metrics == nil {
		metrics = adminmetrics.NewMetrics()
	}
	return &SafeOpsService{
		cfg:     cfg,
		jobs:    jobs,
		audits:  consolerepo.NewAuditRepository(db),
		jobRepo: consolerepo.NewJobRunRepository(db),
		metrics: metrics,
		nowFunc: time.Now,
	}
}

// Execute schedules a safe operation and records audit history.
func (s *SafeOpsService) Execute(ctx context.Context, input SafeOpRequest) (*JobRunRecord, error) {
	if s == nil || s.jobs == nil {
		return nil, ErrJobServiceUnavailable
	}
	actor := input.Actor
	if strings.TrimSpace(actor.PermissionCode) == "" {
		return nil, validationError{Field: "permission_code", Message: "permission code required"}
	}
	action := strings.TrimSpace(string(input.Action))
	if action == "" {
		return nil, validationError{Field: "action", Message: "action is required"}
	}
	scopeRef := strings.TrimSpace(input.ScopeRef)
	if scopeRef == "" {
		return nil, validationError{Field: "scope_ref", Message: "scope reference is required"}
	}
	jobType := jobTypeForAction(input.Action)
	scheduleInput := ScheduleSafeOpInput{
		TenantUuid:    input.TenantUuid,
		Environment:   input.Environment,
		JobType:       jobType,
		TriggerSource: TriggerSourceManual,
		Action:        input.Action,
		ScopeType:     input.ScopeType,
		ScopeRef:      scopeRef,
		TargetID:      strings.TrimSpace(input.TargetID),
		Reason:        strings.TrimSpace(input.Reason),
		DryRun:        input.DryRun,
		Message:       input.Reason,
		Actor:         actor,
	}

	run, err := s.jobs.ScheduleSafeOp(ctx, scheduleInput)
	if err != nil {
		s.recordMetric(input.Action, "error")
		return nil, err
	}

	audit, err := s.createAuditEvent(ctx, input, run)
	if err != nil {
		_ = s.jobs.UpdateRunStatus(ctx, UpdateRunStatusInput{
			RunID:   run.ID,
			Status:  JobStatusFailed,
			Message: fmt.Sprintf("safe-op audit failure: %v", err),
		})
		s.recordMetric(input.Action, "error")
		return nil, err
	}
	if err := s.jobRepo.AttachAuditEvent(ctx, run.ID, audit.ID); err != nil {
		_ = s.jobs.UpdateRunStatus(ctx, UpdateRunStatusInput{
			RunID:   run.ID,
			Status:  JobStatusFailed,
			Message: fmt.Sprintf("failed to link audit event: %v", err),
		})
		s.recordMetric(input.Action, "error")
		return nil, err
	}
	run.AuditEventID = audit.ID
	s.recordMetric(input.Action, "scheduled")
	return run, nil
}

func (s *SafeOpsService) createAuditEvent(ctx context.Context, request SafeOpRequest, run *JobRunRecord) (*model.AuditEvent, error) {
	now := s.now()
	resourceRef := run.ID
	audit := &model.AuditEvent{
		ID:             uuid.NewString(),
		PluginID:       app.PluginID,
		ActorID:        request.Actor.ID,
		PermissionCode: request.Actor.PermissionCode,
		Action:         auditActionForSafeOp(request.Action),
		ResourceType:   "job.run",
		ResourceRef:    strPtr(resourceRef),
		Summary:        strPtr(s.summary(request)),
		OccurredAt:     now,
		CreatedAt:      now,
	}
	if request.TenantUuid != nil && strings.TrimSpace(*request.TenantUuid) != "" {
		clean := strings.TrimSpace(*request.TenantUuid)
		audit.TenantUuid = &clean
	}
	if strings.TrimSpace(request.Actor.Name) != "" {
		audit.ActorName = strPtr(request.Actor.Name)
	}
	if strings.TrimSpace(request.Actor.Email) != "" {
		audit.ActorEmail = strPtr(request.Actor.Email)
	}
	diff := map[string]any{
		"scope_type": request.ScopeType,
		"scope_ref":  request.ScopeRef,
		"target_id":  request.TargetID,
		"reason":     request.Reason,
		"dry_run":    request.DryRun,
	}
	if body, err := json.Marshal(diff); err == nil {
		audit.Diff = datatypes.JSON(body)
	}
	if err := s.audits.Create(ctx, audit); err != nil {
		return nil, err
	}
	return audit, nil
}

func (s *SafeOpsService) summary(request SafeOpRequest) string {
	scope := fmt.Sprintf("%s:%s", request.ScopeType, request.ScopeRef)
	action := strings.ToUpper(string(request.Action))
	if request.DryRun {
		action = action + " (DRY-RUN)"
	}
	if strings.TrimSpace(request.Reason) != "" {
		return fmt.Sprintf("%s requested for %s — %s", action, scope, request.Reason)
	}
	return fmt.Sprintf("%s requested for %s", action, scope)
}

func (s *SafeOpsService) recordMetric(action SafeOpAction, outcome string) {
	if s.metrics == nil {
		return
	}
	s.metrics.RecordSafeOp(string(action), outcome)
}

func (s *SafeOpsService) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc()
	}
	return time.Now()
}

func jobTypeForAction(action SafeOpAction) JobType {
	switch action {
	case SafeOpActionReplay:
		return JobTypeWebhookReplay
	case SafeOpActionRetry:
		return JobTypeTaskRetry
	case SafeOpActionDrain:
		return JobTypeQueueDrain
	case SafeOpActionDisable:
		return JobTypeCustom
	default:
		return JobTypeCustom
	}
}

func auditActionForSafeOp(action SafeOpAction) string {
	switch action {
	case SafeOpActionReplay:
		return "SAFE_OP_REPLAY"
	case SafeOpActionRetry:
		return "SAFE_OP_RETRY"
	case SafeOpActionDrain:
		return "SAFE_OP_DRAIN"
	case SafeOpActionDisable:
		return "SAFE_OP_DISABLE"
	default:
		return "SAFE_OP_EXECUTE"
	}
}
