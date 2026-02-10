package org_sync

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// SourceMemberUnit maps source members to source units (departments).
type SourceMemberUnit struct {
	SourceMemberUnitUUID string    `gorm:"column:source_member_unit_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"source_member_unit_uuid"`
	TenantUUID           string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_org_sync_member_units_tenant;uniqueIndex:uq_org_sync_member_units_identity,priority:1" json:"tenant_uuid"`
	SourceAccountUUID    string    `gorm:"column:source_account_uuid;type:uuid;not null;index:idx_org_sync_member_units_account;uniqueIndex:uq_org_sync_member_units_identity,priority:2" json:"source_account_uuid"`
	ChannelAccountUUID   string    `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_org_sync_member_units_channel" json:"channel_account_uuid"`
	SourceMemberUUID     string    `gorm:"column:source_member_uuid;type:uuid;not null;index:idx_org_sync_member_units_member;uniqueIndex:uq_org_sync_member_units_identity,priority:3" json:"source_member_uuid"`
	SourceUnitUUID       string    `gorm:"column:source_unit_uuid;type:uuid;not null;index:idx_org_sync_member_units_unit;uniqueIndex:uq_org_sync_member_units_identity,priority:4" json:"source_unit_uuid"`
	ExternalMemberID     string    `gorm:"column:external_member_id;type:text;not null;index:idx_org_sync_member_units_ext_member" json:"external_member_id"`
	ExternalUnitID       string    `gorm:"column:external_unit_id;type:text;not null;index:idx_org_sync_member_units_ext_unit" json:"external_unit_id"`
	Order                int       `gorm:"column:order;type:int;not null;default:0" json:"order"`
	CreatedAt            time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SourceMemberUnit) TableName() string {
	return models.S(models.TableOrgSyncSourceMemberUnits)
}
