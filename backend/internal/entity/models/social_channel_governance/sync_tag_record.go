package social_channel_governance

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/gorm"
)

type SyncTagRecord struct {
	TagUUID            string         `gorm:"column:tag_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"tag_uuid"`
	TenantUUID         string         `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_sync_tag_records_tenant;uniqueIndex:uq_social_sync_tag_records_identity,priority:1" json:"tenant_uuid"`
	ChannelAccountUUID string         `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_social_sync_tag_records_channel;uniqueIndex:uq_social_sync_tag_records_identity,priority:2" json:"channel_account_uuid"`
	ChannelCode        string         `gorm:"column:channel_code;type:varchar(32);not null;default:'wechat'" json:"channel_code"`
	AppType            string         `gorm:"column:app_type;type:varchar(32);not null;default:'wecom'" json:"app_type"`
	RemoteTagID        string         `gorm:"column:remote_tag_id;type:text;not null;default:'';uniqueIndex:uq_social_sync_tag_records_identity,priority:3" json:"remote_tag_id"`
	RemoteGroupID      string         `gorm:"column:remote_group_id;type:text;not null;default:''" json:"remote_group_id"`
	RemoteGroupName    string         `gorm:"column:remote_group_name;type:text;not null;default:''" json:"remote_group_name"`
	TagName            string         `gorm:"column:tag_name;type:text;not null;default:''" json:"tag_name"`
	Version            string         `gorm:"column:version;type:text;not null;default:''" json:"version"`
	TagOrder           int            `gorm:"column:tag_order;type:int;not null;default:0" json:"tag_order"`
	SnapshotVersion    string         `gorm:"column:snapshot_version;type:text;not null;default:''" json:"snapshot_version"`
	LastPulledAt       *time.Time     `gorm:"column:last_pulled_at;type:timestamptz" json:"last_pulled_at,omitempty"`
	Source             string         `gorm:"column:source;type:varchar(32);not null;default:'wecom'" json:"source"`
	CreatedAt          time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;type:timestamptz;index:idx_social_sync_tag_records_deleted_at" json:"deleted_at,omitempty"`
}

func (SyncTagRecord) TableName() string {
	return models.S(models.TableSocialSyncTagRecords)
}
