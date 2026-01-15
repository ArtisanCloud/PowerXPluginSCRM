package runtime_ops

import "time"

// MarketplaceOverage captures hourly summaries for reporting to Marketplace.
type MarketplaceOverage struct {
	ID           string     `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PluginID     string     `gorm:"column:plugin_id;type:text;not null" json:"plugin_id"`
	TenantUuid   string     `gorm:"column:tenant_uuid;type:uuid" json:"tenant_uuid,omitempty"`
	HourWindow   time.Time  `gorm:"column:hour_window;type:timestamptz;not null" json:"hour_window"`
	QuotaMetric  string     `gorm:"column:quota_metric;type:text;not null" json:"quota_metric"`
	BreachCount  int        `gorm:"column:breach_count;type:int;not null;default:0" json:"breach_count"`
	LastBreachAt *time.Time `gorm:"column:last_breach_at;type:timestamptz" json:"last_breach_at,omitempty"`
	Reported     bool       `gorm:"column:reported;type:boolean;not null;default:false" json:"reported"`
	CreatedAt    time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}
