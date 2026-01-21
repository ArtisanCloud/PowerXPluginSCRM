package org_sync

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// MemberMapping links a source member to a main org member.
type MemberMapping struct {
	MemberMappingUUID string     `gorm:"column:member_mapping_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"member_mapping_uuid"`
	TenantUUID        string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_org_sync_member_mappings_tenant;uniqueIndex:uq_org_sync_member_mappings_source,priority:1" json:"tenant_uuid"`
	SourceMemberUUID  string     `gorm:"column:source_member_uuid;type:uuid;not null;uniqueIndex:uq_org_sync_member_mappings_source,priority:2" json:"source_member_uuid"`
	MainMemberID      string     `gorm:"column:main_member_id;type:text;not null;index:idx_org_sync_member_mappings_main" json:"main_member_id"`
	MappingStatus     string     `gorm:"column:mapping_status;type:varchar(32);not null;default:'pending';index:idx_org_sync_member_mappings_status" json:"mapping_status"`
	MatchedBy         string     `gorm:"column:matched_by;type:varchar(32)" json:"matched_by,omitempty"`
	ConfirmedBy       string     `gorm:"column:confirmed_by;type:text" json:"confirmed_by,omitempty"`
	ConfirmedAt       *time.Time `gorm:"column:confirmed_at;type:timestamptz" json:"confirmed_at,omitempty"`
	CreatedAt         time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (MemberMapping) TableName() string {
	return models.S(models.TableOrgSyncMemberMappings)
}
