package marketplace

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	LicenseStatusTrial     = "trial"
	LicenseStatusActive    = "active"
	LicenseStatusExpired   = "expired"
	LicenseStatusRevoked   = "revoked"
	LicenseStatusSuspended = "suspended"

	LicenseEventIssued        = "issued"
	LicenseEventRenewed       = "renewed"
	LicenseEventRevoked       = "revoked"
	LicenseEventUsageReported = "usage_reported"
	LicenseEventTrialExpired  = "trial_expired"
	LicenseEventOfflineExtend = "offline_extend"
)

// License represents an active authorization granted to a tenant.
type License struct {
	ID              string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid      string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	ListingID       string            `gorm:"column:listing_id;type:uuid;not null;index" json:"listing_id"`
	PlanID          string            `gorm:"column:plan_id;type:uuid;not null;index" json:"plan_id"`
	LicenseToken    string            `gorm:"column:license_token;type:text;not null" json:"license_token"`
	Status          string            `gorm:"column:status;type:text;not null;default:'active'" json:"status"`
	IssuedAt        time.Time         `gorm:"column:issued_at;type:timestamptz;not null" json:"issued_at"`
	ExpiresAt       time.Time         `gorm:"column:expires_at;type:timestamptz;not null" json:"expires_at"`
	RenewalToken    *string           `gorm:"column:renewal_token;type:text" json:"renewal_token,omitempty"`
	OfflineUntil    *time.Time        `gorm:"column:offline_until;type:timestamptz" json:"offline_until,omitempty"`
	LastValidatedAt *time.Time        `gorm:"column:last_validated_at;type:timestamptz" json:"last_validated_at,omitempty"`
	IssuedBy        *string           `gorm:"column:issued_by;type:text" json:"issued_by,omitempty"`
	Metadata        datatypes.JSONMap `gorm:"column:metadata;type:jsonb" json:"metadata"`
	CreatedAt       time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	Events          []LicenseEvent    `gorm:"foreignKey:LicenseID" json:"events,omitempty"`
}

func (*License) TableName() string {
	return models.S(models.TableMarketplaceLicenses)
}

// LicenseEvent records significant lifecycle actions for a license.
type LicenseEvent struct {
	ID           string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid   string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	LicenseID    string            `gorm:"column:license_id;type:uuid;not null;index" json:"license_id"`
	EventType    string            `gorm:"column:event_type;type:text;not null" json:"event_type"`
	EventPayload datatypes.JSONMap `gorm:"column:event_payload;type:jsonb" json:"event_payload"`
	EmittedAt    time.Time         `gorm:"column:emitted_at;type:timestamptz;not null" json:"emitted_at"`
	ActorID      *string           `gorm:"column:actor_id;type:text" json:"actor_id,omitempty"`
	TraceID      *string           `gorm:"column:trace_id;type:text" json:"trace_id,omitempty"`
	CreatedAt    time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

func (*LicenseEvent) TableName() string {
	return models.S(models.TableMarketplaceLicenseEvents)
}

// TaxTransaction stores interactions with external tax providers.
type TaxTransaction struct {
	ID                    string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid            string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	BillingID             string            `gorm:"column:billing_id;type:text;not null" json:"billing_id"`
	ExternalProvider      string            `gorm:"column:external_provider;type:text;not null" json:"external_provider"`
	ExternalTransactionID *string           `gorm:"column:external_transaction_id;type:text" json:"external_transaction_id,omitempty"`
	Jurisdiction          string            `gorm:"column:jurisdiction;type:text" json:"jurisdiction,omitempty"`
	TaxAmount             float64           `gorm:"column:tax_amount;type:numeric(18,4);not null" json:"tax_amount"`
	Currency              string            `gorm:"column:currency;type:text;not null" json:"currency"`
	SettlementCurrency    string            `gorm:"column:settlement_currency;type:text" json:"settlement_currency,omitempty"`
	ExchangeRate          *float64          `gorm:"column:exchange_rate;type:numeric(18,6)" json:"exchange_rate,omitempty"`
	TaxAmountSettlement   *float64          `gorm:"column:tax_amount_settlement;type:numeric(18,4)" json:"tax_amount_settlement,omitempty"`
	RawPayload            datatypes.JSONMap `gorm:"column:raw_payload;type:jsonb" json:"raw_payload"`
	Status                string            `gorm:"column:status;type:text;not null;default:'pending'" json:"status"`
	SyncedAt              *time.Time        `gorm:"column:synced_at;type:timestamptz" json:"synced_at,omitempty"`
	CreatedAt             time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (*TaxTransaction) TableName() string {
	return models.S(models.TableMarketplaceTaxTransactions)
}
