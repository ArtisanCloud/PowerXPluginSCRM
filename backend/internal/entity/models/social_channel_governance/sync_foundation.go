package social_channel_governance

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	CapabilityStatusSupported    = "supported"
	CapabilityStatusPartial      = "partial"
	CapabilityStatusNotSupported = "not_supported"
	CapabilityStatusPlanned      = "planned"
)

type SyncFoundationBinding struct {
	BindingUUID        string            `gorm:"column:binding_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"binding_uuid"`
	TenantUUID         string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_sync_foundation_bindings_tenant;uniqueIndex:uq_social_sync_foundation_binding,priority:1" json:"tenant_uuid"`
	CorpID             string            `gorm:"column:corp_id;type:text;not null;default:'';uniqueIndex:uq_social_sync_foundation_binding,priority:2" json:"corp_id"`
	ChannelAccountUUID string            `gorm:"column:channel_account_uuid;type:uuid;not null;uniqueIndex:uq_social_sync_foundation_binding,priority:3" json:"channel_account_uuid"`
	AccessMode         string            `gorm:"column:access_mode;type:varchar(32);not null;default:'delegated';index:idx_social_sync_foundation_bindings_access_mode" json:"access_mode"`
	AuthStatus         string            `gorm:"column:auth_status;type:varchar(32);not null;default:'pending';index:idx_social_sync_foundation_bindings_auth_status" json:"auth_status"`
	TokenStatus        string            `gorm:"column:token_status;type:varchar(32);not null;default:'invalid'" json:"token_status"`
	CallbackStatus     string            `gorm:"column:callback_status;type:varchar(32);not null;default:'pending'" json:"callback_status"`
	LastAuthorizedAt   *time.Time        `gorm:"column:last_authorized_at;type:timestamptz" json:"last_authorized_at,omitempty"`
	LastSyncAt         *time.Time        `gorm:"column:last_sync_at;type:timestamptz" json:"last_sync_at,omitempty"`
	Metadata           datatypes.JSONMap `gorm:"column:metadata;type:jsonb;default:'{}'::jsonb" json:"metadata"`
	CreatedAt          time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SyncFoundationBinding) TableName() string {
	return models.S(models.TableSocialSyncFoundationBindings)
}

type SyncJob struct {
	JobUUID         string            `gorm:"column:job_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"job_uuid"`
	TenantUUID      string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_sync_jobs_tenant;uniqueIndex:uq_social_sync_jobs_idempotency,priority:1" json:"tenant_uuid"`
	Domain          string            `gorm:"column:domain;type:varchar(64);not null;index:idx_social_sync_jobs_domain" json:"domain"`
	Direction       string            `gorm:"column:direction;type:varchar(16);not null;default:'pull'" json:"direction"`
	Mode            string            `gorm:"column:mode;type:varchar(32);not null;default:'incremental'" json:"mode"`
	Status          string            `gorm:"column:status;type:varchar(32);not null;default:'pending';index:idx_social_sync_jobs_status" json:"status"`
	AttemptNo       int               `gorm:"column:attempt_no;type:int;not null;default:0" json:"attempt_no"`
	MaxAttempts     int               `gorm:"column:max_attempts;type:int;not null;default:3" json:"max_attempts"`
	IdempotencyKey  string            `gorm:"column:idempotency_key;type:text;not null;default:'';uniqueIndex:uq_social_sync_jobs_idempotency,priority:2" json:"idempotency_key"`
	Payload         datatypes.JSONMap `gorm:"column:payload;type:jsonb;default:'{}'::jsonb" json:"payload"`
	ResultSummary   datatypes.JSONMap `gorm:"-" json:"result_summary,omitempty"`
	ErrorCode       string            `gorm:"column:error_code;type:varchar(64);not null;default:''" json:"error_code"`
	ErrorMessage    string            `gorm:"column:error_message;type:text;not null;default:''" json:"error_message"`
	StartedAt       *time.Time        `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt      *time.Time        `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	CapabilityState string            `gorm:"column:capability_status;type:varchar(32);not null;default:'supported'" json:"capability_status"`
	CreatedAt       time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SyncJob) TableName() string {
	return models.S(models.TableSocialSyncJobs)
}

