package org_sync

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/gorm"
)

// SyncLog records each org sync trigger and its summary.
type SyncLog struct {
	SyncLogUUID        string         `gorm:"column:sync_log_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"sync_log_uuid"`
	TenantUUID         string         `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_org_sync_sync_logs_tenant" json:"tenant_uuid"`
	SourceAccountUUID  string         `gorm:"column:source_account_uuid;type:uuid;not null;index:idx_org_sync_sync_logs_account" json:"source_account_uuid"`
	ChannelAccountUUID string         `gorm:"column:channel_account_uuid;type:uuid;index:idx_org_sync_sync_logs_channel" json:"channel_account_uuid,omitempty"`
	Status             string         `gorm:"column:status;type:varchar(32);not null;default:'queued';index:idx_org_sync_sync_logs_status" json:"status"`
	Message            string         `gorm:"column:message;type:text" json:"message,omitempty"`
	UnitsTotal         int            `gorm:"column:units_total;type:int;not null;default:0" json:"units_total"`
	MembersTotal       int            `gorm:"column:members_total;type:int;not null;default:0" json:"members_total"`
	UnitsNew           int            `gorm:"column:units_new;type:int;not null;default:0" json:"units_new"`
	MembersNew         int            `gorm:"column:members_new;type:int;not null;default:0" json:"members_new"`
	UnitsUpdated       int            `gorm:"column:units_updated;type:int;not null;default:0" json:"units_updated"`
	MembersUpdated     int            `gorm:"column:members_updated;type:int;not null;default:0" json:"members_updated"`
	UnitsConflict      int            `gorm:"column:units_conflict;type:int;not null;default:0" json:"units_conflict"`
	MembersConflict    int            `gorm:"column:members_conflict;type:int;not null;default:0" json:"members_conflict"`
	UnitsPending       int            `gorm:"column:units_pending;type:int;not null;default:0" json:"units_pending"`
	MembersPending     int            `gorm:"column:members_pending;type:int;not null;default:0" json:"members_pending"`
	ProgressTotal      int            `gorm:"column:progress_total;type:int;not null;default:0" json:"progress_total"`
	ProgressCurrent    int            `gorm:"column:progress_current;type:int;not null;default:0" json:"progress_current"`
	ProgressPercent    int            `gorm:"column:progress_percent;type:int;not null;default:0" json:"progress_percent"`
	Stage              string         `gorm:"column:stage;type:varchar(64);not null;default:''" json:"stage"`
	DurationMs         int64          `gorm:"column:duration_ms;type:bigint;not null;default:0" json:"duration_ms"`
	CreatedAt          time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;type:timestamptz" json:"deleted_at,omitempty"`
}

func (SyncLog) TableName() string {
	return models.S(models.TableOrgSyncSyncLogs)
}
