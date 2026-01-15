package integration

import (
	"time"

	models "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	// WebhookStatusActive 表示订阅生效。
	WebhookStatusActive = "ACTIVE"
	// WebhookStatusPaused 表示暂时暂停投递。
	WebhookStatusPaused = "PAUSED"
	// WebhookStatusDisabled 表示订阅被禁用。
	WebhookStatusDisabled = "DISABLED"
)

// WebhookSubscription 描述租户订阅某类事件的配置。
type WebhookSubscription struct {
	ID          string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid  string            `gorm:"column:tenant_uuid;type:uuid;not null" json:"tenant_uuid"`
	EventType   string            `gorm:"column:event_type;type:text;not null" json:"event_type"`
	TargetURL   string            `gorm:"column:target_url;type:text;not null" json:"target_url"`
	Secret      string            `gorm:"column:secret;type:text" json:"-"`
	RetryPolicy datatypes.JSON    `gorm:"column:retry_policy;type:jsonb" json:"retry_policy,omitempty"`
	Status      string            `gorm:"column:status;type:text;not null;default:'ACTIVE'" json:"status"`
	Metadata    datatypes.JSONMap `gorm:"column:metadata;type:jsonb;default:'{}'::jsonb" json:"metadata"`
	CreatedAt   time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName returns the underlying table name.
func (WebhookSubscription) TableName() string {
	return models.S(models.TableIntegrationWebhookSubscriptions)
}
