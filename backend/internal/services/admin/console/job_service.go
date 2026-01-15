package console

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	consolerepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/admin_console"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// JobStatus represents persisted job run status values.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// SafeOpScope enumerates safe-operation scope types.
type SafeOpScope string

const (
	SafeOpScopeTenant       SafeOpScope = "tenant"
	SafeOpScopeEnvironment  SafeOpScope = "environment"
	SafeOpScopeSubscription SafeOpScope = "subscription"
)

// SafeOpAction enumerates supported safe-operation actions.
type SafeOpAction string

const (
	SafeOpActionReplay  SafeOpAction = "replay"
	SafeOpActionRetry   SafeOpAction = "retry"
	SafeOpActionDrain   SafeOpAction = "drain"
	SafeOpActionDisable SafeOpAction = "disable"
)

// JobType enumerates stored job types.
type JobType string

const (
	JobTypeWebhookReplay JobType = "webhook_replay"
	JobTypeTaskRetry     JobType = "task_retry"
	JobTypeQueueDrain    JobType = "queue_drain"
	JobTypeHealthProbe   JobType = "health_probe"
	JobTypeCustom        JobType = "custom"
)

// TriggerSource enumerates how a job was triggered.
type TriggerSource string

const (
	TriggerSourceManual   TriggerSource = "manual"
	TriggerSourceSchedule TriggerSource = "schedule"
	TriggerSourceAlert    TriggerSource = "alert"
	TriggerSourceAPI      TriggerSource = "api"
)

var (
	// ErrOperationInProgress indicates a conflicting safe-op is executing.
	ErrOperationInProgress = errors.New("safe operation already in progress for target scope")
	// ErrRetryNotAllowed indicates a run cannot be retried.
	ErrRetryNotAllowed = errors.New("job run cannot be retried")
	// ErrJobRunNotFound indicates the referenced job run does not exist.
	ErrJobRunNotFound = errors.New("job run not found")
	// ErrJobServiceUnavailable indicates the service lacks required dependencies.
	ErrJobServiceUnavailable = errors.New("job service unavailable")
)

// SafeOpLocker coordinates advisory locks for safe operations.
type SafeOpLocker interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key string) error
}

// memoryLocker is the default in-process locker used for development/tests.
type memoryLocker struct {
	mu    sync.Mutex
	locks map[string]time.Time
}

func newMemoryLocker() *memoryLocker {
	return &memoryLocker{
		locks: make(map[string]time.Time),
	}
}

func (m *memoryLocker) TryLock(_ context.Context, key string, ttl time.Duration) (bool, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	if expiry, ok := m.locks[key]; ok {
		if expiry.After(now) {
			return false, nil
		}
	}
	m.locks[key] = now.Add(ttl)
	return true, nil
}

func (m *memoryLocker) Unlock(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.locks, key)
	return nil
}

// JobService coordinates job history persistence and retry orchestration.
type JobService struct {
	cfg     *config.Config
	repo    *consolerepo.JobRunRepository
	locker  SafeOpLocker
	metrics *adminmetrics.Metrics
	nowFunc func() time.Time

	lockTTL time.Duration
	locks   sync.Map // runID -> lockKey
}

// JobServiceOption customises job service construction.
type JobServiceOption func(*JobService)

// WithLocker overrides the default safe-op locker.
func WithLocker(locker SafeOpLocker) JobServiceOption {
	return func(s *JobService) {
		if locker != nil {
			s.locker = locker
		}
	}
}

// WithJobClock allows tests to control time for job scheduling.
func WithJobClock(now func() time.Time) JobServiceOption {
	return func(s *JobService) {
		if now != nil {
			s.nowFunc = now
		}
	}
}

