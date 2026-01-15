package operations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	oprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/operations"
	opmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/operations"
	runtimeops "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"gorm.io/datatypes"
)

// IncidentDispatcher emits incident notifications externally.
type IncidentDispatcher interface {
	DispatchIncidentEvent(ctx context.Context, eventType string, incident *opmodels.Incident, payload map[string]any) error
}

type noopIncidentDispatcher struct{}

func (noopIncidentDispatcher) DispatchIncidentEvent(ctx context.Context, eventType string, incident *opmodels.Incident, payload map[string]any) error {
	return nil
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

const (
	IncidentSeveritySev0 = "sev0"
	IncidentSeveritySev1 = "sev1"
	IncidentSeveritySev2 = "sev2"
	IncidentSeveritySev3 = "sev3"
	IncidentSeveritySev4 = "sev4"

	IncidentStatusDetected     = "detected"
	IncidentStatusAcknowledged = "acknowledged"
	IncidentStatusMitigated    = "mitigated"
	IncidentStatusMonitoring   = "monitoring"
	IncidentStatusResolved     = "resolved"
	IncidentStatusClosed       = "closed"
)

var validSeverities = map[string]struct{}{
	IncidentSeveritySev0: {},
	IncidentSeveritySev1: {},
	IncidentSeveritySev2: {},
	IncidentSeveritySev3: {},
	IncidentSeveritySev4: {},
}

var validStatuses = []string{
	IncidentStatusDetected,
	IncidentStatusAcknowledged,
	IncidentStatusMitigated,
	IncidentStatusMonitoring,
	IncidentStatusResolved,
	IncidentStatusClosed,
}

var severityUpdateCadence = map[string]time.Duration{
	IncidentSeveritySev0: 10 * time.Minute,
	IncidentSeveritySev1: 15 * time.Minute,
	IncidentSeveritySev2: 30 * time.Minute,
	IncidentSeveritySev3: 60 * time.Minute,
	IncidentSeveritySev4: 24 * time.Hour,
}

// IncidentService orchestrates incident lifecycle operations.
type IncidentService struct {
	cfg        *config.Config
	repo       *oprepo.IncidentRepository
	metrics    *opmetrics.Metrics
	dispatcher IncidentDispatcher
	readiness  runtimeops.ReadinessBlueprint
	httpClient httpDoer
}

// NewIncidentService builds the service.
func NewIncidentService(repo *oprepo.IncidentRepository, cfg *config.Config, metrics *opmetrics.Metrics, dispatcher IncidentDispatcher) *IncidentService {
	if metrics == nil {
		metrics = opmetrics.NewMetrics()
	}
	if dispatcher == nil {
		dispatcher = noopIncidentDispatcher{}
	}
	return &IncidentService{
		cfg:        cfg,
		repo:       repo,
		metrics:    metrics,
		dispatcher: dispatcher,
		readiness:  runtimeops.DefaultReadinessBlueprint(),
		httpClient: http.DefaultClient,
	}
}

// CreateIncidentRequest captures inputs for incident creation.
type CreateIncidentRequest struct {
	TenantUuid      *string         `json:"tenant_uuid"`
	Severity        string          `json:"severity"`
	DetectionSource string          `json:"detection_source"`
	Summary         string          `json:"summary"`
	Impact          map[string]any  `json:"impact"`
	Mitigation      string          `json:"mitigation"`
	Labels          map[string]bool `json:"labels"`
	Confidentiality string          `json:"confidentiality"`
	NextUpdateAt    *time.Time      `json:"next_update_at"`
}

// UpdateIncidentRequest captures mutable incident fields.
type UpdateIncidentRequest struct {
	Status          *string         `json:"status"`
	Mitigation      *string         `json:"mitigation"`
	RootCause       *string         `json:"root_cause"`
	NextUpdateAt    *time.Time      `json:"next_update_at"`
	Labels          map[string]bool `json:"labels"`
	Confidentiality *string         `json:"confidentiality"`
}

// TimelineEntryRequest describes timeline update payload.
type TimelineEntryRequest struct {
	EntryType          string         `json:"entry_type"`
	Message            string         `json:"message"`
	StakeholderChannel string         `json:"stakeholder_channel"`
	AuthorRole         string         `json:"author_role"`
	Metadata           map[string]any `json:"metadata"`
}

// IncidentResponse aggregates incident data with timeline/checklist.
type IncidentResponse struct {
	Incident        *opmodels.Incident                `json:"incident"`
	Timeline        []*opmodels.IncidentTimelineEntry `json:"timeline"`
	Checklist       []*opmodels.IncidentChecklistItem `json:"checklist"`
	ChecklistStatus ChecklistSummary                  `json:"checklist_status"`
}

// ChecklistSummary summarizes readiness gates across operations domains.
type ChecklistSummary struct {
	SupportReady  bool     `json:"support_ready"`
	IncidentReady bool     `json:"incident_ready"`
	SLAReady      bool     `json:"sla_ready"`
	BlockingItems []string `json:"blocking_items"`
}

// CreateIncident registers a new incident.
func (s *IncidentService) CreateIncident(ctx context.Context, req CreateIncidentRequest) (*opmodels.Incident, error) {
	if s.repo == nil {
		return nil, errors.New("incident repository not configured")
	}
	if _, err := s.ensureIncidentReadiness(ctx); err != nil {
		return nil, err
	}
	severity := strings.ToLower(strings.TrimSpace(req.Severity))
	if _, ok := validSeverities[severity]; !ok {
		return nil, fmt.Errorf("invalid severity %q", req.Severity)
	}
	detection := strings.TrimSpace(req.DetectionSource)
	if detection == "" {
		return nil, errors.New("detection_source is required")
	}
	summary := strings.TrimSpace(req.Summary)
	if summary == "" {
		return nil, errors.New("summary is required")
	}
	now := time.Now().UTC()
	incident := &opmodels.Incident{
		PluginID:        app.PluginID,
		Severity:        severity,
		Status:          IncidentStatusDetected,
		DetectionSource: detection,
		Summary:         summary,
		Impact:          datatypes.JSONMap(req.Impact),
		Mitigation:      req.Mitigation,
		Confidentiality: strings.TrimSpace(req.Confidentiality),
	}
	if req.TenantUuid != nil && strings.TrimSpace(*req.TenantUuid) != "" {
		clean := strings.TrimSpace(*req.TenantUuid)
		incident.TenantUuid = &clean
	}
	incident.Labels = make(datatypes.JSONMap)
	for k, v := range req.Labels {
		incident.Labels[k] = v
	}
	if req.NextUpdateAt != nil {
		incident.NextUpdateAt = req.NextUpdateAt
	} else if dur, ok := severityUpdateCadence[severity]; ok {
		next := now.Add(dur)
		incident.NextUpdateAt = &next
	}
	saved, err := s.repo.CreateIncident(ctx, incident)
	if err != nil {
		return nil, err
	}
	if err := s.setIncidentReadinessStatus(ctx, "sev_matrix_defined", true, fmt.Sprintf("Incident declared with severity %s", severity)); err != nil {
		return nil, err
	}
	s.metrics.RecordIncidentEvent(saved.Severity, "created")
	_ = s.dispatcher.DispatchIncidentEvent(ctx, "operations.incident.created", saved, map[string]any{"severity": saved.Severity})
	return saved, nil
}

// UpdateIncident applies mutable changes to an incident.
func (s *IncidentService) UpdateIncident(ctx context.Context, incidentID string, req UpdateIncidentRequest) (*opmodels.Incident, error) {
	if s.repo == nil {
		return nil, errors.New("incident repository not configured")
	}
	if _, err := s.ensureIncidentReadiness(ctx); err != nil {
		return nil, err
	}
	incident, err := s.repo.GetIncident(ctx, app.PluginID, incidentID)
	if err != nil {
		return nil, err
	}
	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		if !contains(validStatuses, status) {
			return nil, fmt.Errorf("invalid status %q", *req.Status)
		}
		incident.Status = status
		now := time.Now().UTC()
		switch status {
		case IncidentStatusAcknowledged:
			incident.AcknowledgedAt = &now
		case IncidentStatusMitigated:
			incident.MitigatedAt = &now
		case IncidentStatusMonitoring:
			// no timestamp update
		case IncidentStatusResolved:
			incident.ResolvedAt = &now
		case IncidentStatusClosed:
			incident.ClosedAt = &now
		}
	}
	if req.Mitigation != nil {
		incident.Mitigation = *req.Mitigation
	}
	if req.RootCause != nil {
		incident.RootCause = *req.RootCause
	}
	if req.NextUpdateAt != nil {
		incident.NextUpdateAt = req.NextUpdateAt
	}
	if req.Labels != nil {
		if incident.Labels == nil {
			incident.Labels = datatypes.JSONMap{}
		}
		for k, v := range req.Labels {
			incident.Labels[k] = v
		}
	}
	if req.Confidentiality != nil {
		incident.Confidentiality = strings.TrimSpace(*req.Confidentiality)
	}
	if err := s.repo.UpdateIncident(ctx, incident); err != nil {
		return nil, err
	}
	if req.Status != nil {
		s.metrics.RecordIncidentEvent(incident.Severity, incident.Status)
		_ = s.dispatcher.DispatchIncidentEvent(ctx, "operations.incident.status_changed", incident, map[string]any{"status": incident.Status})
	}
	return incident, nil
}

