package runtime_ops

import "time"

// QuotaLedger records aggregated usage for a specific scope window.
type QuotaLedger struct {
	ID              string     `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ScopeType       string     `gorm:"column:scope_type;type:text;not null" json:"scope_type"`
	ScopeRef        string     `gorm:"column:scope_ref;type:text;not null" json:"scope_ref"`
	WindowStart     time.Time  `gorm:"column:window_start;type:timestamptz;not null" json:"window_start"`
	WindowEnd       time.Time  `gorm:"column:window_end;type:timestamptz;not null" json:"window_end"`
	TokensConsumed  float64    `gorm:"column:tokens_consumed;type:numeric;default:0" json:"tokens_consumed"`
	CPUSeconds      float64    `gorm:"column:cpu_seconds;type:numeric;default:0" json:"cpu_seconds"`
	BandwidthMB     float64    `gorm:"column:bandwidth_mb;type:numeric;default:0" json:"bandwidth_mb"`
	Invocations     float64    `gorm:"column:invocations;type:numeric;default:0" json:"invocations"`
	OverLimitAction string     `gorm:"column:over_limit_action;type:text" json:"over_limit_action,omitempty"`
	ReportedAt      *time.Time `gorm:"column:reported_at;type:timestamptz" json:"reported_at,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}
