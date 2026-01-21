package org_sync

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/gorm"
)

// SourceAccount represents a channel account bound to org sync.
type SourceAccount struct {
	SourceAccountUUID  string         `gorm:"column:source_account_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"source_account_uuid"`
	TenantUUID         string         `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_org_sync_source_accounts_tenant" json:"tenant_uuid"`
	Provider           string         `gorm:"column:provider;type:varchar(32);not null;index:idx_org_sync_source_accounts_provider" json:"provider"`
	AppType            string         `gorm:"column:app_type;type:varchar(64);not null" json:"app_type"`
	ChannelAccountUUID *string        `gorm:"column:channel_account_uuid;type:uuid;index:idx_org_sync_source_accounts_channel" json:"channel_account_uuid,omitempty"`
	DisplayName        string         `gorm:"column:display_name;type:text;not null" json:"display_name"`
	Status             string         `gorm:"column:status;type:varchar(32);not null;default:'active';index:idx_org_sync_source_accounts_status" json:"status"`
	LastSyncAt         *time.Time     `gorm:"column:last_sync_at;type:timestamptz" json:"last_sync_at,omitempty"`
	LastSyncStatus     string         `gorm:"column:last_sync_status;type:varchar(32)" json:"last_sync_status,omitempty"`
	LastSyncMessage    string         `gorm:"column:last_sync_message;type:text" json:"last_sync_message,omitempty"`
	CreatedAt          time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;type:timestamptz" json:"deleted_at,omitempty"`
}

func (SourceAccount) TableName() string {
	return models.S(models.TableOrgSyncSourceAccounts)
}
