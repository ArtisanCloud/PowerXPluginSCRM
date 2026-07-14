package opportunity

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
)

type OpportunityPayment struct {
	PaymentUUID     string     `gorm:"column:payment_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"payment_uuid"`
	TenantUUID      string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_opp_payment_tenant_opp,priority:1" json:"tenant_uuid"`
	OpportunityUUID string     `gorm:"column:opportunity_uuid;type:uuid;not null;index:idx_opp_payment_tenant_opp,priority:2" json:"opportunity_uuid"`
	ContractUUID    string     `gorm:"column:contract_uuid;type:uuid;not null;index:idx_opp_payment_contract" json:"contract_uuid"`
	Title           string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	PlannedAmount   float64    `gorm:"column:planned_amount;type:numeric(18,2);not null;default:0" json:"planned_amount"`
	PaidAmount      float64    `gorm:"column:paid_amount;type:numeric(18,2);not null;default:0" json:"paid_amount"`
	Currency        string     `gorm:"column:currency;type:varchar(8);not null;default:'CNY'" json:"currency"`
	Status          string     `gorm:"column:status;type:varchar(24);not null;default:'planned'" json:"status"`
	DueAt           *time.Time `gorm:"column:due_at;type:timestamptz" json:"due_at,omitempty"`
	PaidAt          *time.Time `gorm:"column:paid_at;type:timestamptz" json:"paid_at,omitempty"`
	Method          string     `gorm:"column:method;type:varchar(64)" json:"method,omitempty"`
	TransactionNo   string     `gorm:"column:transaction_no;type:varchar(128)" json:"transaction_no,omitempty"`
	Note            string     `gorm:"column:note;type:text" json:"note,omitempty"`
	CreatedBy       string     `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy       string     `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (OpportunityPayment) TableName() string {
	return domainmodels.S(domainmodels.TableOpportunityPayments)
}
