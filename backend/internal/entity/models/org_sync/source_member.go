package org_sync

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// SourceMember represents a member from a channel account.
type SourceMember struct {
	SourceMemberUUID   string    `gorm:"column:source_member_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"source_member_uuid"`
	TenantUUID         string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_org_sync_source_members_tenant;uniqueIndex:uq_org_sync_source_members_identity,priority:1" json:"tenant_uuid"`
	SourceAccountUUID  string    `gorm:"column:source_account_uuid;type:uuid;not null;index:idx_org_sync_source_members_account;uniqueIndex:uq_org_sync_source_members_identity,priority:2" json:"source_account_uuid"`
	ChannelAccountUUID string    `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_org_sync_source_members_channel" json:"channel_account_uuid"`
	ExternalMemberID   string    `gorm:"column:external_member_id;type:text;not null;uniqueIndex:uq_org_sync_source_members_identity,priority:3" json:"external_member_id"`
	Name               string    `gorm:"column:name;type:text;not null" json:"name"`
	Phone              string    `gorm:"column:phone;type:text;index:idx_org_sync_source_members_phone" json:"phone"`
	Email              string    `gorm:"column:email;type:text;index:idx_org_sync_source_members_email" json:"email"`
	ProfileStatus      string    `gorm:"column:profile_status;type:varchar(32);not null;default:'full';index:idx_org_sync_source_members_profile_status" json:"profile_status"`
	Status             string    `gorm:"column:status;type:varchar(32);not null;default:'active';index:idx_org_sync_source_members_status" json:"status"`
	CreatedAt          time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SourceMember) TableName() string {
	return models.S(models.TableOrgSyncSourceMembers)
}
