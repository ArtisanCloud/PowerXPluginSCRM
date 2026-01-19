package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	LeadEventTypeIntake       = "intake"
	LeadEventTypeMerge        = "merge"
	LeadEventTypeStatusChange = "status_change"
	LeadEventTypeAssign       = "assign"
)

// LeadEvent records lead lifecycle events.
type LeadEvent struct {
	EventUUID  string            `gorm:"column:event_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"event_uuid"`
	LeadUUID   string            `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_capture_events_lead" json:"lead_uuid"`
	TenantUUID string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_events_tenant" json:"tenant_uuid"`
	EventType  string            `gorm:"column:event_type;type:varchar(64);not null" json:"event_type"`
	Payload    datatypes.JSONMap `gorm:"column:payload;type:jsonb;default:'{}'::jsonb" json:"payload"`
	CreatedAt  time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadEvent) TableName() string {
	return models.S(models.TableLeadCaptureEvents)
}
