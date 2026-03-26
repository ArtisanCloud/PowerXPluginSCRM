package acquisition

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
	"gorm.io/datatypes"
)

const (
	LiveCodeStatusDraft    = "draft"
	LiveCodeStatusActive   = "active"
	LiveCodeStatusDisabled = "disabled"

	WelcomeModeSend   = "send"
	WelcomeModeSilent = "silent"

	WelcomeSyncStatusPending        = "pending"
	WelcomeSyncStatusSyncing        = "syncing"
	WelcomeSyncStatusSuccess        = "success"
	WelcomeSyncStatusFailed         = "failed"
	WelcomeSyncStatusManualRequired = "manual_required"
	WelcomeSyncStatusNotImplemented = "not_implemented"
)

// StaffLiveCode is the v2 independent model for staff acquisition QR/live code.
type StaffLiveCode struct {
	StaffCodeUUID      string    `gorm:"column:staff_code_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"staff_code_uuid"`
	TenantUUID         string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_acq_staff_codes_tenant;uniqueIndex:uq_acq_staff_codes_tenant_code_key,priority:1" json:"tenant_uuid"`
	Channel            string    `gorm:"column:channel;type:varchar(64);not null;index:idx_acq_staff_codes_channel" json:"channel"`
	AppType            string    `gorm:"column:app_type;type:varchar(64);not null;index:idx_acq_staff_codes_app_type" json:"app_type"`
	ChannelAccountUUID string    `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_acq_staff_codes_account" json:"channel_account_uuid"`
	ActivityName       string    `gorm:"column:activity_name;type:varchar(128);not null;index:idx_acq_staff_codes_activity" json:"activity_name"`
	CodeKey            string    `gorm:"column:code_key;type:varchar(128);not null;uniqueIndex:uq_acq_staff_codes_tenant_code_key,priority:2" json:"code_key"`
	MemberUUIDs        []string  `gorm:"column:member_uuids;type:jsonb;serializer:json" json:"member_uuids"`
	CorpTagIDs         []string  `gorm:"column:corp_tag_ids;type:jsonb;serializer:json" json:"corp_tag_ids"`
	RemarkEnabled      bool      `gorm:"column:new_customer_remark_enabled;type:boolean;not null;default:false" json:"new_customer_remark_enabled"`
	Status             string    `gorm:"column:status;type:varchar(32);not null;default:'draft';index:idx_acq_staff_codes_status" json:"status"`
	CreatedBy          string    `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy          string    `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt          time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (StaffLiveCode) TableName() string {
	return domainmodels.S(domainmodels.TableAcquisitionStaffLiveCodes)
}

// StaffWelcomeConfig binds one welcome payload to one staff code.
type StaffWelcomeConfig struct {
	ConfigUUID     string         `gorm:"column:config_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"config_uuid"`
	TenantUUID     string         `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_acq_staff_welcome_tenant" json:"tenant_uuid"`
	StaffCodeUUID  string         `gorm:"column:staff_code_uuid;type:uuid;not null;uniqueIndex:uq_acq_staff_welcome_code" json:"staff_code_uuid"`
	WelcomeMode    string         `gorm:"column:welcome_mode;type:varchar(32);not null;default:'send'" json:"welcome_mode"`
	ContentBlocks  datatypes.JSON `gorm:"column:content_blocks;type:jsonb;not null;default:'[]'::jsonb" json:"content_blocks"`
	PayloadPreview datatypes.JSON `gorm:"column:payload_preview;type:jsonb;not null;default:'{}'::jsonb" json:"payload_preview"`
	SyncStatus     string         `gorm:"column:sync_status;type:varchar(32);not null;default:'pending';index:idx_acq_staff_welcome_sync_status" json:"sync_status"`
	LastSyncError  string         `gorm:"column:last_sync_error;type:text" json:"last_sync_error,omitempty"`
	LastSyncedAt   *time.Time     `gorm:"column:last_synced_at;type:timestamptz" json:"last_synced_at,omitempty"`
	Version        int            `gorm:"column:version;type:int;not null;default:1" json:"version"`
	CreatedBy      string         `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy      string         `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (StaffWelcomeConfig) TableName() string {
	return domainmodels.S(domainmodels.TableAcquisitionStaffWelcomeConfigs)
}

// StaffWelcomeSyncAttempt keeps publish attempt history.
type StaffWelcomeSyncAttempt struct {
	AttemptUUID   string     `gorm:"column:attempt_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"attempt_uuid"`
	TenantUUID    string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_acq_staff_welcome_attempt_tenant" json:"tenant_uuid"`
	StaffCodeUUID string     `gorm:"column:staff_code_uuid;type:uuid;not null;index:idx_acq_staff_welcome_attempt_code" json:"staff_code_uuid"`
	ConfigVersion int        `gorm:"column:config_version;type:int;not null" json:"config_version"`
	AttemptNo     int        `gorm:"column:attempt_no;type:int;not null" json:"attempt_no"`
	Result        string     `gorm:"column:result;type:varchar(16);not null" json:"result"`
	ErrorCode     string     `gorm:"column:error_code;type:varchar(64)" json:"error_code,omitempty"`
	ErrorMessage  string     `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	StartedAt     *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt    *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (StaffWelcomeSyncAttempt) TableName() string {
	return domainmodels.S(domainmodels.TableAcquisitionStaffWelcomeSyncAttempts)
}

// GroupLiveCode is the group-code skeleton for v2 phase.
type GroupLiveCode struct {
	GroupCodeUUID      string    `gorm:"column:group_code_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"group_code_uuid"`
	TenantUUID         string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_acq_group_codes_tenant" json:"tenant_uuid"`
	Channel            string    `gorm:"column:channel;type:varchar(64);not null;index:idx_acq_group_codes_channel" json:"channel"`
	AppType            string    `gorm:"column:app_type;type:varchar(64);not null;index:idx_acq_group_codes_app_type" json:"app_type"`
	ChannelAccountUUID string    `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_acq_group_codes_account" json:"channel_account_uuid"`
	ActivityName       string    `gorm:"column:activity_name;type:varchar(128);not null" json:"activity_name"`
	Status             string    `gorm:"column:status;type:varchar(32);not null;default:'draft'" json:"status"`
	CapabilityStatus   string    `gorm:"column:capability_status;type:varchar(32);not null;default:'not_implemented'" json:"capability_status"`
	CreatedBy          string    `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy          string    `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt          time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (GroupLiveCode) TableName() string {
	return domainmodels.S(domainmodels.TableAcquisitionGroupLiveCodes)
}
