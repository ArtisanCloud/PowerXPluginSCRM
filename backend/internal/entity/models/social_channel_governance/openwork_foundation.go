package social_channel_governance

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	WeComAuthBindingStatusPending  = "pending"
	WeComAuthBindingStatusActive   = "active"
	WeComAuthBindingStatusCanceled = "canceled"
	WeComAuthBindingStatusDisabled = "disabled"
	SyncJobStatusQueued            = "queued"
	SyncJobStatusRunning           = "running"
	SyncJobStatusSuccess           = "success"
	SyncJobStatusFailed            = "failed"
	SyncJobStatusDeadLetter        = "dead_letter"
	SyncConflictStatusOpen         = "open"
	SyncConflictStatusReplayed     = "replayed"
	SyncConflictStatusResolved     = "resolved"
	SyncDomainTags                 = "tags"
	SyncDomainOrg                  = "org"
	SyncDomainExternalContacts     = "external_contacts"
	SyncModeBootstrap              = "bootstrap"
	SyncModeIncremental            = "incremental"
	SyncModePushback               = "pushback"
)

// ChannelAuthBinding stores channel-level authorization binding in a provider-agnostic form.
type ChannelAuthBinding struct {
	BindingUUID        string            `gorm:"column:binding_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"binding_uuid"`
	TenantUUID         string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_channel_auth_bindings_tenant" json:"tenant_uuid"`
	ChannelAccountUUID string            `gorm:"column:channel_account_uuid;type:uuid;index:idx_social_channel_auth_bindings_channel_account" json:"channel_account_uuid,omitempty"`
	ChannelCode        string            `gorm:"column:channel_code;type:varchar(32);not null;default:'wechat';index:idx_social_channel_auth_bindings_channel" json:"channel_code"`
	AppType            string            `gorm:"column:app_type;type:varchar(32);not null;default:'wecom';index:idx_social_channel_auth_bindings_app" json:"app_type"`
	ProviderCode       string            `gorm:"column:provider_code;type:varchar(64);not null;default:'openwork';index:idx_social_channel_auth_bindings_provider" json:"provider_code"`
	ExternalOrgID      string            `gorm:"column:external_org_id;type:text;not null;default:'';index:idx_social_channel_auth_bindings_org;uniqueIndex:uq_social_channel_auth_binding_identity,priority:1" json:"external_org_id"`
	ExternalAgentID    string            `gorm:"column:external_agent_id;type:text;not null;default:'';uniqueIndex:uq_social_channel_auth_binding_identity,priority:2" json:"external_agent_id"`
	ExternalOrgName    string            `gorm:"column:external_org_name;type:text;not null;default:''" json:"external_org_name"`
	CredentialA        string            `gorm:"column:credential_a;type:text;not null;default:''" json:"-"`
	CredentialB        string            `gorm:"column:credential_b;type:text;not null;default:''" json:"-"`
	CredentialC        string            `gorm:"column:credential_c;type:text;not null;default:''" json:"-"`
	Status             string            `gorm:"column:status;type:varchar(32);not null;default:'pending';index:idx_social_channel_auth_bindings_status" json:"status"`
	IsDefault          bool              `gorm:"column:is_default;type:boolean;not null;default:false;index:idx_social_channel_auth_bindings_default" json:"is_default"`
	DefaultSwitchedAt  *time.Time        `gorm:"column:default_switched_at;type:timestamptz" json:"default_switched_at,omitempty"`
	LastEventType      string            `gorm:"column:last_event_type;type:varchar(64);not null;default:''" json:"last_event_type"`
	LastEventAt        *time.Time        `gorm:"column:last_event_at;type:timestamptz" json:"last_event_at,omitempty"`
	AuthScope          datatypes.JSONMap `gorm:"column:auth_scope;type:jsonb;default:'{}'::jsonb" json:"auth_scope"`
	Metadata           datatypes.JSONMap `gorm:"column:metadata;type:jsonb;default:'{}'::jsonb" json:"metadata"`
	CreatedAt          time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt    `gorm:"column:deleted_at;type:timestamptz" json:"deleted_at,omitempty"`
}

func (ChannelAuthBinding) TableName() string {
	return models.S(models.TableSocialChannelAuthBindings)
}

