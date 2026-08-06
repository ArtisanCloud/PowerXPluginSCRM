package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	LeadActivityTypeExternal = "external_activity"

	LeadBridgeFieldDisplayName = "display_name"
	LeadBridgeFieldPhone       = "phone"
	LeadBridgeFieldEmail       = "email"

	LeadHandoffStatusPending  = "pending"
	LeadHandoffStatusAccepted = "accepted"
	LeadHandoffStatusFailed   = "failed"

	LeadBridgeConflictStatusOpen     = "open"
	LeadBridgeConflictStatusResolved = "resolved"
)

// LeadBridgeMapping stores the UUID mapping between SCRM private-domain leads and external lead objects.
type LeadBridgeMapping struct {
	MappingUUID           string    `gorm:"column:mapping_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"mapping_uuid"`
	TenantUUID            string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_bridge_mapping_tenant;uniqueIndex:ux_lead_bridge_mapping_external" json:"tenant_uuid"`
	LeadUUID              string    `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_bridge_mapping_lead" json:"lead_uuid"`
	ExternalPluginID      string    `gorm:"column:external_plugin_id;type:varchar(128);not null;index:idx_lead_bridge_mapping_external;uniqueIndex:ux_lead_bridge_mapping_external" json:"external_plugin_id"`
	ExternalLeadUUID      string    `gorm:"column:external_lead_uuid;type:uuid;not null;index:idx_lead_bridge_mapping_external;uniqueIndex:ux_lead_bridge_mapping_external" json:"external_lead_uuid"`
	ExternalLeadType      string    `gorm:"column:external_lead_type;type:varchar(64);not null;default:'sales_lead'" json:"external_lead_type"`
	RelationStatus        string    `gorm:"column:relation_status;type:varchar(32);not null;default:'active'" json:"relation_status"`
	LastExternalEventUUID string    `gorm:"column:last_external_event_uuid;type:uuid" json:"last_external_event_uuid,omitempty"`
	CreatedAt             time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadBridgeMapping) TableName() string {
	return models.S(models.TableLeadBridgeMappings)
}

// LeadHandoff tracks the SCRM-to-CRM handoff request lifecycle.
type LeadHandoff struct {
	HandoffUUID        string            `gorm:"column:handoff_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"handoff_uuid"`
	TenantUUID         string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_handoffs_tenant" json:"tenant_uuid"`
	LeadUUID           string            `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_handoffs_lead" json:"lead_uuid"`
	TargetPluginID     string            `gorm:"column:target_plugin_id;type:varchar(128);not null" json:"target_plugin_id"`
	TargetCapabilityID string            `gorm:"column:target_capability_id;type:varchar(160);not null" json:"target_capability_id"`
	Status             string            `gorm:"column:status;type:varchar(32);not null;default:'pending';index:idx_lead_handoffs_status" json:"status"`
	RequestPayload     datatypes.JSONMap `gorm:"column:request_payload;type:jsonb;default:'{}'::jsonb" json:"request_payload,omitempty"`
	ResponsePayload    datatypes.JSONMap `gorm:"column:response_payload;type:jsonb;default:'{}'::jsonb" json:"response_payload,omitempty"`
	LastError          string            `gorm:"column:last_error;type:text" json:"last_error,omitempty"`
	CreatedAt          time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadHandoff) TableName() string {
	return models.S(models.TableLeadHandoffs)
}

