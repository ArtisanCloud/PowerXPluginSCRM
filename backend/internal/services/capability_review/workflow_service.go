package capability_review

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	capobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/capability"
	capsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/capability"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	StatusPending          = "pending"
	StatusInReview         = "in_review"
	StatusChangesRequested = "changes_requested"
	StatusApproved         = "approved"
	StatusRejected         = "rejected"
)

var finalStatuses = map[string]struct{}{
	StatusApproved: {},
	StatusRejected: {},
}

// Errors returned by the workflow service.
var (
	ErrTaskNotFound      = errors.New("capability_review: task not found")
	ErrCapabilityMissing = errors.New("capability_review: capability record required")
	ErrInvalidDecision   = errors.New("capability_review: invalid decision")
	ErrInvalidComment    = errors.New("capability_review: comment required")
)

type reviewPolicy struct {
	Roles          []string
	SLA            time.Duration
	EscalationLead time.Duration
	RiskScore      int
	MinReviewers   int
}

// WorkflowService coordinates review task generation and lifecycle management.
type WorkflowService struct {
	logger  *logrus.Entry
	metrics *capobs.Metrics

	mu            sync.RWMutex
	tasks         map[string]*ReviewTask
	byCapability  map[string][]string
	capabilityRef map[string]*capsvc.CapabilityRecord
	policies      map[string]reviewPolicy
	now           func() time.Time
}

