package operations

import (
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// SupportChannel represents a configured support channel for a plugin or tenant scope.
type SupportChannel struct {
	ID             string            `gorm:"primaryKey;type:uuid" json:"id"`
	PluginID       string            `gorm:"column:plugin_id;index:idx_support_channels_scope" json:"plugin_id"`
	TenantUuid     *string           `gorm:"column:tenant_uuid;index:idx_support_channels_scope" json:"tenant_uuid,omitempty"`
	Channel        string            `gorm:"column:channel" json:"channel"`
	IsEnabled      bool              `gorm:"column:is_enabled" json:"is_enabled"`
	ServiceWindow  datatypes.JSONMap `gorm:"column:service_window" json:"service_window"`
	EscalationPath datatypes.JSONMap `gorm:"column:escalation_path" json:"escalation_path"`
	Metadata       datatypes.JSONMap `gorm:"column:metadata" json:"metadata"`
	Version        int               `gorm:"column:version" json:"version"`
	CreatedAt      time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

// TableName implements gorm tablename interface.
func (SupportChannel) TableName() string {
	return basemodels.S(basemodels.TableOperationsSupportChannels)
}
