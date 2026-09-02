package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

const (
	LeadStatusCaptured            = "captured"
	LeadStatusEnriched            = "enriched"
	LeadStatusDeduplicated        = "deduplicated"
	LeadStatusRouted              = "routed"
	LeadStatusEngaging            = "engaging"
	LeadStatusQualifiedForHandoff = "qualified_for_handoff"
	LeadStatusHandoffPending      = "handoff_pending"
	LeadStatusHandoffAccepted     = "handoff_accepted"
	LeadStatusInvalid             = "invalid"
	LeadStatusArchived            = "archived"
	LeadStatusDisconnected        = "disconnected"
	LeadStatusHandoffFailed       = "handoff_failed"
)

func IsValidLeadStatus(status string) bool {
	switch status {
	case LeadStatusCaptured,
		LeadStatusEnriched,
		LeadStatusDeduplicated,
		LeadStatusRouted,
		LeadStatusEngaging,
		LeadStatusQualifiedForHandoff,
		LeadStatusHandoffPending,
		LeadStatusHandoffAccepted,
		LeadStatusInvalid,
		LeadStatusArchived,
		LeadStatusDisconnected,
		LeadStatusHandoffFailed:
		return true
	default:
		return false
	}
}

// Lead represents a captured lead in the current tenant.
type Lead struct {
	LeadUUID           string    `gorm:"column:lead_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"lead_uuid"`
	TenantUUID         string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_leads_tenant" json:"tenant_uuid"`
	DisplayName        string    `gorm:"column:display_name;type:text" json:"display_name"`
	Phone              string    `gorm:"column:phone;type:text" json:"phone"`
	Email              string    `gorm:"column:email;type:text" json:"email"`
	Status             string    `gorm:"column:status;type:varchar(32);not null;default:'captured';index:idx_lead_capture_leads_status" json:"status"`
	OwnerUserUUID      string    `gorm:"column:owner_user_uuid;type:text;index:idx_lead_capture_leads_owner" json:"owner_user_uuid"`
	SourceChannel      string    `gorm:"column:source_channel;type:varchar(64);index:idx_lead_capture_leads_source" json:"source_channel"`
	SourceAppType      string    `gorm:"column:source_app_type;type:varchar(64)" json:"source_app_type"`
	SourceAccountUUID  *string   `gorm:"column:source_account_uuid;type:uuid" json:"source_account_uuid,omitempty"`
	CreatedAt          time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	HasMerge           bool      `gorm:"-" json:"has_merge"`
	ExternalUserID     string    `gorm:"-" json:"external_userid,omitempty"`
	ExternalWechatID   string    `gorm:"-" json:"external_wechat_id,omitempty"`
	WeComFollowUserID  string    `gorm:"-" json:"wecom_follow_userid,omitempty"`
	WeComAdderUserID   string    `gorm:"-" json:"wecom_adder_userid,omitempty"`
	LeadOriginType     string    `gorm:"-" json:"lead_origin_type,omitempty"`     // channel | local
	ChannelSyncStatus  string    `gorm:"-" json:"channel_sync_status,omitempty"`  // synced | unsynced
	OwnerBindingStatus string    `gorm:"-" json:"owner_binding_status,omitempty"` // not_channel | missing_owner | mapped | unmapped
	OwnerMappingStatus string    `gorm:"-" json:"owner_mapping_status,omitempty"` // deprecated: keep for compatibility
}

func (Lead) TableName() string {
	return models.S(models.TableLeadCaptureLeads)
}
