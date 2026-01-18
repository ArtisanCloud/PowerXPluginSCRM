package social_channel_governance

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ChannelAccountStatusPending   = "pending"
	ChannelAccountStatusConnected = "connected"
	ChannelAccountStatusExpired   = "expired"
	ChannelAccountStatusDisabled  = "disabled"
)

// ChannelAccount stores governance settings for a connected channel account.
type ChannelAccount struct {
	AccountUUID     string            `gorm:"column:account_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"account_uuid"`
	TenantUuid      string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_channel_accounts_tenant;uniqueIndex:uq_social_channel_accounts_identity,priority:1" json:"tenant_uuid"`
	ChannelCode     string            `gorm:"column:channel_code;type:varchar(32);not null;uniqueIndex:uq_social_channel_accounts_identity,priority:2" json:"channel_code"`
	AppType         string            `gorm:"column:app_type;type:varchar(32);not null;uniqueIndex:uq_social_channel_accounts_identity,priority:3" json:"app_type"`
	AccountID       string            `gorm:"column:account_id;type:text;not null;uniqueIndex:uq_social_channel_accounts_identity,priority:4" json:"account_id"`
	DisplayName     string            `gorm:"column:display_name;type:text;not null" json:"display_name"`
	Status          string            `gorm:"column:status;type:varchar(32);not null;default:'pending';index:idx_social_channel_accounts_status" json:"status"`
	OwnerUserUUID   string            `gorm:"column:owner_user_uuid;type:text;not null;index:idx_social_channel_accounts_owner" json:"owner_user_uuid"`
	MemberUserUUIDs []string          `gorm:"column:member_user_uuids;type:jsonb;serializer:json" json:"member_user_uuids,omitempty"`
	Capabilities    datatypes.JSONMap `gorm:"column:capabilities;type:jsonb;default:'{}'::jsonb" json:"capabilities"`
	Credentials     datatypes.JSONMap `gorm:"column:credentials;type:jsonb;default:'{}'::jsonb" json:"credentials,omitempty"`
	CreatedAt       time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt    `gorm:"column:deleted_at;type:timestamptz" json:"deleted_at,omitempty"`
}

func (ChannelAccount) TableName() string {
	return models.S(models.TableSocialChannelAccounts)
}
