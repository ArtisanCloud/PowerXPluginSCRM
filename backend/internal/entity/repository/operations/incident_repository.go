package operations

import (
	"context"
	"errors"
	"strings"
	"time"

	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IncidentRepository encapsulates persistence for incidents and related artefacts.
type IncidentRepository struct {
	db            *gorm.DB
	incidentsRepo *repository.BaseRepository[opmodels.Incident]
	timelineRepo  *repository.BaseRepository[opmodels.IncidentTimelineEntry]
	checkRepo     *repository.BaseRepository[opmodels.IncidentChecklistItem]
	readinessRepo *repository.BaseRepository[opmodels.ReadinessChecklistItem]
}

// NewIncidentRepository constructs the repository with shared DB handles.
func NewIncidentRepository(db *gorm.DB) *IncidentRepository {
	return &IncidentRepository{
		db:            db,
		incidentsRepo: repository.NewBaseRepository[opmodels.Incident](db),
		timelineRepo:  repository.NewBaseRepository[opmodels.IncidentTimelineEntry](db),
		checkRepo:     repository.NewBaseRepository[opmodels.IncidentChecklistItem](db),
		readinessRepo: repository.NewBaseRepository[opmodels.ReadinessChecklistItem](db),
	}
}

// CreateIncident inserts a new incident record.
func (r *IncidentRepository) CreateIncident(ctx context.Context, incident *opmodels.Incident) (*opmodels.Incident, error) {
	if incident == nil {
		return nil, errors.New("incident is required")
	}
	if strings.TrimSpace(incident.PluginID) == "" {
		return nil, errors.New("incident plugin_id is required")
	}
	now := time.Now().UTC()
	if incident.ID == "" {
		incident.ID = uuid.NewString()
	}
	if incident.DetectedAt.IsZero() {
		incident.DetectedAt = now
	}
	if strings.TrimSpace(incident.Status) == "" {
		incident.Status = "detected"
	}
	incident.CreatedAt = now
	incident.UpdatedAt = now
	if _, err := r.incidentsRepo.Create(ctx, incident); err != nil {
		return nil, err
	}
	return incident, nil
}

// UpdateIncident persists mutations on an incident.
func (r *IncidentRepository) UpdateIncident(ctx context.Context, incident *opmodels.Incident) error {
	if incident == nil {
		return errors.New("incident is required")
	}
	if strings.TrimSpace(incident.ID) == "" {
		return errors.New("incident id is required")
	}
	incident.UpdatedAt = time.Now().UTC()
	_, err := r.incidentsRepo.Update(ctx, incident)
	return err
}

// GetIncident fetches an incident bounded to the plugin scope.
func (r *IncidentRepository) GetIncident(ctx context.Context, pluginID, id string) (*opmodels.Incident, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("incident id is required")
	}
	var record opmodels.Incident
	err := r.db.WithContext(ctx).
		Where("id = ? AND plugin_id = ?", id, pluginID).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListIncidents lists incidents within the plugin scope and optional filters.
func (r *IncidentRepository) ListIncidents(
	ctx context.Context,
	pluginID string,
	severities []string,
	statuses []string,
	labels []string,
	from *time.Time,
	to *time.Time,
) ([]*opmodels.Incident, error) {
	query := r.db.WithContext(ctx).Where("plugin_id = ?", pluginID)
	if len(severities) > 0 {
		query = query.Where("severity IN ?", severities)
	}
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	if len(labels) > 0 {
		for _, label := range labels {
			label = strings.TrimSpace(label)
			if label == "" {
				continue
			}
			query = query.Where("labels ->>? = 'true'", label)
		}
	}
	if from != nil {
		query = query.Where("detected_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("detected_at <= ?", *to)
	}
	var incidents []*opmodels.Incident
	if err := query.Order("detected_at DESC").Find(&incidents).Error; err != nil {
		return nil, err
	}
	return incidents, nil
}

// AppendTimelineEntry records a communication update for an incident.
func (r *IncidentRepository) AppendTimelineEntry(ctx context.Context, entry *opmodels.IncidentTimelineEntry) (*opmodels.IncidentTimelineEntry, error) {
	if entry == nil {
		return nil, errors.New("timeline entry is required")
	}
	if strings.TrimSpace(entry.IncidentID) == "" {
		return nil, errors.New("incident_id is required")
	}
	now := time.Now().UTC()
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.PostedAt.IsZero() {
		entry.PostedAt = now
	}
	entry.CreatedAt = now
	if _, err := r.timelineRepo.Create(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// ListTimeline returns timeline entries ordered chronologically.
func (r *IncidentRepository) ListTimeline(ctx context.Context, incidentID string) ([]*opmodels.IncidentTimelineEntry, error) {
	if strings.TrimSpace(incidentID) == "" {
		return nil, errors.New("incident_id is required")
	}
	var entries []*opmodels.IncidentTimelineEntry
	if err := r.db.WithContext(ctx).
		Where("incident_id = ?", incidentID).
		Order("posted_at ASC").
		Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// UpsertChecklistItem saves or updates an incident readiness checklist item.
func (r *IncidentRepository) UpsertChecklistItem(ctx context.Context, item *opmodels.IncidentChecklistItem) (*opmodels.IncidentChecklistItem, error) {
	if item == nil {
		return nil, errors.New("checklist item is required")
	}
	if strings.TrimSpace(item.IncidentID) == "" {
		return nil, errors.New("incident_id is required")
	}
	now := time.Now().UTC()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

// ListChecklistItems returns checklist items ordered by key.
func (r *IncidentRepository) ListChecklistItems(ctx context.Context, incidentID string) ([]*opmodels.IncidentChecklistItem, error) {
	if strings.TrimSpace(incidentID) == "" {
		return nil, errors.New("incident_id is required")
	}
	var items []*opmodels.IncidentChecklistItem
	if err := r.db.WithContext(ctx).
		Where("incident_id = ?", incidentID).
		Order("item_key").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// UpsertReadinessItem upserts an incident readiness checklist entry.
func (r *IncidentRepository) UpsertReadinessItem(ctx context.Context, item *opmodels.ReadinessChecklistItem) (*opmodels.ReadinessChecklistItem, error) {
	if item == nil {
		return nil, errors.New("readiness item is required")
	}
	if strings.TrimSpace(item.PluginID) == "" {
		return nil, errors.New("plugin_id is required")
	}
	if strings.TrimSpace(item.Type) == "" {
		return nil, errors.New("readiness type is required")
	}
	if strings.TrimSpace(item.ItemKey) == "" {
		return nil, errors.New("readiness item key is required")
	}
	now := time.Now().UTC()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

// ListReadinessByType returns readiness checklist items for incident readiness.
func (r *IncidentRepository) ListReadinessByType(ctx context.Context, pluginID, checklistType string) ([]*opmodels.ReadinessChecklistItem, error) {
	var items []*opmodels.ReadinessChecklistItem
	if err := r.db.WithContext(ctx).
		Where("plugin_id = ? AND type = ?", pluginID, checklistType).
		Order("item_key").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