// NewJobService constructs a job service using shared dependencies.
func NewJobService(deps *app.Deps, opts ...JobServiceOption) *JobService {
	if deps == nil || deps.DB == nil {
		return &JobService{}
	}
	locker := SafeOpLocker(newMemoryLocker())
	metrics := deps.AdminConsoleMetrics
	if metrics == nil {
		metrics = adminmetrics.NewMetrics()
	}
	service := &JobService{
		cfg:     deps.Config,
		repo:    consolerepo.NewJobRunRepository(deps.DB),
		locker:  locker,
		metrics: metrics,
		nowFunc: time.Now,
	}
	for _, opt := range opts {
		opt(service)
	}
	lockTTL := time.Duration(120) * time.Second
	if deps != nil && deps.Config != nil {
		lockTTL = time.Duration(deps.Config.AdminConsoleSafeOpsLockTTL()) * time.Second
	}
	service.lockTTL = lockTTL
	return service
}

// SafeOpDetails describes the contextual information for a safe operation.
type SafeOpDetails struct {
	Action    SafeOpAction `json:"action"`
	ScopeType SafeOpScope  `json:"scope_type"`
	ScopeRef  string       `json:"scope_ref"`
	TargetID  string       `json:"target_id,omitempty"`
	Reason    string       `json:"reason,omitempty"`
	DryRun    bool         `json:"dry_run,omitempty"`
}

