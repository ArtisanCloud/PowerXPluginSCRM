package marketplace

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	NotificationStatusPending = "pending"
	NotificationStatusSent    = "sent"
	NotificationStatusFailed  = "failed"
)

// Notification represents marketplace-scoped communication entries (email/webhook/in-app).
type Notification struct {
	ID            string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid    string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	RecipientType string            `gorm:"column:recipient_type;type:text;not null" json:"recipient_type"`
	RecipientID   string            `gorm:"column:recipient_id;type:text;not null" json:"recipient_id"`
	Channel       string            `gorm:"column:channel;type:text;not null" json:"channel"`
	TemplateCode  string            `gorm:"column:template_code;type:text;not null" json:"template_code"`
	Payload       datatypes.JSONMap `gorm:"column:payload;type:jsonb" json:"payload"`
	ScheduledAt   *time.Time        `gorm:"column:scheduled_at;type:timestamptz" json:"scheduled_at,omitempty"`
	SentAt        *time.Time        `gorm:"column:sent_at;type:timestamptz" json:"sent_at,omitempty"`
	Status        string            `gorm:"column:status;type:text;not null;default:'pending'" json:"status"`
	CreatedAt     time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName implements gorm tabler.
func (*Notification) TableName() string {
	return models.S(models.TableMarketplaceNotifications)
}