// ReviewTask describes a single review assignment.
type ReviewTask struct {
	ID             string        `json:"id"`
	CapabilityID   string        `json:"capability_id"`
	Role           string        `json:"role"`
	Sensitivity    string        `json:"sensitivity"`
	Assignees      []string      `json:"assignees"`
	Status         string        `json:"status"`
	SLADeadline    time.Time     `json:"sla_due_at"`
	RiskScore      int           `json:"risk_score"`
	Comments       []TaskComment `json:"comments"`
	Attachments    []TaskAsset   `json:"attachments"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	LastEscalation time.Time     `json:"last_escalated_at"`
	ReworkCount    int           `json:"rework_count"`
	DecisionBy     string        `json:"decision_by,omitempty"`
	Escalations    []Escalation  `json:"escalations,omitempty"`
	OwnerEmail     string        `json:"owner_email,omitempty"`
	TenantScope    string        `json:"tenant_scope,omitempty"`
	Tags           []string      `json:"tags,omitempty"`
	Scenario       string        `json:"scenario,omitempty"`
	LastDecisionAt time.Time     `json:"last_decision_at,omitempty"`
	LastDecision   string        `json:"last_decision,omitempty"`
	PolicyRef      reviewPolicy  `json:"-"`
}

// TaskComment captures reviewer/author remarks.
type TaskComment struct {
	ID          string      `json:"id"`
	Author      string      `json:"author"`
	Message     string      `json:"message"`
	Attachments []TaskAsset `json:"attachments"`
	CreatedAt   time.Time   `json:"created_at"`
}

// TaskAsset represents an attachment or supporting file.
type TaskAsset struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// CommentInput defines a comment payload.
type CommentInput struct {
	Author      string
	Message     string
	Attachments []AttachmentInput
}

// AttachmentInput is used when adding assets.
type AttachmentInput struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// DecisionInput captures the reviewer decision payload.
type DecisionInput struct {
	Actor       string
	Decision    string
	Note        string
	Attachments []AttachmentInput
}

// ResubmitInput records developer remediation information.
type ResubmitInput struct {
	Actor       string
	Note        string
	Attachments []AttachmentInput
}

// Escalation describes an SLA escalation event.
type Escalation struct {
	TaskID       string    `json:"task_id"`
	CapabilityID string    `json:"capability_id"`
	Reason       string    `json:"reason"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewWorkflowService builds a service instance with default policies.
func NewWorkflowService(deps *app.Deps) *WorkflowService {
	var log *logrus.Entry
	var metrics *capobs.Metrics
	if deps != nil {
		log = deps.RuntimeLogger(deps.Ctx, "capability_review_service", nil)
		metrics = deps.CapabilityMetrics
	}
	if log == nil {
		log = logger.WithRuntimeFields(app.PluginID, "", "", "capability_review_service", nil)
	}
	return &WorkflowService{
		logger:        log,
		metrics:       metrics,
		tasks:         make(map[string]*ReviewTask),
		byCapability:  make(map[string][]string),
		capabilityRef: make(map[string]*capsvc.CapabilityRecord),
		policies:      defaultPolicies(),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// SetClock overrides the internal clock (primarily for tests).
func (s *WorkflowService) SetClock(fn func() time.Time) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = fn
}

// EnsureTasks generates review tasks per capability sensitivity.
func (s *WorkflowService) EnsureTasks(ctx context.Context, record *capsvc.CapabilityRecord) ([]*ReviewTask, error) {
	if record == nil || strings.TrimSpace(record.ID) == "" {
		return nil, ErrCapabilityMissing
	}

	sensitivity := strings.ToLower(strings.TrimSpace(record.Sensitivity))
	if sensitivity == "" {
		sensitivity = "medium"
	}
	policy, ok := s.policies[sensitivity]
	if !ok {
		policy = s.policies["medium"]
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if ids, ok := s.byCapability[record.ID]; ok && len(ids) > 0 {
		return s.cloneTasksLocked(ids), nil
	}

	now := s.now()
	var taskIDs []string
	for _, role := range policy.Roles {
		task := &ReviewTask{
			ID:           uuid.NewString(),
			CapabilityID: record.ID,
			Role:         role,
			Sensitivity:  sensitivity,
			Assignees:    defaultAssignees(role, policy.MinReviewers),
			Status:       StatusPending,
			SLADeadline:  now.Add(policy.SLA),
			RiskScore:    policy.RiskScore,
			CreatedAt:    now,
			UpdatedAt:    now,
			OwnerEmail:   record.Owner.Email,
			TenantScope:  record.TenantScope,
			Tags:         append([]string{}, record.Tags...),
			Scenario:     record.Scenario,
			PolicyRef:    policy,
		}
		s.tasks[task.ID] = task
		taskIDs = append(taskIDs, task.ID)
		capobs.EmitReviewEvent(ctx, s.logger, capobs.Event{
			Type:         capobs.EventTaskCreated,
			CapabilityID: record.ID,
			TaskID:       task.ID,
			Status:       task.Status,
			Message:      role + " review task created",
			Deadline:     task.SLADeadline,
			Metadata: map[string]any{
				"role":        role,
				"sensitivity": sensitivity,
			},
		})
	}
	s.byCapability[record.ID] = taskIDs
	s.capabilityRef[record.ID] = cloneRecord(record)

	return s.cloneTasksLocked(taskIDs), nil
}

// ListTasks returns review tasks for a capability.
func (s *WorkflowService) ListTasks(capabilityID string) []*ReviewTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cloneTasksLocked(s.byCapability[capabilityID])
}

// AddComment appends a comment to the task and moves it into in_review.
func (s *WorkflowService) AddComment(ctx context.Context, taskID string, input CommentInput) (*ReviewTask, error) {
	if strings.TrimSpace(input.Message) == "" {
		return nil, ErrInvalidComment
	}
	task, err := s.getTask(taskID)
	if err != nil {
		return nil, err
	}

	now := s.now()
	comment := TaskComment{
		ID:        uuid.NewString(),
		Author:    strings.TrimSpace(input.Author),
		Message:   strings.TrimSpace(input.Message),
		CreatedAt: now,
	}
	for _, att := range input.Attachments {
		if strings.TrimSpace(att.Name) == "" && strings.TrimSpace(att.URL) == "" {
			continue
		}
		comment.Attachments = append(comment.Attachments, TaskAsset{
			Name: strings.TrimSpace(att.Name),
			URL:  strings.TrimSpace(att.URL),
		})
	}

	s.mu.Lock()
	task.Comments = append(task.Comments, comment)
	task.Status = StatusInReview
	task.UpdatedAt = now
	s.mu.Unlock()

	capobs.EmitReviewEvent(ctx, s.logger, capobs.Event{
		Type:         capobs.EventCommentAdded,
		CapabilityID: task.CapabilityID,
		TaskID:       task.ID,
		Status:       task.Status,
		Message:      comment.Message,
		Metadata: map[string]any{
			"author": comment.Author,
		},
	})

	return cloneTask(task), nil
}

// Resolve registers a reviewer decision.
func (s *WorkflowService) Resolve(ctx context.Context, taskID string, input DecisionInput) (*ReviewTask, error) {
	decision := strings.ToLower(strings.TrimSpace(input.Decision))
	if decision == "" {
		return nil, ErrInvalidDecision
	}
	task, err := s.getTask(taskID)
	if err != nil {
		return nil, err
	}

	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()

	switch decision {
	case "approve", "approved":
		task.Status = StatusApproved
	case "reject", "rejected":
		task.Status = StatusRejected
	case "request_changes", "changes_requested":
		task.Status = StatusChangesRequested
	default:
		return nil, ErrInvalidDecision
	}

	if note := strings.TrimSpace(input.Note); note != "" {
		task.Comments = append(task.Comments, TaskComment{
			ID:        uuid.NewString(),
			Author:    strings.TrimSpace(input.Actor),
			Message:   note,
			CreatedAt: now,
		})
	}
	if len(input.Attachments) > 0 {
		for _, att := range input.Attachments {
			if strings.TrimSpace(att.Name) == "" && strings.TrimSpace(att.URL) == "" {
				continue
			}
			task.Attachments = append(task.Attachments, TaskAsset{
				Name: strings.TrimSpace(att.Name),
				URL:  strings.TrimSpace(att.URL),
			})
		}
	}
	task.LastDecision = decision
	task.LastDecisionAt = now
	task.DecisionBy = strings.TrimSpace(input.Actor)
	task.UpdatedAt = now

	capobs.EmitReviewEvent(ctx, s.logger, capobs.Event{
		Type:         capobs.EventTaskUpdated,
		CapabilityID: task.CapabilityID,
		TaskID:       task.ID,
		Status:       task.Status,
		Message:      decision,
		Deadline:     task.SLADeadline,
		Metadata: map[string]any{
			"actor": task.DecisionBy,
		},
	})
	if s.metrics != nil {
		s.metrics.ObserveAsyncWorkflowDuration(task.CapabilityID, "review", task.Status, now.Sub(task.CreatedAt))
	}

	return cloneTask(task), nil
}

// Resubmit resets tasks to pending after developer remediation.
func (s *WorkflowService) Resubmit(ctx context.Context, capabilityID string, input ResubmitInput) ([]*ReviewTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskIDs, ok := s.byCapability[capabilityID]
	if !ok || len(taskIDs) == 0 {
		return nil, ErrCapabilityMissing
	}
	now := s.now()
	for _, id := range taskIDs {
		task := s.tasks[id]
		if task == nil {
			continue
		}
		if _, ok := finalStatuses[task.Status]; ok {
			continue
		}
		task.Status = StatusPending
		task.ReworkCount++
		task.UpdatedAt = now
		task.Comments = append(task.Comments, TaskComment{
			ID:        uuid.NewString(),
			Author:    strings.TrimSpace(input.Actor),
			Message:   strings.TrimSpace(input.Note),
			CreatedAt: now,
		})
		if len(input.Attachments) > 0 {
			for _, att := range input.Attachments {
				if strings.TrimSpace(att.Name) == "" && strings.TrimSpace(att.URL) == "" {
					continue
				}
				task.Attachments = append(task.Attachments, TaskAsset{
					Name: strings.TrimSpace(att.Name),
					URL:  strings.TrimSpace(att.URL),
				})
			}
		}
	}

	capobs.EmitReviewEvent(ctx, s.logger, capobs.Event{
		Type:         capobs.EventCapabilityResubmitted,
		CapabilityID: capabilityID,
		Status:       StatusPending,
		Message:      strings.TrimSpace(input.Note),
	})

	return s.cloneTasksLocked(taskIDs), nil
}

// EvaluateSLA inspects updated tasks and emits escalation events when due.
func (s *WorkflowService) EvaluateSLA(ctx context.Context, now time.Time) []Escalation {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.IsZero() {
		now = s.now()
	}
	var escalations []Escalation
	for _, task := range s.tasks {
		if task == nil {
			continue
		}
		if _, done := finalStatuses[task.Status]; done {
			continue
		}
		if task.PolicyRef.EscalationLead <= 0 {
			continue
		}
		if !task.LastEscalation.IsZero() {
			continue
		}
		if task.SLADeadline.Sub(now) > task.PolicyRef.EscalationLead {
			continue
		}
		esc := Escalation{
			TaskID:       task.ID,
			CapabilityID: task.CapabilityID,
			Reason:       "sla_escalation",
			Timestamp:    now,
		}
		task.LastEscalation = now
		task.Escalations = append(task.Escalations, esc)
		escalations = append(escalations, esc)
		capobs.EmitReviewEvent(ctx, s.logger, capobs.Event{
			Type:         capobs.EventTaskEscalated,
			CapabilityID: task.CapabilityID,
			TaskID:       task.ID,
			Status:       task.Status,
			Message:      "SLA escalation triggered",
			Deadline:     task.SLADeadline,
		})
	}
	return escalations
}

func (s *WorkflowService) getTask(id string) (*ReviewTask, error) {
	s.mu.RLock()
	task := s.tasks[id]
	s.mu.RUnlock()
	if task == nil {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (s *WorkflowService) cloneTasksLocked(ids []string) []*ReviewTask {
	if len(ids) == 0 {
		return nil
	}
	out := make([]*ReviewTask, 0, len(ids))
	for _, id := range ids {
		if task, ok := s.tasks[id]; ok && task != nil {
			out = append(out, cloneTask(task))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Role < out[j].Role
	})
	return out
}

func cloneTask(task *ReviewTask) *ReviewTask {
	if task == nil {
		return nil
	}
	cp := *task
	if task.Assignees != nil {
		cp.Assignees = append([]string{}, task.Assignees...)
	}
	if task.Comments != nil {
		cp.Comments = append([]TaskComment{}, task.Comments...)
		for i := range task.Comments {
			if len(task.Comments[i].Attachments) > 0 {
				cp.Comments[i].Attachments = append([]TaskAsset{}, task.Comments[i].Attachments...)
			}
		}
	}
	if task.Attachments != nil {
		cp.Attachments = append([]TaskAsset{}, task.Attachments...)
	}
	if task.Tags != nil {
		cp.Tags = append([]string{}, task.Tags...)
	}
	if task.Escalations != nil {
		cp.Escalations = append([]Escalation{}, task.Escalations...)
	}
	return &cp
}

func cloneRecord(rec *capsvc.CapabilityRecord) *capsvc.CapabilityRecord {
	if rec == nil {
		return nil
	}
	cp := *rec
	if rec.Tags != nil {
		cp.Tags = append([]string{}, rec.Tags...)
	}
	return &cp
}

func defaultPolicies() map[string]reviewPolicy {
	return map[string]reviewPolicy{
		"low": {
			Roles:          []string{"operations"},
			SLA:            36 * time.Hour,
			EscalationLead: 6 * time.Hour,
			RiskScore:      20,
			MinReviewers:   1,
		},
		"medium": {
			Roles:          []string{"security", "operations"},
			SLA:            48 * time.Hour,
			EscalationLead: 4 * time.Hour,
			RiskScore:      55,
			MinReviewers:   1,
		},
		"high": {
			Roles:          []string{"security", "compliance"},
			SLA:            48 * time.Hour,
			EscalationLead: 4 * time.Hour,
			RiskScore:      80,
			MinReviewers:   2,
		},
	}
}

func defaultAssignees(role string, min int) []string {
	role = strings.TrimSpace(role)
	if role == "" {
		role = "reviewer"
	}
	if min < 1 {
		min = 1
	}
	assignees := []string{}
	for i := 0; i < min; i++ {
		suffix := "lead"
		if i > 0 {
			suffix = "backup-" + strconv.Itoa(i)
		}
		assignees = append(assignees, role+"-"+suffix)
	}
	return assignees
}
