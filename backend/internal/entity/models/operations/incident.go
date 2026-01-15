package operations

import (
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// Incident represents an operations incident lifecycle record.
type Incident struct {
	ID              string            `gorm:"primaryKey;type:uuid" json:"id"`
	PluginID        string            `gorm:"column:plugin_id;index:idx_operations_incidents_scope" json:"plugin_id"`
	TenantUuid      *string           `gorm:"column:tenant_uuid;index:idx_operations_incidents_scope" json:"tenant_uuid,omitempty"`
	Severity        string            `gorm:"column:severity" json:"severity"`
	Status          string            `gorm:"column:status" json:"status"`
	DetectionSource string            `gorm:"column:detection_source" json:"detection_source"`
	Summary         string            `gorm:"column:summary" json:"summary"`
	Impact          datatypes.JSONMap `gorm:"column:impact" json:"impact"`
	Mitigation      string            `gorm:"column:mitigation" json:"mitigation"`
	RootCause       string            `gorm:"column:root_cause" json:"root_cause"`
	NextUpdateAt    *time.Time        `gorm:"column:next_update_at" json:"next_update_at,omitempty"`
	Labels          datatypes.JSONMap `gorm:"column:labels" json:"labels"`
	Confidentiality string            `gorm:"column:confidentiality" json:"confidentiality"`
	DetectedAt      time.Time         `gorm:"column:detected_at" json:"detected_at"`
	AcknowledgedAt  *time.Time        `gorm:"column:acknowledged_at" json:"acknowledged_at,omitempty"`
	MitigatedAt     *time.Time        `gorm:"column:mitigated_at" json:"mitigated_at,omitempty"`
	ResolvedAt      *time.Time        `gorm:"column:resolved_at" json:"resolved_at,omitempty"`
	ClosedAt        *time.Time        `gorm:"column:closed_at" json:"closed_at,omitempty"`
	CreatedAt       time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns schema-qualified table name.
func (Incident) TableName() string {
	return basemodels.S(basemodels.TableOperationsIncidents)
}

// IncidentTimelineEntry represents a stakeholder communication update for an incident.
type IncidentTimelineEntry struct {
	ID                 string            `gorm:"primaryKey;type:uuid" json:"id"`
	IncidentID         string            `gorm:"column:incident_id;index" json:"incident_id"`
	EntryType          string            `gorm:"column:entry_type" json:"entry_type"`
	Message            string            `gorm:"column:message" json:"message"`
	StakeholderChannel string            `gorm:"column:stakeholder_channel" json:"stakeholder_channel"`
	AuthorRole         string            `gorm:"column:author_role" json:"author_role"`
	PostedAt           time.Time         `gorm:"column:posted_at" json:"posted_at"`
	Metadata           datatypes.JSONMap `gorm:"column:metadata" json:"metadata"`
	CreatedAt          time.Time         `gorm:"column:created_at" json:"created_at"`
}

// TableName returns schema-qualified table name.
func (IncidentTimelineEntry) TableName() string {
	return basemodels.S(basemodels.TableOperationsIncidentUpdates)
}

// IncidentChecklistItem tracks incident-specific checklist tasks.
type IncidentChecklistItem struct {
	ID          string     `gorm:"primaryKey;type:uuid" json:"id"`
	IncidentID  string     `gorm:"column:incident_id;index" json:"incident_id"`
	ItemKey     string     `gorm:"column:item_key" json:"item_key"`
	Description string     `gorm:"column:description" json:"description"`
	Status      string     `gorm:"column:status" json:"status"`
	CompletedAt *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns schema-qualified table name.
func (IncidentChecklistItem) TableName() string {
	return basemodels.S(basemodels.TableOperationsIncidentChecklist)
}
