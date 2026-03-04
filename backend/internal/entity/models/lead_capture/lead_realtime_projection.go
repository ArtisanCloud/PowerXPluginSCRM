package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

type LeadRealtimeProjection struct {
	ProjectionUUID  string    `gorm:"column:projection_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"projection_uuid"`
	TenantUUID      string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_realtime_tenant;uniqueIndex:uq_lead_capture_realtime_projection,priority:1" json:"tenant_uuid"`
	LeadUUID        string    `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_capture_realtime_lead;uniqueIndex:uq_lead_capture_realtime_projection,priority:2" json:"lead_uuid"`
	ConversationID  string    `gorm:"column:conversation_id;type:text;not null;index:idx_lead_capture_realtime_conversation;uniqueIndex:uq_lead_capture_realtime_projection,priority:3" json:"conversation_id"`
	LatestMessage   string    `gorm:"column:latest_message;type:text" json:"latest_message"`
	LatestActorType string    `gorm:"column:latest_actor_type;type:varchar(32)" json:"latest_actor_type"`
	LatestAt        time.Time `gorm:"column:latest_at;type:timestamptz" json:"latest_at"`
	UnreadCount     int       `gorm:"column:unread_count;type:int;not null;default:0" json:"unread_count"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

func (LeadRealtimeProjection) TableName() string {
	return models.S(models.TableLeadCaptureRealtimeProjection)
}