// JobRunRecord represents a job run with derived metadata.
type JobRunRecord struct {
	ID             string        `json:"id"`
	PluginID       string        `json:"plugin_id"`
	TenantUuid     string        `json:"tenant_uuid,omitempty"`
	Environment    string        `json:"environment,omitempty"`
	JobType        JobType       `json:"job_type"`
	TriggerSource  TriggerSource `json:"trigger_source"`
	Status         JobStatus     `json:"status"`
	StartedAt      *time.Time    `json:"started_at,omitempty"`
	FinishedAt     *time.Time    `json:"finished_at,omitempty"`
	DurationMillis int64         `json:"duration_ms,omitempty"`
	Message        string        `json:"message,omitempty"`
	RetryOf        string        `json:"retry_of,omitempty"`
	AuditEventID   string        `json:"audit_event_id,omitempty"`
	CreatedBy      string        `json:"created_by"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	SafeOp         SafeOpDetails `json:"safe_op"`
}

// ScheduleSafeOpInput captures parameters for safe operation scheduling.
type ScheduleSafeOpInput struct {
	TenantUuid    *string
	Environment   string
	JobType       JobType
	TriggerSource TriggerSource
	Action        SafeOpAction
	ScopeType     SafeOpScope
	ScopeRef      string
	TargetID      string
	Reason        string
	DryRun        bool
	Message       string
	Actor         Actor
}

// RetryRunInput describes retry invocation parameters.
type RetryRunInput struct {
	RunID      string
	TenantUuid *string
	Actor      Actor
}

// UpdateRunStatusInput updates job run status lifecycle.
type UpdateRunStatusInput struct {
	RunID      string
	Status     JobStatus
	Message    string
	StartedAt  *time.Time
	FinishedAt *time.Time
}

// ScheduleSafeOp persists a new job run representing a safe operation.
func (s *JobService) ScheduleSafeOp(ctx context.Context, input ScheduleSafeOpInput) (*JobRunRecord, error) {
	if s == nil || s.repo == nil {
		return nil, ErrJobServiceUnavailable
	}
	if err := validateSafeOpInput(input); err != nil {
		return nil, err
	}
	if input.TriggerSource == "" {
		input.TriggerSource = TriggerSourceManual
	}
	key := LockKey(input)
	acquired, err := s.locker.TryLock(ctx, key, s.lockTTL)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, ErrOperationInProgress
	}
	if err := s.checkActiveRun(ctx, input); err != nil {
		_ = s.locker.Unlock(ctx, key)
		return nil, err
	}
	run := s.buildRun(input, nil)
	if err := s.repo.Create(ctx, run); err != nil {
		_ = s.locker.Unlock(ctx, key)
		return nil, err
	}
	s.locks.Store(run.ID, key)
	return mapJobRunRecord(run, input.ScopeType, input.ScopeRef, input.Action, input.TargetID, input.Reason, input.DryRun), nil
}

// RetryRun creates a new job run retrying the provided run identifier.
func (s *JobService) RetryRun(ctx context.Context, input RetryRunInput) (*JobRunRecord, error) {
	if s == nil || s.repo == nil {
		return nil, ErrJobServiceUnavailable
	}
	if strings.TrimSpace(input.RunID) == "" {
		return nil, fmt.Errorf("run id is required")
	}
	existing, err := s.repo.GetByID(ctx, input.RunID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrJobRunNotFound
	}
	if !isRetryableStatus(existing.Status) {
		return nil, ErrRetryNotAllowed
	}
	if input.TenantUuid != nil {
		if existing.TenantUuid == nil || strings.TrimSpace(*existing.TenantUuid) != strings.TrimSpace(*input.TenantUuid) {
			return nil, ErrRetryNotAllowed
		}
	}
	scopeType := SafeOpScope(existing.ScopeType)
	if scopeType == "" {
		scopeType = SafeOpScopeTenant
	}
	scopeRef := ""
	if existing.ScopeRef != nil {
		scopeRef = *existing.ScopeRef
	}
	action := SafeOpAction(existing.Action)
	if action == "" {
		action = SafeOpActionRetry
	}
	scheduleInput := ScheduleSafeOpInput{
		TenantUuid:    existing.TenantUuid,
		Environment:   existing.Environment,
		JobType:       JobType(existing.JobType),
		TriggerSource: TriggerSourceManual,
		Action:        action,
		ScopeType:     scopeType,
		ScopeRef:      scopeRef,
		TargetID:      deref(existing.TargetID),
		Reason:        deref(existing.Reason),
		DryRun:        existing.DryRun,
		Message:       deref(existing.Message),
		Actor:         input.Actor,
	}
	key := LockKey(scheduleInput)
	acquired, err := s.locker.TryLock(ctx, key, s.lockTTL)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, ErrOperationInProgress
	}
	if err := s.checkActiveRun(ctx, scheduleInput); err != nil {
		_ = s.locker.Unlock(ctx, key)
		return nil, err
	}
	run := s.buildRun(scheduleInput, &existing.ID)
	if err := s.repo.Create(ctx, run); err != nil {
		_ = s.locker.Unlock(ctx, key)
		return nil, err
	}
	s.locks.Store(run.ID, key)
	return mapJobRunRecord(run, scheduleInput.ScopeType, scheduleInput.ScopeRef, scheduleInput.Action, scheduleInput.TargetID, scheduleInput.Reason, scheduleInput.DryRun), nil
}

// UpdateRunStatus updates persisted status for a run and releases locks when complete.
func (s *JobService) UpdateRunStatus(ctx context.Context, input UpdateRunStatusInput) error {
	if s == nil || s.repo == nil {
		return ErrJobServiceUnavailable
	}
	if strings.TrimSpace(input.RunID) == "" {
		return fmt.Errorf("run id is required")
	}
	run, err := s.repo.UpdateStatus(ctx, input.RunID, string(input.Status), strPtr(input.Message), input.StartedAt, input.FinishedAt)
	if err != nil {
		return err
	}
	if run != nil && s.metrics != nil {
		outcome := strings.ToLower(string(input.Status))
		if strings.TrimSpace(run.Action) != "" {
			s.metrics.RecordSafeOp(strings.TrimSpace(run.Action), outcome)
		}
	}
	if isTerminalStatus(input.Status) {
		if key, ok := s.locks.Load(input.RunID); ok {
			_ = s.locker.Unlock(ctx, key.(string))
			s.locks.Delete(input.RunID)
		}
	}
	return nil
}

// ListRuns fetches job runs using repository filters.
func (s *JobService) ListRuns(ctx context.Context, tenantID *string, jobType string, status string, cursor string, limit int) ([]JobRunRecord, string, error) {
	if s == nil || s.repo == nil {
		return nil, "", ErrJobServiceUnavailable
	}
	opts := consolerepo.JobRunListOptions{
		PluginID:   app.PluginID,
		TenantUuid: tenantID,
		JobType:    jobType,
		Status:     status,
		Cursor:     cursor,
		Limit:      limit,
	}
	runs, next, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, "", err
	}
	result := make([]JobRunRecord, len(runs))
	for i, run := range runs {
		scopeType := SafeOpScope(run.ScopeType)
		scopeRef := deref(run.ScopeRef)
		action := SafeOpAction(run.Action)
		target := deref(run.TargetID)
		reason := deref(run.Reason)
		if mapped := mapJobRunRecord(&runs[i], scopeType, scopeRef, action, target, reason, run.DryRun); mapped != nil {
			result[i] = *mapped
		}
	}
	return result, next, nil
}

func (s *JobService) checkActiveRun(ctx context.Context, input ScheduleSafeOpInput) error {
	active, err := s.repo.HasActiveRun(ctx, app.PluginID, input.TenantUuid, string(input.Action), string(input.ScopeType), input.ScopeRef)
	if err != nil {
		return err
	}
	if active {
		return ErrOperationInProgress
	}
	return nil
}

func (s *JobService) buildRun(input ScheduleSafeOpInput, retryOf *string) *model.JobRun {
	now := s.now()
	runID := uuid.NewString()
	scopeRef := strings.TrimSpace(input.ScopeRef)
	var tenantID *string
	if input.TenantUuid != nil && strings.TrimSpace(*input.TenantUuid) != "" {
		clean := strings.TrimSpace(*input.TenantUuid)
		tenantID = &clean
	}
	var retryPtr *string
	if retryOf != nil && strings.TrimSpace(*retryOf) != "" {
		clean := strings.TrimSpace(*retryOf)
		retryPtr = &clean
	}
	target := strings.TrimSpace(input.TargetID)
	reason := strings.TrimSpace(input.Reason)
	metadata := safeOpMetadata{
		Action:    string(input.Action),
		ScopeType: string(input.ScopeType),
		ScopeRef:  scopeRef,
		TargetID:  target,
		Reason:    reason,
		DryRun:    input.DryRun,
	}
	payload, _ := json.Marshal(metadata)
	return &model.JobRun{
		ID:            runID,
		PluginID:      app.PluginID,
		TenantUuid:    tenantID,
		Environment:   strings.TrimSpace(input.Environment),
		JobType:       string(input.JobType),
		TriggerSource: string(input.TriggerSource),
		Status:        string(JobStatusPending),
		Action:        string(input.Action),
		ScopeType:     string(input.ScopeType),
		ScopeRef:      strPtr(scopeRef),
		TargetID:      strPtr(target),
		Reason:        strPtr(reason),
		DryRun:        input.DryRun,
		Message:       strPtr(strings.TrimSpace(input.Message)),
		RetryOf:       retryPtr,
		CreatedBy:     input.Actor.ID,
		CreatedAt:     now,
		UpdatedAt:     now,
		Metadata:      datatypes.JSON(payload),
	}
}

func (s *JobService) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc()
	}
	return time.Now()
}

func validateSafeOpInput(input ScheduleSafeOpInput) error {
	if strings.TrimSpace(input.ScopeRef) == "" {
		return validationError{Field: "scope_ref", Message: "scope reference is required"}
	}
	if input.Action == "" {
		return validationError{Field: "action", Message: "action is required"}
	}
	if input.JobType == "" {
		return validationError{Field: "job_type", Message: "job type is required"}
	}
	if input.TriggerSource == "" {
		input.TriggerSource = TriggerSourceManual
	}
	if strings.TrimSpace(input.Actor.ID) == "" {
		return validationError{Field: "actor_id", Message: "actor id required"}
	}
	return nil
}

func isTerminalStatus(status JobStatus) bool {
	switch status {
	case JobStatusSucceeded, JobStatusFailed, JobStatusCancelled:
		return true
	default:
		return false
	}
}

func isRetryableStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(JobStatusFailed), string(JobStatusCancelled):
		return true
	default:
		return false
	}
}

// LockKey returns a deterministic advisory lock key for a safe-op input.
func LockKey(input ScheduleSafeOpInput) string {
	scope := strings.TrimSpace(input.ScopeRef)
	return fmt.Sprintf("%s|%s|%s", app.PluginID, scope, strings.TrimSpace(string(input.Action)))
}

type safeOpMetadata struct {
	Action    string `json:"action"`
	ScopeType string `json:"scope_type"`
	ScopeRef  string `json:"scope_ref"`
	TargetID  string `json:"target_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
}