// AppendTimeline appends a timeline entry to an incident.
func (s *IncidentService) AppendTimeline(ctx context.Context, incidentID string, req TimelineEntryRequest) (*opmodels.IncidentTimelineEntry, error) {
	if s.repo == nil {
		return nil, errors.New("incident repository not configured")
	}
	if _, err := s.ensureIncidentReadiness(ctx); err != nil {
		return nil, err
	}
	entryType := strings.TrimSpace(req.EntryType)
	if entryType == "" {
		return nil, errors.New("entry_type is required")
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, errors.New("message is required")
	}
	incident, err := s.repo.GetIncident(ctx, app.PluginID, incidentID)
	if err != nil {
		return nil, err
	}
	event := &opmodels.IncidentTimelineEntry{
		IncidentID:         incidentID,
		EntryType:          entryType,
		Message:            message,
		StakeholderChannel: strings.TrimSpace(req.StakeholderChannel),
		AuthorRole:         strings.TrimSpace(req.AuthorRole),
		Metadata:           datatypes.JSONMap(req.Metadata),
	}
	recorded, err := s.repo.AppendTimelineEntry(ctx, event)
	if err != nil {
		return nil, err
	}
	if dur, ok := severityUpdateCadence[incident.Severity]; ok {
		next := time.Now().UTC().Add(dur)
		incident.NextUpdateAt = &next
		if err := s.repo.UpdateIncident(ctx, incident); err != nil {
			return nil, err
		}
	}
	if channel := strings.TrimSpace(req.StakeholderChannel); channel != "" {
		if err := s.setIncidentReadinessStatus(ctx, "communication_channels_tested", true, fmt.Sprintf("Timeline broadcast via %s", channel)); err != nil {
			return nil, err
		}
	}
	s.syncStatusPage(ctx, incident, recorded)
	s.metrics.RecordIncidentEvent(event.EntryType, "timeline")
	return recorded, nil
}

