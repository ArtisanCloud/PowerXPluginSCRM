package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

type LeadConversationPending struct {
	PendingUUID        string     `gorm:"column:pending_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"pending_uuid"`
	TenantUUID         string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_conv_pending_tenant" json:"tenant_uuid"`
	EventUUID          string     `gorm:"column:event_uuid;type:uuid;not null;uniqueIndex:uq_lead_capture_conv_pending_event" json:"event_uuid"`
	ChannelAccountUUID string     `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_lead_capture_conv_pending_account" json:"channel_account_uuid"`
	ConversationID     string     `gorm:"column:conversation_id;type:text;not null;index:idx_lead_capture_conv_pending_conversation" json:"conversation_id"`
	Reason             string     `gorm:"column:reason;type:varchar(32);not null;default:'no_match'" json:"reason"`
	Status             string     `gorm:"column:status;type:varchar(16);not null;default:'pending'" json:"status"`
	CreatedAt          time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	ResolvedAt         *time.Time `gorm:"column:resolved_at;type:timestamptz" json:"resolved_at,omitempty"`
}

func (LeadConversationPending) TableName() string {
	return models.S(models.TableLeadCaptureConversationPending)
}
