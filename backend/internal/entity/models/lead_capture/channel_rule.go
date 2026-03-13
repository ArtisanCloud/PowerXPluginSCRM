package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// ChannelRule stores tenant-scoped behavior switches by channel/app type.
type ChannelRule struct {
	RuleUUID                     string    `gorm:"column:rule_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"rule_uuid"`
	TenantUUID                   string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_channel_rules_tenant;uniqueIndex:uq_lead_capture_channel_rules_tenant_channel_app,priority:1" json:"tenant_uuid"`
	Channel                      string    `gorm:"column:channel;type:varchar(64);not null;index:idx_lead_capture_channel_rules_channel;uniqueIndex:uq_lead_capture_channel_rules_tenant_channel_app,priority:2" json:"channel"`
	AppType                      string    `gorm:"column:app_type;type:varchar(64);not null;index:idx_lead_capture_channel_rules_app;uniqueIndex:uq_lead_capture_channel_rules_tenant_channel_app,priority:3" json:"app_type"`
	AutoCreateLeadFromCustomerDM bool      `gorm:"column:auto_create_lead_from_customer_dm;type:boolean;not null;default:false" json:"auto_create_lead_from_customer_dm"`
	CreatedAt                    time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt                    time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (ChannelRule) TableName() string {
	return models.S(models.TableLeadCaptureChannelRules)
}
