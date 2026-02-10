package org_sync

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// SourceUnit represents an org unit from a channel account.
type SourceUnit struct {
	SourceUnitUUID       string    `gorm:"column:source_unit_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"source_unit_uuid"`
	TenantUUID           string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_org_sync_source_units_tenant;uniqueIndex:uq_org_sync_source_units_identity,priority:1" json:"tenant_uuid"`
	SourceAccountUUID    string    `gorm:"column:source_account_uuid;type:uuid;not null;index:idx_org_sync_source_units_account;uniqueIndex:uq_org_sync_source_units_identity,priority:2" json:"source_account_uuid"`
	ChannelAccountUUID   string    `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_org_sync_source_units_channel" json:"channel_account_uuid"`
	ExternalUnitID       string    `gorm:"column:external_unit_id;type:text;not null;uniqueIndex:uq_org_sync_source_units_identity,priority:3" json:"external_unit_id"`
	ParentExternalUnitID *string   `gorm:"column:parent_external_unit_id;type:text" json:"parent_external_unit_id,omitempty"`
	Name                 string    `gorm:"column:name;type:text;not null" json:"name"`
	Order                int       `gorm:"column:order;type:int;not null;default:0" json:"order"`
	Status               string    `gorm:"column:status;type:varchar(32);not null;default:'active';index:idx_org_sync_source_units_status" json:"status"`
	CreatedAt            time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (SourceUnit) TableName() string {
	return models.S(models.TableOrgSyncSourceUnits)
}