func mapJobRunRecord(run *model.JobRun, scopeType SafeOpScope, scopeRef string, action SafeOpAction, target string, reason string, dryRun bool) *JobRunRecord {
	if run == nil {
		return nil
	}
	record := &JobRunRecord{
		ID:             run.ID,
		PluginID:       run.PluginID,
		Environment:    run.Environment,
		JobType:        JobType(run.JobType),
		TriggerSource:  TriggerSource(run.TriggerSource),
		Status:         JobStatus(run.Status),
		StartedAt:      run.StartedAt,
		FinishedAt:     run.FinishedAt,
		DurationMillis: run.DurationMillis,
		CreatedBy:      run.CreatedBy,
		CreatedAt:      run.CreatedAt,
		UpdatedAt:      run.UpdatedAt,
		SafeOp: SafeOpDetails{
			Action:    action,
			ScopeType: scopeType,
			ScopeRef:  strings.TrimSpace(scopeRef),
			TargetID:  strings.TrimSpace(target),
			Reason:    strings.TrimSpace(reason),
			DryRun:    dryRun,
		},
	}
	if run.TenantUuid != nil {
		record.TenantUuid = *run.TenantUuid
	}
	if run.Message != nil {
		record.Message = *run.Message
	}
	if run.RetryOf != nil {
		record.RetryOf = *run.RetryOf
	}
	if run.AuditEventID != nil {
		record.AuditEventID = *run.AuditEventID
	}
	if scopeType == "" && strings.TrimSpace(run.ScopeType) != "" {
		record.SafeOp.ScopeType = SafeOpScope(run.ScopeType)
	}
	if record.SafeOp.ScopeRef == "" && run.ScopeRef != nil {
		record.SafeOp.ScopeRef = strings.TrimSpace(*run.ScopeRef)
	}
	if record.SafeOp.Action == "" && strings.TrimSpace(run.Action) != "" {
		record.SafeOp.Action = SafeOpAction(run.Action)
	}
	if record.SafeOp.TargetID == "" && run.TargetID != nil {
		record.SafeOp.TargetID = strings.TrimSpace(*run.TargetID)
	}
	if record.SafeOp.Reason == "" && run.Reason != nil {
		record.SafeOp.Reason = strings.TrimSpace(*run.Reason)
	}
	record.SafeOp.DryRun = dryRun || run.DryRun
	if len(run.Metadata) > 0 {
		var meta safeOpMetadata
		if err := json.Unmarshal(run.Metadata, &meta); err == nil {
			if record.SafeOp.Action == "" && meta.Action != "" {
				record.SafeOp.Action = SafeOpAction(meta.Action)
			}
			if record.SafeOp.ScopeType == "" && meta.ScopeType != "" {
				record.SafeOp.ScopeType = SafeOpScope(meta.ScopeType)
			}
			if record.SafeOp.ScopeRef == "" && meta.ScopeRef != "" {
				record.SafeOp.ScopeRef = strings.TrimSpace(meta.ScopeRef)
			}
			if record.SafeOp.TargetID == "" && meta.TargetID != "" {
				record.SafeOp.TargetID = meta.TargetID
			}
			if record.SafeOp.Reason == "" && meta.Reason != "" {
				record.SafeOp.Reason = meta.Reason
			}
			record.SafeOp.DryRun = record.SafeOp.DryRun || meta.DryRun
		}
	}
	return record
}
