package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

type ConversationEvent struct {
	EventUUID          string            `gorm:"column:event_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"event_uuid"`
	TenantUUID         string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_conv_events_tenant" json:"tenant_uuid"`
	Channel            string            `gorm:"column:channel;type:varchar(32);not null;default:'wechat'" json:"channel"`
	AppType            string            `gorm:"column:app_type;type:varchar(32);not null;default:'wecom'" json:"app_type"`
	ChannelAccountUUID string            `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_lead_capture_conv_events_account" json:"channel_account_uuid"`
	ExternalEventID    string            `gorm:"column:external_event_id;type:text;not null" json:"external_event_id"`
	IdempotencyKey     string            `gorm:"column:idempotency_key;type:text;not null;uniqueIndex:uq_lead_capture_conv_events_idempotency" json:"idempotency_key"`
	ConversationID     string            `gorm:"column:conversation_id;type:text;not null;index:idx_lead_capture_conv_events_conversation" json:"conversation_id"`
	ActorType          string            `gorm:"column:actor_type;type:varchar(32);not null" json:"actor_type"`
	ActorID            string            `gorm:"column:actor_id;type:text;not null" json:"actor_id"`
	Direction          string            `gorm:"column:direction;type:varchar(16);not null" json:"direction"`
	MessageType        string            `gorm:"column:message_type;type:varchar(32);not null" json:"message_type"`
	ContentText        string            `gorm:"column:content_text;type:text" json:"content_text"`
	RawPayload         datatypes.JSONMap `gorm:"column:raw_payload;type:jsonb;default:'{}'::jsonb" json:"raw_payload"`
	OccurredAt         time.Time         `gorm:"column:occurred_at;type:timestamptz;not null;index:idx_lead_capture_conv_events_occurred" json:"occurred_at"`
	CreatedAt          time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

func (ConversationEvent) TableName() string {
	return models.S(models.TableLeadCaptureConversationEvents)
}
