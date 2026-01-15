package admin_console

import (
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// AuditEvent captures privileged admin actions taken through the Dev Console.
type AuditEvent struct {
	ID             string         `gorm:"column:id;primaryKey" json:"id"`
	PluginID       string         `gorm:"column:plugin_id;type:text;not null" json:"plugin_id"`
	TenantUuid     *string        `gorm:"column:tenant_uuid;type:uuid" json:"tenant_uuid,omitempty"`
	ActorID        string         `gorm:"column:actor_id;type:text;not null" json:"actor_id"`
	ActorName      *string        `gorm:"column:actor_name;type:text" json:"actor_name,omitempty"`
	ActorEmail     *string        `gorm:"column:actor_email;type:text" json:"actor_email,omitempty"`
	PermissionCode string         `gorm:"column:permission_code;type:text;not null" json:"permission_code"`
	Action         string         `gorm:"column:action;type:text;not null" json:"action"`
	ResourceType   string         `gorm:"column:resource_type;type:text;not null" json:"resource_type"`
	ResourceRef    *string        `gorm:"column:resource_ref;type:text" json:"resource_ref,omitempty"`
	Summary        *string        `gorm:"column:summary;type:text" json:"summary,omitempty"`
	Diff           datatypes.JSON `gorm:"column:diff;type:jsonb" json:"diff,omitempty"`
	OccurredAt     time.Time      `gorm:"column:occurred_at;not null;default:now()" json:"occurred_at"`
	CreatedAt      time.Time      `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	ConfigChanges  []ConfigChange `gorm:"foreignKey:AuditEventID" json:"config_changes,omitempty"`
	RelatedJobRuns []JobRun       `gorm:"foreignKey:AuditEventID" json:"job_runs,omitempty"`
}

// TableName implements gorm tablename interface.
func (AuditEvent) TableName() string {
	return basemodels.S(basemodels.TableAdminConsoleAuditEvents)
}

// ConfigChange stores configuration deltas applied via the console.
type ConfigChange struct {
	ID                string         `gorm:"column:id;primaryKey" json:"id"`
	PluginID          string         `gorm:"column:plugin_id;type:text;not null" json:"plugin_id"`
	TenantUuid        *string        `gorm:"column:tenant_uuid;type:uuid" json:"tenant_uuid,omitempty"`
	SectionKey        string         `gorm:"column:section_key;type:text;not null" json:"section_key"`
	ChangeType        string         `gorm:"column:change_type;type:text;not null" json:"change_type"`
	PreviousSnapshot  datatypes.JSON `gorm:"column:previous_snapshot;type:jsonb" json:"previous_snapshot,omitempty"`
	NextSnapshot      datatypes.JSON `gorm:"column:next_snapshot;type:jsonb" json:"next_snapshot,omitempty"`
	ValidationSummary datatypes.JSON `gorm:"column:validation_summary;type:jsonb" json:"validation_summary,omitempty"`
	AuditEventID      string         `gorm:"column:audit_event_id;type:uuid;not null" json:"audit_event_id"`
	AppliedAt         time.Time      `gorm:"column:applied_at;not null;default:now()" json:"applied_at"`
	AuditEvent        *AuditEvent    `gorm:"foreignKey:AuditEventID" json:"audit_event,omitempty"`
}

// TableName implements gorm tablename interface.
func (ConfigChange) TableName() string {
	return basemodels.S(basemodels.TableAdminConsoleConfigChanges)
}

// JobRun tracks safe operation executions and troubleshooting jobs.
type JobRun struct {
	ID             string         `gorm:"column:id;primaryKey" json:"id"`
	PluginID       string         `gorm:"column:plugin_id;type:text;not null" json:"plugin_id"`
	TenantUuid     *string        `gorm:"column:tenant_uuid;type:uuid" json:"tenant_uuid,omitempty"`
	Environment    string         `gorm:"column:environment;type:text" json:"environment"`
	JobType        string         `gorm:"column:job_type;type:text;not null" json:"job_type"`
	TriggerSource  string         `gorm:"column:trigger_source;type:text;not null" json:"trigger_source"`
	Status         string         `gorm:"column:status;type:text;not null" json:"status"`
	Action         string         `gorm:"column:action;type:text" json:"action,omitempty"`
	ScopeType      string         `gorm:"column:scope_type;type:text" json:"scope_type,omitempty"`
	ScopeRef       *string        `gorm:"column:scope_ref;type:text" json:"scope_ref,omitempty"`
	TargetID       *string        `gorm:"column:target_id;type:text" json:"target_id,omitempty"`
	Reason         *string        `gorm:"column:reason;type:text" json:"reason,omitempty"`
	DryRun         bool           `gorm:"column:dry_run;type:boolean" json:"dry_run"`
	Metadata       datatypes.JSON `gorm:"column:metadata;type:jsonb" json:"metadata,omitempty"`
	StartedAt      *time.Time     `gorm:"column:started_at" json:"started_at,omitempty"`
	FinishedAt     *time.Time     `gorm:"column:finished_at" json:"finished_at,omitempty"`
	DurationMillis int64          `gorm:"column:duration_ms;type:bigint;->" json:"duration_ms"`
	Message        *string        `gorm:"column:message;type:text" json:"message,omitempty"`
	RetryOf        *string        `gorm:"column:retry_of;type:uuid" json:"retry_of,omitempty"`
	AuditEventID   *string        `gorm:"column:audit_event_id;type:uuid" json:"audit_event_id,omitempty"`
	CreatedBy      string         `gorm:"column:created_by;type:text;not null" json:"created_by"`
	CreatedAt      time.Time      `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
	RetryTarget    *JobRun        `gorm:"foreignKey:RetryOf" json:"retry_target,omitempty"`
	AuditEvent     *AuditEvent    `gorm:"foreignKey:AuditEventID" json:"audit_event,omitempty"`
}

// TableName implements gorm tablename interface.
func (JobRun) TableName() string {
	return basemodels.S(basemodels.TableAdminConsoleJobRuns)
}
