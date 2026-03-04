package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

const (
	LeadSyncTaskProviderFramework     = "framework"
	LeadSyncTaskProviderLocalFallback = "local_fallback"
	LeadSyncTaskResolveExplicit       = "explicit"
	LeadSyncTaskResolveDefault        = "default"
)

type LeadSyncTask struct {
	TaskUUID             string     `gorm:"column:task_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"task_uuid"`
	ExternalTaskID       *string    `gorm:"column:external_task_id;type:text" json:"external_task_id,omitempty"`
	TenantUUID           string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_sync_tasks_tenant" json:"tenant_uuid"`
	Channel              string     `gorm:"column:channel;type:varchar(32);not null;default:'wechat';index:idx_lead_capture_sync_tasks_channel" json:"channel"`
	AppType              string     `gorm:"column:app_type;type:varchar(32);not null;default:'wecom';index:idx_lead_capture_sync_tasks_app" json:"app_type"`
	ChannelAccountUUID   string     `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_lead_capture_sync_tasks_account" json:"channel_account_uuid"`
	AccountResolveSource string     `gorm:"column:account_resolve_source;type:varchar(16);not null;default:'explicit'" json:"account_resolve_source"`
	TaskProvider         string     `gorm:"column:task_provider;type:varchar(32);not null;default:'local_fallback';index:idx_lead_capture_sync_tasks_provider" json:"task_provider"`
	TriggerType          string     `gorm:"column:trigger_type;type:varchar(32);not null;default:'manual'" json:"trigger_type"`
	Status               string     `gorm:"column:status;type:varchar(32);not null;default:'queued';index:idx_lead_capture_sync_tasks_status" json:"status"`
	StatsTotal           int        `gorm:"column:stats_total;type:int;not null;default:0" json:"stats_total"`
	StatsCreated         int        `gorm:"column:stats_created;type:int;not null;default:0" json:"stats_created"`
	StatsUpdated         int        `gorm:"column:stats_updated;type:int;not null;default:0" json:"stats_updated"`
	StatsMerged          int        `gorm:"column:stats_merged;type:int;not null;default:0" json:"stats_merged"`
	ErrorCode            string     `gorm:"column:error_code;type:varchar(128)" json:"error_code,omitempty"`
	ErrorMessage         string     `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	StartedAt            *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt           *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	CreatedAt            time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadSyncTask) TableName() string {
	return models.S(models.TableLeadCaptureSyncTasks)
}
