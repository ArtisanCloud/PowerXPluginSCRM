package social_channel_governance

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ChannelPlatformSetting stores plugin-level platform configuration shared by all tenants.
type ChannelPlatformSetting struct {
	SettingUUID  string            `gorm:"column:setting_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"setting_uuid"`
	ChannelCode  string            `gorm:"column:channel_code;type:varchar(32);not null;uniqueIndex:uq_social_channel_platform_settings,priority:1" json:"channel_code"`
	ProviderCode string            `gorm:"column:provider_code;type:varchar(64);not null;uniqueIndex:uq_social_channel_platform_settings,priority:2" json:"provider_code"`
	Enabled      bool              `gorm:"column:enabled;type:boolean;not null;default:false" json:"enabled"`
	Config       datatypes.JSONMap `gorm:"column:config;type:jsonb;default:'{}'::jsonb" json:"config"`
	CreatedAt    time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt    `gorm:"column:deleted_at;type:timestamptz" json:"deleted_at,omitempty"`
}

func (ChannelPlatformSetting) TableName() string {
	return models.S(models.TableSocialChannelPlatformSettings)
}
