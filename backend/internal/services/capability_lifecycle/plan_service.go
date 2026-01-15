package capability_lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/capability"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const lifecycleStorage = "../../contracts/exposure/capability-lifecycle.json"
const lifecycleStorageEnv = "POWERXPLUGIN_LIFECYCLE_STORAGE"

var (
	errCapabilityIDRequired = errors.New("capability id required")
	errDiffRequired         = errors.New("diff summary required")
	errChangeTypeInvalid    = errors.New("change type invalid")
	ErrPlanNotFound         = errors.New("lifecycle plan not found")
)

var allowedChangeTypes = []string{"upgrade", "deprecation", "rollback"}
var allowedStatuses = []string{"draft", "pending", "approved", "paused", "completed"}

// PlanService manages lifecycle plans for capability changes.
type PlanService struct {
	logger      *logrus.Entry
	notifier    capability.Notifier
	mu          sync.RWMutex
	plans       map[string]*LifecyclePlan
	storagePath string
}

// LifecyclePlan describes a published plan for capability upgrade/deprecation.
type LifecyclePlan struct {
	ID                   string            `json:"id"`
	CapabilityID         string            `json:"capability_id"`
	ChangeType           string            `json:"change_type"`
	DiffSummary          string            `json:"diff_summary"`
	NotificationChannels []string          `json:"notification_channels"`
	GracePeriodHours     int               `json:"grace_period_hours"`
	DualRunUntil         string            `json:"dual_run_until"`
	RollbackPlan         string            `json:"rollback_plan"`
	Windows              []RolloutWindow   `json:"windows"`
	Status               string            `json:"status"`
	CreatedBy            string            `json:"created_by"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

// RolloutWindow represents a single experimentation window.
type RolloutWindow struct {
	Label     string `json:"label"`
	StartAt   string `json:"start_at"`
	EndAt     string `json:"end_at"`
	Percent   int    `json:"percent"`
	Condition string `json:"condition,omitempty"`
}

// PlanTemplate enumerates allowed options for UI.
type PlanTemplate struct {
	ChangeTypes    []string `json:"change_types"`
	StatusOptions  []string `json:"status_options"`
	ChannelOptions []string `json:"channel_options"`
}

// PlanInput is accepted from HTTP handlers.
type PlanInput struct {
	CapabilityID         string            `json:"capability_id"`
	ChangeType           string            `json:"change_type"`
	DiffSummary          string            `json:"diff_summary"`
	NotificationChannels []string          `json:"notification_channels"`
	GracePeriodHours     int               `json:"grace_period_hours"`
	DualRunUntil         string            `json:"dual_run_until"`
	RollbackPlan         string            `json:"rollback_plan"`
	Windows              []RolloutWindow   `json:"windows"`
	Metadata             map[string]string `json:"metadata"`
	Actor                string            `json:"actor"`
}

// StatusInput updates plan status.
type StatusInput struct {
	PlanID string `json:"plan_id"`
	Status string `json:"status"`
	Actor  string `json:"actor"`
	Notes  string `json:"notes"`
}

// NewPlanService returns an instance backed by JSON storage.
func NewPlanService(deps *app.Deps) *PlanService {
	var log *logrus.Entry
	if deps != nil {
		log = deps.RuntimeLogger(deps.Ctx, "capability_lifecycle_service", nil)
	}
	if log == nil {
		log = logger.WithRuntimeFields(app.PluginID, "", "", "capability_lifecycle_service", nil)
	}
	svc := &PlanService{
		logger:      log,
		notifier:    capability.NewNotifier(log),
		plans:       map[string]*LifecyclePlan{},
		storagePath: resolveLifecycleStorage(),
	}
	svc.loadFromDisk()
	return svc
}

// Template returns change type/status/channel options.
func (s *PlanService) Template() *PlanTemplate {
	return &PlanTemplate{
		ChangeTypes:    append([]string{}, allowedChangeTypes...),
		StatusOptions:  append([]string{}, allowedStatuses...),
		ChannelOptions: []string{"email", "slack", "webhook", "host_banner"},
	}
}

// CreatePlan validates and persists a lifecycle plan.
func (s *PlanService) CreatePlan(ctx context.Context, input *PlanInput) (*LifecyclePlan, error) {
	if input == nil || strings.TrimSpace(input.CapabilityID) == "" {
		return nil, errCapabilityIDRequired
	}
	changeType := strings.ToLower(strings.TrimSpace(input.ChangeType))
	if !containsString(allowedChangeTypes, changeType) {
		return nil, errChangeTypeInvalid
	}
	if strings.TrimSpace(input.DiffSummary) == "" {
		return nil, errDiffRequired
	}
	now := time.Now().UTC()
	plan := &LifecyclePlan{
		ID:                   uuid.NewString(),
		CapabilityID:         strings.TrimSpace(input.CapabilityID),
		ChangeType:           changeType,
		DiffSummary:          strings.TrimSpace(input.DiffSummary),
		NotificationChannels: dedupeStrings(input.NotificationChannels),
		GracePeriodHours:     normalizeGrace(input.GracePeriodHours),
		DualRunUntil:         strings.TrimSpace(input.DualRunUntil),
		RollbackPlan:         strings.TrimSpace(input.RollbackPlan),
		Windows:              normalizeWindows(input.Windows),
		Status:               "pending",
		CreatedBy:            strings.TrimSpace(input.Actor),
		CreatedAt:            now,
		UpdatedAt:            now,
		Metadata:             input.Metadata,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[plan.ID] = plan
	if err := s.persistLocked(); err != nil {
		return nil, err
	}
	s.notifier.Emit(ctx, capability.Event{
		Type:         capability.EventLifecyclePlanCreated,
		CapabilityID: plan.CapabilityID,
		Status:       plan.Status,
		Channels:     plan.NotificationChannels,
		Payload: map[string]any{
			"plan_id":          plan.ID,
			"change_type":      plan.ChangeType,
			"grace_period":     plan.GracePeriodHours,
			"dual_run_until":   plan.DualRunUntil,
			"windows":          plan.Windows,
			"rollback_plan":    plan.RollbackPlan,
			"notification_set": plan.NotificationChannels,
		},
		Metadata: map[string]any{
			"actor": plan.CreatedBy,
		},
	})
	return clonePlan(plan), nil
}

// ListPlans returns plans for the capability (or all).
func (s *PlanService) ListPlans(capabilityID string) []*LifecyclePlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*LifecyclePlan
	for _, plan := range s.plans {
		if plan == nil {
			continue
		}
		if capabilityID != "" && !strings.EqualFold(plan.CapabilityID, capabilityID) {
			continue
		}
		out = append(out, clonePlan(plan))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// UpdateStatus changes a plan status.
func (s *PlanService) UpdateStatus(ctx context.Context, input *StatusInput) (*LifecyclePlan, error) {
	if input == nil || strings.TrimSpace(input.PlanID) == "" {
		return nil, ErrPlanNotFound
	}
	status := strings.ToLower(strings.TrimSpace(input.Status))
	if !containsString(allowedStatuses, status) {
		return nil, errors.New("invalid status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[input.PlanID]
	if !ok || plan == nil {
		return nil, ErrPlanNotFound
	}
	plan.Status = status
	plan.UpdatedAt = time.Now().UTC()
	if err := s.persistLocked(); err != nil {
		return nil, err
	}
	s.notifier.Emit(ctx, capability.Event{
		Type:         capability.EventLifecyclePlanUpdated,
		CapabilityID: plan.CapabilityID,
		Status:       status,
		Channels:     plan.NotificationChannels,
		Payload: map[string]any{
			"plan_id": plan.ID,
			"status":  status,
			"notes":   strings.TrimSpace(input.Notes),
		},
		Metadata: map[string]any{
			"actor": input.Actor,
		},
	})
	return clonePlan(plan), nil
}

func (s *PlanService) loadFromDisk() {
	data, err := os.ReadFile(s.storagePath)
	if err != nil {
		if !os.IsNotExist(err) {
			s.logger.WithError(err).Warn("failed to read lifecycle plans")
		}
		return
	}
	var snapshot lifecycleSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		s.logger.WithError(err).Warn("failed to parse lifecycle plans")
		return
	}
	for _, plan := range snapshot.Plans {
		if plan == nil || plan.ID == "" {
			continue
		}
		s.plans[plan.ID] = plan
	}
}

func (s *PlanService) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0o755); err != nil {
		return err
	}
	list := make([]*LifecyclePlan, 0, len(s.plans))
	for _, plan := range s.plans {
		if plan == nil {
			continue
		}
		cp := *plan
		list = append(list, &cp)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.Before(list[j].CreatedAt)
	})
	snapshot := lifecycleSnapshot{
		Plans:     list,
		UpdatedAt: time.Now().UTC(),
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storagePath, data, 0o644)
}

func resolveLifecycleStorage() string {
	if override := strings.TrimSpace(os.Getenv(lifecycleStorageEnv)); override != "" {
		return filepath.Clean(override)
	}
	workDir, err := os.Getwd()
	if err != nil {
		return lifecycleStorage
	}
	return filepath.Clean(filepath.Join(workDir, lifecycleStorage))
}

func containsString(items []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item)) == target {
			return true
		}
	}
	return false
}

func dedupeStrings(items []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, item := range items {
		str := strings.TrimSpace(item)
		if str == "" {
			continue
		}
		key := strings.ToLower(str)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, str)
	}
	return out
}

func normalizeGrace(value int) int {
	if value <= 0 {
		return 72
	}
	return value
}

func normalizeWindows(windows []RolloutWindow) []RolloutWindow {
	if len(windows) == 0 {
		return nil
	}
	var out []RolloutWindow
	for _, w := range windows {
		if strings.TrimSpace(w.StartAt) == "" || strings.TrimSpace(w.EndAt) == "" {
			continue
		}
		if w.Percent < 0 {
			w.Percent = 0
		}
		if w.Percent > 100 {
			w.Percent = 100
		}
		w.Label = strings.TrimSpace(w.Label)
		w.Condition = strings.TrimSpace(w.Condition)
		out = append(out, w)
	}
	return out
}

func clonePlan(plan *LifecyclePlan) *LifecyclePlan {
	if plan == nil {
		return nil
	}
	cp := *plan
	if plan.Windows != nil {
		cp.Windows = append([]RolloutWindow{}, plan.Windows...)
	}
	if plan.NotificationChannels != nil {
		cp.NotificationChannels = append([]string{}, plan.NotificationChannels...)
	}
	if plan.Metadata != nil {
		cp.Metadata = map[string]string{}
		for k, v := range plan.Metadata {
			cp.Metadata[k] = v
		}
	}
	return &cp
}

type lifecycleSnapshot struct {
	Plans     []*LifecyclePlan `json:"plans"`
	UpdatedAt time.Time        `json:"updated_at"`
}
