package org_sync

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// SourceMemberProfile stores authorized profile details for a source member.
type SourceMemberProfile struct {
	SourceMemberProfileUUID string    `gorm:"column:source_member_profile_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"source_member_profile_uuid"`
	TenantUUID              string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_org_sync_member_profiles_tenant;uniqueIndex:uq_org_sync_member_profiles_source,priority:1" json:"tenant_uuid"`
	SourceMemberUUID        string    `gorm:"column:source_member_uuid;type:uuid;not null;uniqueIndex:uq_org_sync_member_profiles_source,priority:2" json:"source_member_uuid"`
	ChannelAccountUUID      string    `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_org_sync_member_profiles_channel" json:"channel_account_uuid"`
	ExternalMemberID        string    `gorm:"column:external_member_id;type:text;not null;index:idx_org_sync_member_profiles_external" json:"external_member_id"`
	Name                    string    `gorm:"column:name;type:text;not null" json:"name"`
	Phone                   string    `gorm:"column:phone;type:text;index:idx_org_sync_member_profiles_phone" json:"phone"`
	Email                   string    `gorm:"column:email;type:text;index:idx_org_sync_member_profiles_email" json:"email"`
	BizMail                 string    `gorm:"column:biz_mail;type:text" json:"biz_mail"`
	Position                string    `gorm:"column:position;type:text" json:"position"`
	MainDepartmentID        string    `gorm:"column:main_department_id;type:text" json:"main_department_id"`
	Address                 string    `gorm:"column:address;type:text" json:"address"`
	AvatarURL               string    `gorm:"column:avatar_url;type:text" json:"avatar_url"`
	CreatedAt               time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SourceMemberProfile) TableName() string {
	return models.S(models.TableOrgSyncMemberProfiles)
}
