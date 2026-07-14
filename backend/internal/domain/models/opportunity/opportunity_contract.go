package opportunity

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
)

type OpportunityContract struct {
	ContractUUID    string     `gorm:"column:contract_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"contract_uuid"`
	TenantUUID      string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_opp_contract_tenant_opp,priority:1" json:"tenant_uuid"`
	OpportunityUUID string     `gorm:"column:opportunity_uuid;type:uuid;not null;index:idx_opp_contract_tenant_opp,priority:2" json:"opportunity_uuid"`
	CustomerUUID    string     `gorm:"column:customer_uuid;type:uuid" json:"customer_uuid,omitempty"`
	QuoteItemUUID   string     `gorm:"column:quote_item_uuid;type:uuid" json:"quote_item_uuid,omitempty"`
	ContractNo      string     `gorm:"column:contract_no;type:varchar(128);not null" json:"contract_no"`
	Title           string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Amount          float64    `gorm:"column:amount;type:numeric(18,2);not null;default:0" json:"amount"`
	Currency        string     `gorm:"column:currency;type:varchar(8);not null;default:'CNY'" json:"currency"`
	Status          string     `gorm:"column:status;type:varchar(32);not null;default:'draft'" json:"status"`
	SignedAt        *time.Time `gorm:"column:signed_at;type:timestamptz" json:"signed_at,omitempty"`
	StorageProvider string     `gorm:"column:storage_provider;type:varchar(32)" json:"storage_provider,omitempty"`
	ObjectKey       string     `gorm:"column:object_key;type:text" json:"object_key,omitempty"`
	FileName        string     `gorm:"column:file_name;type:text" json:"file_name,omitempty"`
	FileSize        int64      `gorm:"column:file_size;type:bigint;not null;default:0" json:"file_size"`
	ContentType     string     `gorm:"column:content_type;type:varchar(128)" json:"content_type,omitempty"`
	DownloadURL     string     `gorm:"-" json:"download_url,omitempty"`
	CreatedBy       string     `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy       string     `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (OpportunityContract) TableName() string {
	return domainmodels.S(domainmodels.TableOpportunityContracts)
}
