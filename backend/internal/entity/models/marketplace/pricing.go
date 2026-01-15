package marketplace

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	PricingPlanTypeFree         = "free"
	PricingPlanTypeOneTime      = "one_time"
	PricingPlanTypeSubscription = "subscription"
	PricingPlanTypeUsage        = "usage"
)

// PricingPlan describes pricing availability for a listing.
type PricingPlan struct {
	ID              string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ListingID       string            `gorm:"column:listing_id;type:uuid;not null;index" json:"listing_id"`
	TenantUuid      string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	PlanCode        string            `gorm:"column:plan_code;type:text;not null" json:"plan_code"`
	PlanType        string            `gorm:"column:plan_type;type:text;not null" json:"plan_type"`
	Currency        string            `gorm:"column:currency;type:text;not null" json:"currency"`
	Amount          *float64          `gorm:"column:amount;type:numeric(18,4)" json:"amount,omitempty"`
	BillingPeriod   string            `gorm:"column:billing_period;type:text" json:"billing_period,omitempty"`
	TrialPeriodDays *int              `gorm:"column:trial_period_days;type:int" json:"trial_period_days,omitempty"`
	QuotaLimit      *float64          `gorm:"column:quota_limit;type:numeric(18,4)" json:"quota_limit,omitempty"`
	OveragePolicy   string            `gorm:"column:overage_policy;type:text" json:"overage_policy,omitempty"`
	FeatureMatrix   datatypes.JSONMap `gorm:"column:feature_matrix;type:jsonb" json:"feature_matrix"`
	IsDefault       bool              `gorm:"column:is_default;type:boolean;not null;default:false" json:"is_default"`
	Status          string            `gorm:"column:status;type:text;not null;default:'active'" json:"status"`
	CreatedAt       time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	Tiers           []PlanTier        `gorm:"foreignKey:PlanID" json:"tiers,omitempty"`
}

func (*PricingPlan) TableName() string {
	return models.S(models.TableMarketplacePricingPlans)
}

// PlanTier describes a usage based pricing tier.
type PlanTier struct {
	ID         string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PlanID     string    `gorm:"column:plan_id;type:uuid;not null;index" json:"plan_id"`
	TenantUuid string    `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	Metric     string    `gorm:"column:metric;type:text;not null" json:"metric"`
	RangeFrom  float64   `gorm:"column:range_from;type:numeric(18,4);not null" json:"range_from"`
	RangeTo    *float64  `gorm:"column:range_to;type:numeric(18,4)" json:"range_to,omitempty"`
	UnitAmount float64   `gorm:"column:unit_amount;type:numeric(18,4);not null" json:"unit_amount"`
	UnitName   string    `gorm:"column:unit_name;type:text" json:"unit_name,omitempty"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (*PlanTier) TableName() string {
	return models.S(models.TableMarketplacePlanTiers)
}
