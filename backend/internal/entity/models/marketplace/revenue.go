package marketplace

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

const (
	RevenueReportStatusDraft      = "draft"
	RevenueReportStatusReady      = "ready"
	RevenueReportStatusExported   = "exported"
	RevenueReportStatusReconciled = "reconciled"
)

// RevenueShareReport captures vendor/platform revenue distribution for a period.
type RevenueShareReport struct {
	ID            string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid    string    `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	VendorID      string    `gorm:"column:vendor_id;type:text;not null;index" json:"vendor_id"`
	PeriodStart   time.Time `gorm:"column:period_start;type:timestamptz;not null" json:"period_start"`
	PeriodEnd     time.Time `gorm:"column:period_end;type:timestamptz;not null" json:"period_end"`
	GrossAmount   float64   `gorm:"column:gross_amount;type:numeric(18,4);not null" json:"gross_amount"`
	VendorShare   float64   `gorm:"column:vendor_share;type:numeric(18,4);not null" json:"vendor_share"`
	PlatformShare float64   `gorm:"column:platform_share;type:numeric(18,4);not null" json:"platform_share"`
	Fees          float64   `gorm:"column:fees;type:numeric(18,4);not null" json:"fees"`
	Currency      string    `gorm:"column:currency;type:text;not null" json:"currency"`
	Status        string    `gorm:"column:status;type:text;not null;default:'draft'" json:"status"`
	GeneratedAt   time.Time `gorm:"column:generated_at;type:timestamptz;autoCreateTime" json:"generated_at"`
	ExportURI     *string   `gorm:"column:export_uri;type:text" json:"export_uri,omitempty"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName implements gorm tabler.
func (*RevenueShareReport) TableName() string {
	return models.S(models.TableMarketplaceRevenueReports)
}
