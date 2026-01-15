package security

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// Advisory distribution channels.
const (
	DistributionChannelMarketplace = "MARKETPLACE"
	DistributionChannelEmail       = "EMAIL"
	DistributionChannelWebhook     = "WEBHOOK"
)

// Advisory distribution statuses.
const (
	DistributionStatusPending      = "PENDING"
	DistributionStatusDelivered    = "DELIVERED"
	DistributionStatusFailed       = "FAILED"
	DistributionStatusAcknowledged = "ACKNOWLEDGED"
)

// AdvisoryDistribution represents a notification delivery record for an advisory.
type AdvisoryDistribution struct {
	ID          string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	AdvisoryID  string            `gorm:"column:advisory_id;type:uuid;not null" json:"advisory_id"`
	TenantUuid  string            `gorm:"column:tenant_uuid;type:uuid;not null" json:"tenant_uuid"`
	Channel     string            `gorm:"column:channel;type:text;not null" json:"channel"`
	DeliveredAt *time.Time        `gorm:"column:delivered_at;type:timestamptz" json:"delivered_at,omitempty"`
	Status      string            `gorm:"column:status;type:text;not null" json:"status"`
	Metadata    datatypes.JSONMap `gorm:"column:metadata;type:jsonb" json:"metadata,omitempty"`
	CreatedAt   time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName returns the fully qualified table name.
func (*AdvisoryDistribution) TableName() string {
	return models.S(models.TableSecurityAdvisoryDistributions)
}
