package org_sync

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// UnitMapping links a source unit to a main org unit.
type UnitMapping struct {
	UnitMappingUUID string     `gorm:"column:unit_mapping_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"unit_mapping_uuid"`
	TenantUUID      string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_org_sync_unit_mappings_tenant;uniqueIndex:uq_org_sync_unit_mappings_source,priority:1" json:"tenant_uuid"`
	SourceUnitUUID  string     `gorm:"column:source_unit_uuid;type:uuid;not null;uniqueIndex:uq_org_sync_unit_mappings_source,priority:2" json:"source_unit_uuid"`
	MainUnitID      string     `gorm:"column:main_unit_id;type:text;not null;index:idx_org_sync_unit_mappings_main" json:"main_unit_id"`
	MappingStatus   string     `gorm:"column:mapping_status;type:varchar(32);not null;default:'pending';index:idx_org_sync_unit_mappings_status" json:"mapping_status"`
	ConfirmedBy     string     `gorm:"column:confirmed_by;type:text" json:"confirmed_by,omitempty"`
	ConfirmedAt     *time.Time `gorm:"column:confirmed_at;type:timestamptz" json:"confirmed_at,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (UnitMapping) TableName() string {
	return models.S(models.TableOrgSyncUnitMappings)
}
