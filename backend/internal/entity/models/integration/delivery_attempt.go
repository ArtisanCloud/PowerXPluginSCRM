package integration

import (
	"time"

	models "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	AttemptStatusPending   = "PENDING"
	AttemptStatusRetrying  = "RETRYING"
	AttemptStatusSucceeded = "SUCCEEDED"
	AttemptStatusFailed    = "FAILED"
	AttemptStatusDLQ       = "DLQ"
)

// DeliveryAttempt 记录每一次 webhook 发送的状态与重试信息。
type DeliveryAttempt struct {
	ID              string         `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	SubscriptionID  string         `gorm:"column:subscription_id;type:uuid;not null" json:"subscription_id"`
	EnvelopeID      string         `gorm:"column:envelope_id;type:uuid" json:"envelope_id,omitempty"`
	Status          string         `gorm:"column:status;type:text;not null" json:"status"`
	RetryCount      int            `gorm:"column:retry_count;type:int;not null;default:0" json:"retry_count"`
	LastError       string         `gorm:"column:last_error;type:text" json:"last_error,omitempty"`
	NextDeliveryAt  *time.Time     `gorm:"column:next_delivery_at;type:timestamptz" json:"next_delivery_at,omitempty"`
	PayloadSnapshot datatypes.JSON `gorm:"column:payload_snapshot;type:jsonb" json:"payload_snapshot,omitempty"`
	CreatedAt       time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for DeliveryAttempt.
func (DeliveryAttempt) TableName() string {
	return models.S(models.TableIntegrationWebhookAttempts)
}