// ChannelAuthEvent stores provider callback/event records in a provider-agnostic form.
type ChannelAuthEvent struct {
	EventUUID     string            `gorm:"column:event_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"event_uuid"`
	TenantUUID    string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_channel_auth_events_tenant" json:"tenant_uuid"`
	ChannelCode   string            `gorm:"column:channel_code;type:varchar(32);not null;default:'wechat';index:idx_social_channel_auth_events_channel" json:"channel_code"`
	ProviderCode  string            `gorm:"column:provider_code;type:varchar(64);not null;default:'openwork';index:idx_social_channel_auth_events_provider" json:"provider_code"`
	EventType     string            `gorm:"column:event_type;type:varchar(64);not null;index:idx_social_channel_auth_events_type" json:"event_type"`
	ExternalOrgID string            `gorm:"column:external_org_id;type:text;not null;default:'';index:idx_social_channel_auth_events_org" json:"external_org_id"`
	ExternalAppID string            `gorm:"column:external_app_id;type:text;not null;default:''" json:"external_app_id"`
	EventTime     *time.Time        `gorm:"column:event_time;type:timestamptz;index:idx_social_channel_auth_events_time" json:"event_time,omitempty"`
	EventKey      string            `gorm:"column:event_key;type:text;not null;uniqueIndex:uq_social_channel_auth_event_key" json:"event_key"`
	Payload       datatypes.JSONMap `gorm:"column:payload;type:jsonb;default:'{}'::jsonb" json:"payload"`
	CreatedAt     time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

func (ChannelAuthEvent) TableName() string {
	return models.S(models.TableSocialChannelAuthEvents)
}

// WeComOpenAuthBinding stores tenant<->corp authorization relationship for OpenWork flow.
type WeComOpenAuthBinding struct {
	BindingUUID        string            `gorm:"column:binding_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"binding_uuid"`
	TenantUUID         string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_wecom_auth_bindings_tenant" json:"tenant_uuid"`
	ChannelAccountUUID string            `gorm:"column:channel_account_uuid;type:uuid;index:idx_social_wecom_auth_bindings_channel_account" json:"channel_account_uuid,omitempty"`
	ChannelCode        string            `gorm:"column:channel_code;type:varchar(32);not null;default:'wechat';index:idx_social_wecom_auth_bindings_channel" json:"channel_code"`
	AppType            string            `gorm:"column:app_type;type:varchar(32);not null;default:'wecom';index:idx_social_wecom_auth_bindings_app" json:"app_type"`
	SuiteID            string            `gorm:"column:suite_id;type:text;not null;index:idx_social_wecom_auth_bindings_suite" json:"suite_id"`
	CorpID             string            `gorm:"column:corp_id;type:text;not null;index:idx_social_wecom_auth_bindings_corp;uniqueIndex:uq_social_wecom_auth_binding_identity,priority:1" json:"corp_id"`
	AgentID            string            `gorm:"column:agent_id;type:text;not null;default:'';uniqueIndex:uq_social_wecom_auth_binding_identity,priority:2" json:"agent_id"`
	CorpName           string            `gorm:"column:corp_name;type:text;not null;default:''" json:"corp_name"`
	PermanentCode      string            `gorm:"column:permanent_code;type:text;not null;default:''" json:"-"`
	SuiteAccessToken   string            `gorm:"column:suite_access_token;type:text;not null;default:''" json:"-"`
	SuiteTicket        string            `gorm:"column:suite_ticket;type:text;not null;default:''" json:"-"`
	Status             string            `gorm:"column:status;type:varchar(32);not null;default:'pending';index:idx_social_wecom_auth_bindings_status" json:"status"`
	IsDefault          bool              `gorm:"column:is_default;type:boolean;not null;default:false;index:idx_social_wecom_auth_bindings_default" json:"is_default"`
	DefaultSwitchedAt  *time.Time        `gorm:"column:default_switched_at;type:timestamptz" json:"default_switched_at,omitempty"`
	LastEventType      string            `gorm:"column:last_event_type;type:varchar(64);not null;default:''" json:"last_event_type"`
	LastEventAt        *time.Time        `gorm:"column:last_event_at;type:timestamptz" json:"last_event_at,omitempty"`
	AuthScope          datatypes.JSONMap `gorm:"column:auth_scope;type:jsonb;default:'{}'::jsonb" json:"auth_scope"`
	Metadata           datatypes.JSONMap `gorm:"column:metadata;type:jsonb;default:'{}'::jsonb" json:"metadata"`
	CreatedAt          time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt    `gorm:"column:deleted_at;type:timestamptz" json:"deleted_at,omitempty"`
}

func (WeComOpenAuthBinding) TableName() string {
	return models.S(models.TableSocialWeComAuthBindings)
}

// WeComOpenAuthEvent stores ingested OpenWork callback events.
type WeComOpenAuthEvent struct {
	EventUUID  string            `gorm:"column:event_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"event_uuid"`
	TenantUUID string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_wecom_auth_events_tenant" json:"tenant_uuid"`
	SuiteID    string            `gorm:"column:suite_id;type:text;not null;index:idx_social_wecom_auth_events_suite" json:"suite_id"`
	EventType  string            `gorm:"column:event_type;type:varchar(64);not null;index:idx_social_wecom_auth_events_type" json:"event_type"`
	CorpID     string            `gorm:"column:corp_id;type:text;not null;default:'';index:idx_social_wecom_auth_events_corp" json:"corp_id"`
	AgentID    string            `gorm:"column:agent_id;type:text;not null;default:''" json:"agent_id"`
	EventTime  *time.Time        `gorm:"column:event_time;type:timestamptz;index:idx_social_wecom_auth_events_time" json:"event_time,omitempty"`
	EventKey   string            `gorm:"column:event_key;type:text;not null;uniqueIndex:uq_social_wecom_auth_event_key" json:"event_key"`
	Payload    datatypes.JSONMap `gorm:"column:payload;type:jsonb;default:'{}'::jsonb" json:"payload"`
	CreatedAt  time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

func (WeComOpenAuthEvent) TableName() string {
	return models.S(models.TableSocialWeComAuthEvents)
}

// SyncBaselineJob tracks dual-sync baseline jobs for tags/org/external contacts.
type SyncBaselineJob struct {
	JobUUID         string            `gorm:"column:job_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"job_uuid"`
	TenantUUID      string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_sync_jobs_tenant" json:"tenant_uuid"`
	BindingUUID     string            `gorm:"column:binding_uuid;type:uuid;not null;index:idx_social_sync_jobs_binding" json:"binding_uuid"`
	Domain          string            `gorm:"column:domain;type:varchar(64);not null;index:idx_social_sync_jobs_domain" json:"domain"`
	Mode            string            `gorm:"column:mode;type:varchar(32);not null;index:idx_social_sync_jobs_mode" json:"mode"`
	Status          string            `gorm:"column:status;type:varchar(32);not null;default:'queued';index:idx_social_sync_jobs_status" json:"status"`
	IdempotencyKey  string            `gorm:"column:idempotency_key;type:text;not null;default:'';index:idx_social_sync_jobs_idempotency" json:"idempotency_key"`
	ResolveSource   string            `gorm:"column:resolve_source;type:varchar(16);not null;default:'default'" json:"resolve_source"`
	TotalCount      int               `gorm:"column:total_count;type:int;not null;default:0" json:"total_count"`
	SuccessCount    int               `gorm:"column:success_count;type:int;not null;default:0" json:"success_count"`
	FailedCount     int               `gorm:"column:failed_count;type:int;not null;default:0" json:"failed_count"`
	ConflictCount   int               `gorm:"column:conflict_count;type:int;not null;default:0" json:"conflict_count"`
	RetryCount      int               `gorm:"column:retry_count;type:int;not null;default:0" json:"retry_count"`
	MaxRetries      int               `gorm:"column:max_retries;type:int;not null;default:3" json:"max_retries"`
	LastError       string            `gorm:"column:last_error;type:text;not null;default:''" json:"last_error"`
	NextRetryAt     *time.Time        `gorm:"column:next_retry_at;type:timestamptz" json:"next_retry_at,omitempty"`
	DeadLetterAt    *time.Time        `gorm:"column:dead_letter_at;type:timestamptz" json:"dead_letter_at,omitempty"`
	StartedAt       *time.Time        `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt      *time.Time        `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	WriteBackFields datatypes.JSONMap `gorm:"column:write_back_fields;type:jsonb;default:'{}'::jsonb" json:"write_back_fields"`
	Context         datatypes.JSONMap `gorm:"column:context;type:jsonb;default:'{}'::jsonb" json:"context"`
	CreatedAt       time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SyncBaselineJob) TableName() string {
	return models.S(models.TableSocialSyncBaselineJobs)
}

// SyncConflictRecord stores bidirectional sync conflict queue records.
type SyncConflictRecord struct {
	ConflictUUID    string            `gorm:"column:conflict_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"conflict_uuid"`
	TenantUUID      string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_sync_conflicts_tenant" json:"tenant_uuid"`
	JobUUID         string            `gorm:"column:job_uuid;type:uuid;not null;index:idx_social_sync_conflicts_job" json:"job_uuid"`
	BindingUUID     string            `gorm:"column:binding_uuid;type:uuid;not null;index:idx_social_sync_conflicts_binding" json:"binding_uuid"`
	Domain          string            `gorm:"column:domain;type:varchar(64);not null;index:idx_social_sync_conflicts_domain" json:"domain"`
	ConflictKey     string            `gorm:"column:conflict_key;type:text;not null;index:idx_social_sync_conflicts_key" json:"conflict_key"`
	Status          string            `gorm:"column:status;type:varchar(32);not null;default:'open';index:idx_social_sync_conflicts_status" json:"status"`
	Reason          string            `gorm:"column:reason;type:text;not null;default:''" json:"reason"`
	ExternalVersion string            `gorm:"column:external_version;type:text;not null;default:''" json:"external_version"`
	LocalVersion    string            `gorm:"column:local_version;type:text;not null;default:''" json:"local_version"`
	ResolutionNote  string            `gorm:"column:resolution_note;type:text;not null;default:''" json:"resolution_note"`
	ReplayCount     int               `gorm:"column:replay_count;type:int;not null;default:0" json:"replay_count"`
	LastReplayedAt  *time.Time        `gorm:"column:last_replayed_at;type:timestamptz" json:"last_replayed_at,omitempty"`
	Payload         datatypes.JSONMap `gorm:"column:payload;type:jsonb;default:'{}'::jsonb" json:"payload"`
	CreatedAt       time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SyncConflictRecord) TableName() string {
	return models.S(models.TableSocialSyncConflictRecords)
}
