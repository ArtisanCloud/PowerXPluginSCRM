package org_sync

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// MemberBinding links IAM member and external member by channel account.
type MemberBinding struct {
	MemberBindingUUID  string     `gorm:"column:member_binding_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"member_binding_uuid"`
	TenantUUID         string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_org_sync_member_bindings_tenant;uniqueIndex:uq_org_sync_member_bindings_identity,priority:1" json:"tenant_uuid"`
	ChannelAccountUUID string     `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_org_sync_member_bindings_channel;uniqueIndex:uq_org_sync_member_bindings_identity,priority:2" json:"channel_account_uuid"`
	MainMemberID       string     `gorm:"column:main_member_id;type:text;not null;index:idx_org_sync_member_bindings_main;uniqueIndex:uq_org_sync_member_bindings_identity,priority:3" json:"main_member_id"`
	ExternalMemberID   string     `gorm:"column:external_member_id;type:text;not null;index:idx_org_sync_member_bindings_external;uniqueIndex:uq_org_sync_member_bindings_identity,priority:4" json:"external_member_id"`
	SyncStatus         string     `gorm:"column:sync_status;type:varchar(32);not null;default:'synced';index:idx_org_sync_member_bindings_status" json:"sync_status"`
	LastPulledAt       *time.Time `gorm:"column:last_pulled_at;type:timestamptz" json:"last_pulled_at,omitempty"`
	LastPushedAt       *time.Time `gorm:"column:last_pushed_at;type:timestamptz" json:"last_pushed_at,omitempty"`
	CreatedAt          time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (MemberBinding) TableName() string {
	return models.S(models.TableOrgSyncMemberBindings)
}
