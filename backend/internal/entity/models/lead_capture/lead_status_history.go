package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// LeadStatusHistory captures status transitions for a lead.
type LeadStatusHistory struct {
	HistoryUUID string    `gorm:"column:history_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"history_uuid"`
	TenantUUID  string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_status_history_tenant" json:"tenant_uuid"`
	LeadUUID    string    `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_capture_status_history_lead" json:"lead_uuid"`
	FromStatus  string    `gorm:"column:from_status;type:varchar(32);not null" json:"from_status"`
	ToStatus    string    `gorm:"column:to_status;type:varchar(32);not null;index:idx_lead_capture_status_history_to" json:"to_status"`
	ChangedAt   time.Time `gorm:"column:changed_at;type:timestamptz;autoCreateTime" json:"changed_at"`
}

func (LeadStatusHistory) TableName() string {
	return models.S(models.TableLeadCaptureStatusHistory)
}