// HandoffAttempt records each explicit handoff attempt and its result.
type HandoffAttempt struct {
	AttemptUUID        string            `gorm:"column:attempt_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"attempt_uuid"`
	TenantUUID         string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_handoff_attempts_tenant" json:"tenant_uuid"`
	HandoffUUID        string            `gorm:"column:handoff_uuid;type:uuid;not null;index:idx_lead_handoff_attempts_handoff" json:"handoff_uuid"`
	LeadUUID           string            `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_handoff_attempts_lead" json:"lead_uuid"`
	CapabilityID       string            `gorm:"column:capability_id;type:varchar(160);not null" json:"capability_id"`
	Status             string            `gorm:"column:status;type:varchar(32);not null" json:"status"`
	RequestPayloadHash string            `gorm:"column:request_payload_hash;type:varchar(128)" json:"request_payload_hash,omitempty"`
	ResponsePayload    datatypes.JSONMap `gorm:"column:response_payload;type:jsonb;default:'{}'::jsonb" json:"response_payload,omitempty"`
	ErrorMessage       string            `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	CreatedAt          time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

func (HandoffAttempt) TableName() string {
	return models.S(models.TableLeadHandoffAttempts)
}

// LeadIdentitySyncState stores per-field sync metadata for the only bidirectional fields.
type LeadIdentitySyncState struct {
	SyncStateUUID       string    `gorm:"column:sync_state_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"sync_state_uuid"`
	TenantUUID          string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_identity_sync_tenant;uniqueIndex:ux_lead_identity_sync_field" json:"tenant_uuid"`
	LeadUUID            string    `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_identity_sync_lead;uniqueIndex:ux_lead_identity_sync_field" json:"lead_uuid"`
	ExternalPluginID    string    `gorm:"column:external_plugin_id;type:varchar(128);not null;index:idx_lead_identity_sync_external;uniqueIndex:ux_lead_identity_sync_field" json:"external_plugin_id"`
	ExternalLeadUUID    string    `gorm:"column:external_lead_uuid;type:uuid;not null;index:idx_lead_identity_sync_external;uniqueIndex:ux_lead_identity_sync_field" json:"external_lead_uuid"`
	FieldName           string    `gorm:"column:field_name;type:varchar(64);not null;uniqueIndex:ux_lead_identity_sync_field" json:"field_name"`
	LastSyncedValueHash string    `gorm:"column:last_synced_value_hash;type:varchar(128)" json:"last_synced_value_hash,omitempty"`
	LastSourcePluginID  string    `gorm:"column:last_source_plugin_id;type:varchar(128)" json:"last_source_plugin_id,omitempty"`
	LastSyncedAt        time.Time `gorm:"column:last_synced_at;type:timestamptz" json:"last_synced_at,omitempty"`
	CreatedAt           time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadIdentitySyncState) TableName() string {
	return models.S(models.TableLeadIdentitySyncStates)
}

// LeadBridgeConflict records identity sync conflicts that require explicit resolution.
type LeadBridgeConflict struct {
	ConflictUUID     string            `gorm:"column:conflict_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"conflict_uuid"`
	TenantUUID       string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_bridge_conflicts_tenant" json:"tenant_uuid"`
	LeadUUID         string            `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_bridge_conflicts_lead" json:"lead_uuid"`
	ExternalPluginID string            `gorm:"column:external_plugin_id;type:varchar(128);not null" json:"external_plugin_id"`
	ExternalLeadUUID string            `gorm:"column:external_lead_uuid;type:uuid;not null" json:"external_lead_uuid"`
	FieldName        string            `gorm:"column:field_name;type:varchar(64);not null" json:"field_name"`
	LocalValue       string            `gorm:"column:local_value;type:text" json:"local_value,omitempty"`
	ExternalValue    string            `gorm:"column:external_value;type:text" json:"external_value,omitempty"`
	Status           string            `gorm:"column:status;type:varchar(32);not null;default:'open';index:idx_lead_bridge_conflicts_status" json:"status"`
	Resolution       datatypes.JSONMap `gorm:"column:resolution;type:jsonb;default:'{}'::jsonb" json:"resolution,omitempty"`
	ResolvedAt       *time.Time        `gorm:"column:resolved_at;type:timestamptz" json:"resolved_at,omitempty"`
	CreatedAt        time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadBridgeConflict) TableName() string {
	return models.S(models.TableLeadBridgeConflicts)
}
