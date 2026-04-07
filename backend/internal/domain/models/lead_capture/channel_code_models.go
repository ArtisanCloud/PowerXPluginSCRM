package lead_capture

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
	"gorm.io/datatypes"
)

const (
	ChannelCodeStatusDraft    = "draft"
	ChannelCodeStatusActive   = "active"
	ChannelCodeStatusDisabled = "disabled"

	WelcomeSyncStatusPending        = "pending"
	WelcomeSyncStatusSyncing        = "syncing"
	WelcomeSyncStatusSuccess        = "success"
	WelcomeSyncStatusFailed         = "failed"
	WelcomeSyncStatusManualRequired = "manual_required"

	WelcomeSyncTriggerManual    = "manual"
	WelcomeSyncTriggerAutoRetry = "auto_retry"

	WelcomeSyncResultSuccess = "success"
	WelcomeSyncResultFailed  = "failed"

	AttributionTypeFirstTouch  = "first_touch"
	AttributionTypeFollowTouch = "follow_touch"
)

// ChannelCode defines tenant-scoped channel entry points.
type ChannelCode struct {
	CodeUUID           string    `gorm:"column:code_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"code_uuid"`
	TenantUUID         string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_channel_codes_tenant;uniqueIndex:uq_lead_capture_channel_codes_tenant_channel_code,priority:1" json:"tenant_uuid"`
	Channel            string    `gorm:"column:channel;type:varchar(64);not null;index:idx_lead_capture_channel_codes_channel;uniqueIndex:uq_lead_capture_channel_codes_tenant_channel_code,priority:2" json:"channel"`
	AppType            string    `gorm:"column:app_type;type:varchar(64);not null;index:idx_lead_capture_channel_codes_app_type" json:"app_type"`
	ChannelAccountUUID string    `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_lead_capture_channel_codes_account" json:"channel_account_uuid"`
	CodeKey            string    `gorm:"column:code_key;type:varchar(128);not null;uniqueIndex:uq_lead_capture_channel_codes_tenant_channel_code,priority:3" json:"code_key"`
	DisplayName        string    `gorm:"column:display_name;type:varchar(128);not null" json:"display_name"`
	TargetType         string    `gorm:"column:target_type;type:varchar(32);not null" json:"target_type"`
	TargetID           string    `gorm:"column:target_id;type:varchar(128);not null" json:"target_id"`
	Status             string    `gorm:"column:status;type:varchar(32);not null;default:'draft';index:idx_lead_capture_channel_codes_status" json:"status"`
	CreatedBy          string    `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy          string    `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt          time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (ChannelCode) TableName() string {
	return domainmodels.S(domainmodels.TableLeadCaptureChannelCodes)
}

// CodeWelcomeConfig stores message payload and publishing state per channel code.
type CodeWelcomeConfig struct {
	ConfigUUID     string         `gorm:"column:config_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"config_uuid"`
	TenantUUID     string         `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_code_welcome_cfg_tenant" json:"tenant_uuid"`
	CodeUUID       string         `gorm:"column:code_uuid;type:uuid;not null;uniqueIndex:uq_lead_capture_code_welcome_cfg_code" json:"code_uuid"`
	WelcomeEnabled bool           `gorm:"column:welcome_enabled;type:boolean;not null;default:false" json:"welcome_enabled"`
	MessageContent datatypes.JSON `gorm:"column:message_content;type:jsonb;not null;default:'{}'::jsonb" json:"message_content"`
	SyncStatus     string         `gorm:"column:sync_status;type:varchar(32);not null;default:'pending';index:idx_lead_capture_code_welcome_cfg_sync_status" json:"sync_status"`
	LastSyncError  string         `gorm:"column:last_sync_error;type:text" json:"last_sync_error,omitempty"`
	LastSyncedAt   *time.Time     `gorm:"column:last_synced_at;type:timestamptz" json:"last_synced_at,omitempty"`
	Version        int            `gorm:"column:version;type:int;not null;default:1" json:"version"`
	CreatedBy      string         `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy      string         `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (CodeWelcomeConfig) TableName() string {
	return domainmodels.S(domainmodels.TableLeadCaptureCodeWelcomeConfigs)
}

// CodeWelcomeSyncAttempt records every publish attempt with retry visibility.
type CodeWelcomeSyncAttempt struct {
	AttemptUUID   string     `gorm:"column:attempt_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"attempt_uuid"`
	TenantUUID    string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_code_welcome_sync_attempt_tenant" json:"tenant_uuid"`
	CodeUUID      string     `gorm:"column:code_uuid;type:uuid;not null;index:idx_lead_capture_code_welcome_sync_attempt_code" json:"code_uuid"`
	ConfigVersion int        `gorm:"column:config_version;type:int;not null" json:"config_version"`
	TriggerSource string     `gorm:"column:trigger_source;type:varchar(32);not null;index:idx_lead_capture_code_welcome_sync_attempt_trigger" json:"trigger_source"`
	AttemptNo     int        `gorm:"column:attempt_no;type:int;not null" json:"attempt_no"`
	Result        string     `gorm:"column:result;type:varchar(16);not null;index:idx_lead_capture_code_welcome_sync_attempt_result" json:"result"`
	ErrorCode     string     `gorm:"column:error_code;type:varchar(64)" json:"error_code,omitempty"`
	ErrorMessage  string     `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	StartedAt     *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt    *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (CodeWelcomeSyncAttempt) TableName() string {
	return domainmodels.S(domainmodels.TableLeadCaptureCodeWelcomeSyncAttempt)
}

// ChannelCodeEvent persists standardized touch events with idempotency guard.
type ChannelCodeEvent struct {
	EventUUID          string         `gorm:"column:event_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"event_uuid"`
	TenantUUID         string         `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_channel_code_events_tenant" json:"tenant_uuid"`
	Channel            string         `gorm:"column:channel;type:varchar(64);not null;index:idx_lead_capture_channel_code_events_channel" json:"channel"`
	AppType            string         `gorm:"column:app_type;type:varchar(64);not null;index:idx_lead_capture_channel_code_events_app_type" json:"app_type"`
	ChannelAccountUUID string         `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_lead_capture_channel_code_events_account" json:"channel_account_uuid"`
	CodeUUID           string         `gorm:"column:code_uuid;type:uuid;not null;index:idx_lead_capture_channel_code_events_code" json:"code_uuid"`
	ExternalEventID    string         `gorm:"column:external_event_id;type:varchar(128);not null" json:"external_event_id"`
	EventType          string         `gorm:"column:event_type;type:varchar(32);not null;index:idx_lead_capture_channel_code_events_type" json:"event_type"`
	IdempotencyKey     string         `gorm:"column:idempotency_key;type:varchar(255);not null;uniqueIndex:uq_lead_capture_channel_code_events_idempotency" json:"idempotency_key"`
	OccurredAt         time.Time      `gorm:"column:occurred_at;type:timestamptz;not null;index:idx_lead_capture_channel_code_events_occurred" json:"occurred_at"`
	Payload            datatypes.JSON `gorm:"column:payload;type:jsonb;not null;default:'{}'::jsonb" json:"payload"`
	CreatedAt          time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (ChannelCodeEvent) TableName() string {
	return domainmodels.S(domainmodels.TableLeadCaptureChannelCodeEvents)
}

// LeadAttributionRecord keeps first-touch primary and follow-touch mapping records.
type LeadAttributionRecord struct {
	AttributionUUID string    `gorm:"column:attribution_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"attribution_uuid"`
	TenantUUID      string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_attribution_records_tenant" json:"tenant_uuid"`
	LeadUUID        string    `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_capture_attribution_records_lead" json:"lead_uuid"`
	CodeUUID        string    `gorm:"column:code_uuid;type:uuid;not null;index:idx_lead_capture_attribution_records_code" json:"code_uuid"`
	EventUUID       string    `gorm:"column:event_uuid;type:uuid;not null;index:idx_lead_capture_attribution_records_event" json:"event_uuid"`
	IsPrimary       bool      `gorm:"column:is_primary;type:boolean;not null;default:false" json:"is_primary"`
	AttributionType string    `gorm:"column:attribution_type;type:varchar(32);not null" json:"attribution_type"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (LeadAttributionRecord) TableName() string {
	return domainmodels.S(domainmodels.TableLeadCaptureLeadAttributionRecords)
}

// CodeConfigChangeLog stores human-readable config change summary for operations and audit.
type CodeConfigChangeLog struct {
	ChangeUUID      string         `gorm:"column:change_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"change_uuid"`
	TenantUUID      string         `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_code_cfg_change_log_tenant" json:"tenant_uuid"`
	CodeUUID        string         `gorm:"column:code_uuid;type:uuid;not null;index:idx_lead_capture_code_cfg_change_log_code" json:"code_uuid"`
	ConfigUUID      string         `gorm:"column:config_uuid;type:uuid;not null;index:idx_lead_capture_code_cfg_change_log_cfg" json:"config_uuid"`
	Version         int            `gorm:"column:version;type:int;not null" json:"version"`
	Summary         string         `gorm:"column:summary;type:text;not null" json:"summary"`
	ChangedFields   []string       `gorm:"column:changed_fields;type:jsonb;serializer:json" json:"changed_fields"`
	PreviousContent datatypes.JSON `gorm:"column:previous_content;type:jsonb;default:'{}'::jsonb" json:"previous_content"`
	NextContent     datatypes.JSON `gorm:"column:next_content;type:jsonb;default:'{}'::jsonb" json:"next_content"`
	ChangedBy       string         `gorm:"column:changed_by;type:varchar(64);not null" json:"changed_by"`
	CreatedAt       time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (CodeConfigChangeLog) TableName() string {
	return domainmodels.S(domainmodels.TableLeadCaptureCodeConfigChangeLogs)
}
