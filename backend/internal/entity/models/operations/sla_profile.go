package operations

import (
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// SLAProfile captures SLA targets and actual performance for a plan type.
type SLAProfile struct {
	ID                    string     `gorm:"primaryKey;type:uuid" json:"id"`
	PluginID              string     `gorm:"column:plugin_id;index:idx_sla_profiles_plugin" json:"plugin_id"`
	PlanType              string     `gorm:"column:plan_type" json:"plan_type"`
	UptimeTarget          float64    `gorm:"column:uptime_target" json:"uptime_target"`
	UptimeActual          float64    `gorm:"column:uptime_actual" json:"uptime_actual"`
	ResponseTargetMs      int32      `gorm:"column:response_target_ms" json:"response_target_ms"`
	ResponseActualMs      int32      `gorm:"column:response_actual_ms" json:"response_actual_ms"`
	SuccessTargetPct      float64    `gorm:"column:success_target_pct" json:"success_target_pct"`
	SuccessActualPct      float64    `gorm:"column:success_actual_pct" json:"success_actual_pct"`
	SupportFrtTargetHours float64    `gorm:"column:support_frt_target_hours" json:"support_frt_target_hours"`
	SupportFrtActualHours float64    `gorm:"column:support_frt_actual_hours" json:"support_frt_actual_hours"`
	SLAScore              float64    `gorm:"column:sla_score" json:"sla_score"`
	IncentiveAppliedAt    *time.Time `gorm:"column:incentive_applied_at" json:"incentive_applied_at,omitempty"`
	PenaltyAppliedAt      *time.Time `gorm:"column:penalty_applied_at" json:"penalty_applied_at,omitempty"`
	Notes                 string     `gorm:"column:notes" json:"notes"`
	ComputedAt            time.Time  `gorm:"column:computed_at" json:"computed_at"`
	CreatedAt             time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName specifies the SLA profiles table name.
func (SLAProfile) TableName() string {
	return basemodels.S(basemodels.TableOperationsSLAScores)
}

// SLAAdjustment records incentive or penalty actions tied to SLA score thresholds.
type SLAAdjustment struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PluginID    string    `gorm:"column:plugin_id" json:"plugin_id"`
	PlanType    string    `gorm:"column:plan_type" json:"plan_type"`
	PeriodStart time.Time `gorm:"column:period_start" json:"period_start"`
	PeriodEnd   time.Time `gorm:"column:period_end" json:"period_end"`
	ScoreBefore float64   `gorm:"column:score_before" json:"score_before"`
	ScoreAfter  float64   `gorm:"column:score_after" json:"score_after"`
	Action      string    `gorm:"column:action" json:"action"`
	Details     string    `gorm:"column:details" json:"details"`
	AppliedBy   *string   `gorm:"column:applied_by" json:"applied_by,omitempty"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
}

// TableName returns the SLA adjustments table name.
func (SLAAdjustment) TableName() string {
	return basemodels.S(basemodels.TableOperationsSLAAdjustments)
}