// GetIncident fetches incident with related timeline/checklist.
func (s *IncidentService) GetIncident(ctx context.Context, incidentID string) (*IncidentResponse, error) {
	incident, err := s.repo.GetIncident(ctx, app.PluginID, incidentID)
	if err != nil {
		return nil, err
	}
	timeline, err := s.repo.ListTimeline(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	checklist, err := s.repo.ListChecklistItems(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	summary, err := s.buildChecklistSummary(ctx)
	if err != nil {
		return nil, err
	}
	return &IncidentResponse{
		Incident:        incident,
		Timeline:        timeline,
		Checklist:       checklist,
		ChecklistStatus: summary,
	}, nil
}

// ListIncidents lists incidents by filters.
func (s *IncidentService) ListIncidents(ctx context.Context, pluginID string, filter IncidentFilter) ([]*opmodels.Incident, error) {
	return s.repo.ListIncidents(ctx, pluginID, filter.Severities, filter.Statuses, filter.Labels, filter.From, filter.To)
}

// IncidentFilter specifies list filters.
type IncidentFilter struct {
	Severities []string
	Statuses   []string
	Labels     []string
	From       *time.Time
	To         *time.Time
}

// UpdateChecklistItem upserts incident checklist item.
func (s *IncidentService) UpdateChecklistItem(ctx context.Context, incidentID, key, description, status string, completed bool) (*opmodels.IncidentChecklistItem, error) {
	if s.repo == nil {
		return nil, errors.New("incident repository not configured")
	}
	item := &opmodels.IncidentChecklistItem{
		IncidentID:  incidentID,
		ItemKey:     key,
		Description: description,
		Status:      status,
	}
	if completed {
		completedAt := time.Now().UTC()
		item.CompletedAt = &completedAt
	}
	saved, err := s.repo.UpsertChecklistItem(ctx, item)
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *IncidentService) ensureIncidentReadiness(ctx context.Context) (map[string]*opmodels.ReadinessChecklistItem, error) {
	pluginID := app.PluginID
	existing, err := s.repo.ListReadinessByType(ctx, pluginID, string(runtimeops.ChecklistIncidentReady))
	if err != nil {
		return nil, err
	}
	byKey := map[string]*opmodels.ReadinessChecklistItem{}
	for _, item := range existing {
		clone := *item
		byKey[item.ItemKey] = &clone
	}
	result := map[string]*opmodels.ReadinessChecklistItem{}
	for _, tmpl := range s.readiness[runtimeops.ChecklistIncidentReady] {
		record, ok := byKey[tmpl.Key]
		if !ok {
			record = &opmodels.ReadinessChecklistItem{
				PluginID:    pluginID,
				Type:        string(runtimeops.ChecklistIncidentReady),
				ItemKey:     tmpl.Key,
				Description: tmpl.Description,
				Status:      runtimeops.ChecklistStatusPending,
				OwnerRole:   tmpl.OwnerRole,
			}
		} else {
			record.Description = tmpl.Description
			record.OwnerRole = tmpl.OwnerRole
		}
		saved, err := s.repo.UpsertReadinessItem(ctx, record)
		if err != nil {
			return nil, err
		}
		result[tmpl.Key] = saved
	}
	return result, nil
}

func (s *IncidentService) setIncidentReadinessStatus(ctx context.Context, key string, completed bool, notes string) error {
	items, err := s.ensureIncidentReadiness(ctx)
	if err != nil {
		return err
	}
	item, ok := items[key]
	if !ok {
		return fmt.Errorf("unknown incident readiness item %s", key)
	}
	switch {
	case completed && strings.EqualFold(item.Status, runtimeops.ChecklistStatusCompleted):
		return nil
	case completed:
		item.Status = runtimeops.ChecklistStatusCompleted
		completedAt := time.Now().UTC()
		item.CompletedAt = &completedAt
	default:
		item.Status = runtimeops.ChecklistStatusPending
		item.CompletedAt = nil
	}
	if notes != "" {
		item.Notes = notes
	}
	_, err = s.repo.UpsertReadinessItem(ctx, item)
	return err
}

func (s *IncidentService) buildChecklistSummary(ctx context.Context) (ChecklistSummary, error) {
	summary := ChecklistSummary{}
	blocking := []string{}
	for _, checklistType := range runtimeops.ListReadinessTypes() {
		blueprint := s.readiness[checklistType]
		existing, err := s.repo.ListReadinessByType(ctx, app.PluginID, string(checklistType))
		if err != nil {
			return summary, err
		}
		byKey := map[string]*opmodels.ReadinessChecklistItem{}
		for _, item := range existing {
			clone := *item
			byKey[item.ItemKey] = &clone
		}
		allBlockingCompleted := true
		for _, tmpl := range blueprint {
			record, ok := byKey[tmpl.Key]
			if !ok {
				record = &opmodels.ReadinessChecklistItem{
					PluginID:    app.PluginID,
					Type:        string(checklistType),
					ItemKey:     tmpl.Key,
					Description: tmpl.Description,
					Status:      runtimeops.ChecklistStatusPending,
					OwnerRole:   tmpl.OwnerRole,
				}
			} else {
				record.Description = tmpl.Description
				record.OwnerRole = tmpl.OwnerRole
			}
			saved, err := s.repo.UpsertReadinessItem(ctx, record)
			if err != nil {
				return summary, err
			}
			if tmpl.Blocking && !strings.EqualFold(saved.Status, runtimeops.ChecklistStatusCompleted) {
				allBlockingCompleted = false
				blocking = append(blocking, tmpl.Key)
			}
		}
		switch checklistType {
		case runtimeops.ChecklistSupportReady:
			summary.SupportReady = allBlockingCompleted
		case runtimeops.ChecklistIncidentReady:
			summary.IncidentReady = allBlockingCompleted
		case runtimeops.ChecklistSLAReady:
			summary.SLAReady = allBlockingCompleted
		}
	}
	summary.BlockingItems = uniqueStrings(blocking)
	return summary, nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func (s *IncidentService) syncStatusPage(ctx context.Context, incident *opmodels.Incident, entry *opmodels.IncidentTimelineEntry) {
	if s.cfg == nil || s.cfg.Operations == nil || incident == nil || entry == nil {
		return
	}
	if !strings.EqualFold(entry.StakeholderChannel, "status_page") {
		return
	}
	url := strings.TrimSpace(s.cfg.Operations.Incident.StatusPageURL)
	if url == "" {
		return
	}
	payload := map[string]any{
		"incidentId":   incident.ID,
		"severity":     incident.Severity,
		"status":       incident.Status,
		"message":      entry.Message,
		"postedAt":     entry.PostedAt,
		"nextUpdateAt": incident.NextUpdateAt,
		"labels":       incident.Labels,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
