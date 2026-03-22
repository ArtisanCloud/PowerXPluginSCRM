package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	LeadActivityTypeIntake       = "intake"
	LeadActivityTypeMerge        = "merge"
	LeadActivityTypeStatusChange = "status_change"
	LeadActivityTypeAssign       = "assign"
	LeadActivityTypeBotCommand   = "bot_command"
	LeadActivityTypeSyncTrace    = "sync_trace"
)

// LeadActivity records lead lifecycle actions for audit tracking.
type LeadActivity struct {
	ActivityUUID string            `gorm:"column:activity_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"activity_uuid"`
	LeadUUID     string            `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_capture_activities_lead" json:"lead_uuid"`
	TenantUUID   string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_activities_tenant" json:"tenant_uuid"`
	ActivityType string            `gorm:"column:activity_type;type:varchar(64);not null" json:"activity_type"`
	Payload      datatypes.JSONMap `gorm:"column:payload;type:jsonb;default:'{}'::jsonb" json:"payload"`
	CreatedAt    time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadActivity) TableName() string {
	return models.S(models.TableLeadCaptureActivities)
}