type SyncCheckpoint struct {
	CheckpointUUID  string    `gorm:"column:checkpoint_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"checkpoint_uuid"`
	TenantUUID      string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_sync_checkpoints_tenant;uniqueIndex:uq_social_sync_checkpoints_scope,priority:1" json:"tenant_uuid"`
	Domain          string    `gorm:"column:domain;type:varchar(64);not null;uniqueIndex:uq_social_sync_checkpoints_scope,priority:2" json:"domain"`
	Direction       string    `gorm:"column:direction;type:varchar(16);not null;uniqueIndex:uq_social_sync_checkpoints_scope,priority:3" json:"direction"`
	Cursor          string    `gorm:"column:cursor;type:text;not null;default:''" json:"cursor"`
	SnapshotVersion string    `gorm:"column:snapshot_version;type:text;not null;default:''" json:"snapshot_version"`
	LastEventTime   time.Time `gorm:"column:last_event_time;type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"last_event_time"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

func (SyncCheckpoint) TableName() string {
	return models.S(models.TableSocialSyncCheckpoints)
}

type SyncConflict struct {
	ConflictUUID       string            `gorm:"column:conflict_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"conflict_uuid"`
	TenantUUID         string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_sync_conflicts_tenant" json:"tenant_uuid"`
	Domain             string            `gorm:"column:domain;type:varchar(64);not null;index:idx_social_sync_conflicts_domain" json:"domain"`
	EntityType         string            `gorm:"column:entity_type;type:varchar(64);not null;default:''" json:"entity_type"`
	EntityKey          string            `gorm:"column:entity_key;type:text;not null;default:''" json:"entity_key"`
	ResolutionStrategy string            `gorm:"column:resolution_strategy;type:varchar(32);not null;default:'remote_first'" json:"resolution_strategy"`
	Status             string            `gorm:"column:status;type:varchar(32);not null;default:'open';index:idx_social_sync_conflicts_status" json:"status"`
	LocalValue         datatypes.JSONMap `gorm:"column:local_value;type:jsonb;default:'{}'::jsonb" json:"local_value"`
	RemoteValue        datatypes.JSONMap `gorm:"column:remote_value;type:jsonb;default:'{}'::jsonb" json:"remote_value"`
	ResolvedBy         string            `gorm:"column:resolved_by;type:text;not null;default:''" json:"resolved_by"`
	ResolvedAt         *time.Time        `gorm:"column:resolved_at;type:timestamptz" json:"resolved_at,omitempty"`
	CreatedAt          time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SyncConflict) TableName() string {
	return models.S(models.TableSocialSyncConflicts)
}

type SyncWritebackPolicy struct {
	PolicyUUID       string            `gorm:"column:policy_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"policy_uuid"`
	TenantUUID       string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_sync_writeback_policies_tenant;uniqueIndex:uq_social_sync_writeback_policies_scope,priority:1" json:"tenant_uuid"`
	Domain           string            `gorm:"column:domain;type:varchar(64);not null;uniqueIndex:uq_social_sync_writeback_policies_scope,priority:2" json:"domain"`
	MappingRules     datatypes.JSONMap `gorm:"column:mapping_rules;type:jsonb;default:'{}'::jsonb" json:"mapping_rules"`
	ProtectedFields  datatypes.JSONMap `gorm:"column:protected_fields;type:jsonb;default:'{}'::jsonb" json:"protected_fields"`
	OverwriteMode    string            `gorm:"column:overwrite_mode;type:varchar(16);not null;default:'safe'" json:"overwrite_mode"`
	Enabled          bool              `gorm:"column:enabled;type:boolean;not null;default:true" json:"enabled"`
	CapabilityStatus string            `gorm:"column:capability_status;type:varchar(32);not null;default:'supported'" json:"capability_status"`
	CreatedAt        time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SyncWritebackPolicy) TableName() string {
	return models.S(models.TableSocialSyncWritebackPolicies)
}

type SyncDeadLetterItem struct {
	DeadLetterUUID   string            `gorm:"column:dead_letter_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"dead_letter_uuid"`
	TenantUUID       string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_sync_dead_letters_tenant" json:"tenant_uuid"`
	JobUUID          string            `gorm:"column:job_uuid;type:uuid;not null;index:idx_social_sync_dead_letters_job" json:"job_uuid"`
	Domain           string            `gorm:"column:domain;type:varchar(64);not null;index:idx_social_sync_dead_letters_domain" json:"domain"`
	Direction        string            `gorm:"column:direction;type:varchar(16);not null;default:'pull'" json:"direction"`
	LastErrorCode    string            `gorm:"column:last_error_code;type:varchar(64);not null;default:''" json:"last_error_code"`
	LastErrorMessage string            `gorm:"column:last_error_message;type:text;not null;default:''" json:"last_error_message"`
	RetryExhaustedAt time.Time         `gorm:"column:retry_exhausted_at;type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"retry_exhausted_at"`
	ReplayStatus     string            `gorm:"column:replay_status;type:varchar(32);not null;default:'pending'" json:"replay_status"`
	ReplayedBy       string            `gorm:"column:replayed_by;type:text;not null;default:''" json:"replayed_by"`
	ReplayedAt       *time.Time        `gorm:"column:replayed_at;type:timestamptz" json:"replayed_at,omitempty"`
	Payload          datatypes.JSONMap `gorm:"column:payload;type:jsonb;default:'{}'::jsonb" json:"payload"`
	CreatedAt        time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SyncDeadLetterItem) TableName() string {
	return models.S(models.TableSocialSyncDeadLetters)
}
