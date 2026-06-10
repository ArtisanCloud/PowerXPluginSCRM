package opportunity

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
)

type OpportunityLineItem struct {
	ItemUUID        string    `gorm:"column:item_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"item_uuid"`
	TenantUUID      string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_opp_line_tenant_opp,priority:1" json:"tenant_uuid"`
	OpportunityUUID string    `gorm:"column:opportunity_uuid;type:uuid;not null;index:idx_opp_line_tenant_opp,priority:2" json:"opportunity_uuid"`
	Kind            string    `gorm:"column:kind;type:varchar(24);not null;default:'manual'" json:"kind"`
	Name            string    `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Quantity        float64   `gorm:"column:quantity;type:numeric(18,2);not null;default:1" json:"quantity"`
	UnitPrice       float64   `gorm:"column:unit_price;type:numeric(18,2);not null;default:0" json:"unit_price"`
	TotalAmount     float64   `gorm:"column:total_amount;type:numeric(18,2);not null;default:0" json:"total_amount"`
	Currency        string    `gorm:"column:currency;type:varchar(8);not null;default:'CNY'" json:"currency"`
	StorageProvider string    `gorm:"column:storage_provider;type:varchar(32)" json:"storage_provider,omitempty"`
	ObjectKey       string    `gorm:"column:object_key;type:text" json:"object_key,omitempty"`
	FileName        string    `gorm:"column:file_name;type:text" json:"file_name,omitempty"`
	FileSize        int64     `gorm:"column:file_size;type:bigint;not null;default:0" json:"file_size"`
	ContentType     string    `gorm:"column:content_type;type:varchar(128)" json:"content_type,omitempty"`
	DownloadURL     string    `gorm:"-" json:"download_url,omitempty"`
	CreatedBy       string    `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy       string    `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (OpportunityLineItem) TableName() string {
	return domainmodels.S(domainmodels.TableOpportunityLineItems)
}
