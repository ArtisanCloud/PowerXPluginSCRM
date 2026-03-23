package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

const (
	SourceCatalogCategoryTrafficPlatform = "traffic_platform"
	SourceCatalogCategoryTrafficSource   = "traffic_source"
)

// LeadSourceCatalog stores tenant-scoped dictionary items for lead traffic attribution.
type LeadSourceCatalog struct {
	CatalogUUID string    `gorm:"column:catalog_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"catalog_uuid"`
	TenantUUID  string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_source_catalogs_tenant;uniqueIndex:uq_lead_capture_source_catalogs_tenant_category_code,priority:1" json:"tenant_uuid"`
	Category    string    `gorm:"column:category;type:varchar(128);not null;index:idx_lead_capture_source_catalogs_category;uniqueIndex:uq_lead_capture_source_catalogs_tenant_category_code,priority:2" json:"category"`
	Code        string    `gorm:"column:code;type:varchar(64);not null;uniqueIndex:uq_lead_capture_source_catalogs_tenant_category_code,priority:3" json:"code"`
	Label       string    `gorm:"column:label;type:varchar(128);not null" json:"label"`
	Sort        int       `gorm:"column:sort;type:int;not null;default:100" json:"sort"`
	Enabled     bool      `gorm:"column:enabled;type:boolean;not null;default:true;index:idx_lead_capture_source_catalogs_enabled" json:"enabled"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadSourceCatalog) TableName() string {
	return models.S(models.TableLeadCaptureSourceCatalogs)
}
