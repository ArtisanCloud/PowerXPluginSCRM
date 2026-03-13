package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// LeadSource records attribution information for a lead.
type LeadSource struct {
	SourceUUID   string    `gorm:"column:source_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"source_uuid"`
	LeadUUID     string    `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_capture_sources_lead" json:"lead_uuid"`
	TenantUUID   string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_sources_tenant" json:"tenant_uuid"`
	ChannelCode  string    `gorm:"column:channel_code;type:varchar(64)" json:"channel_code"`
	AppType      string    `gorm:"column:app_type;type:varchar(64)" json:"app_type"`
	AccountUUID  *string   `gorm:"column:account_uuid;type:uuid" json:"account_uuid,omitempty"`
	CampaignCode string    `gorm:"column:campaign_code;type:text" json:"campaign_code"`
	UTMSource    string    `gorm:"column:utm_source;type:text" json:"utm_source"`
	UTMMedium    string    `gorm:"column:utm_medium;type:text" json:"utm_medium"`
	UTMCampaign  string    `gorm:"column:utm_campaign;type:text" json:"utm_campaign"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadSource) TableName() string {
	return models.S(models.TableLeadCaptureSources)
}
