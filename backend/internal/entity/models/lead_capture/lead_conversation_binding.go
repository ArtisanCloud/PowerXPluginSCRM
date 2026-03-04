package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

type LeadConversationBinding struct {
	BindingUUID        string    `gorm:"column:binding_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"binding_uuid"`
	TenantUUID         string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_conv_bindings_tenant;uniqueIndex:uq_lead_capture_conv_bindings_active,priority:1" json:"tenant_uuid"`
	LeadUUID           string    `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_capture_conv_bindings_lead" json:"lead_uuid"`
	ConversationID     string    `gorm:"column:conversation_id;type:text;not null;index:idx_lead_capture_conv_bindings_conversation;uniqueIndex:uq_lead_capture_conv_bindings_active,priority:2" json:"conversation_id"`
	ChannelAccountUUID string    `gorm:"column:channel_account_uuid;type:uuid;not null;index:idx_lead_capture_conv_bindings_account" json:"channel_account_uuid"`
	BindSource         string    `gorm:"column:bind_source;type:varchar(16);not null;default:'auto'" json:"bind_source"`
	Status             string    `gorm:"column:status;type:varchar(16);not null;default:'active';index:idx_lead_capture_conv_bindings_status;uniqueIndex:uq_lead_capture_conv_bindings_active,priority:3" json:"status"`
	CreatedBy          string    `gorm:"column:created_by;type:text" json:"created_by"`
	CreatedAt          time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadConversationBinding) TableName() string {
	return models.S(models.TableLeadCaptureConversationBindings)
}
