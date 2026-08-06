package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

const LeadAttachmentStorageProviderDatabase = "database"

// LeadAttachment stores files attached to lead lifecycle nodes and activities.
type LeadAttachment struct {
	AttachmentUUID  string    `gorm:"column:attachment_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"attachment_uuid"`
	TenantUUID      string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_attachments_tenant" json:"tenant_uuid"`
	LeadUUID        string    `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_capture_attachments_lead" json:"lead_uuid"`
	ActivityUUID    string    `gorm:"column:activity_uuid;type:uuid;index:idx_lead_capture_attachments_activity" json:"activity_uuid,omitempty"`
	StageKey        string    `gorm:"column:stage_key;type:varchar(64);index:idx_lead_capture_attachments_scope" json:"stage_key,omitempty"`
	ActionKey       string    `gorm:"column:action_key;type:varchar(64);index:idx_lead_capture_attachments_scope" json:"action_key,omitempty"`
	FileName        string    `gorm:"column:file_name;type:text;not null" json:"file_name"`
	ContentType     string    `gorm:"column:content_type;type:text" json:"content_type"`
	FileSize        int64     `gorm:"column:file_size;type:bigint;not null;default:0" json:"file_size"`
	StorageProvider string    `gorm:"column:storage_provider;type:varchar(32);not null;default:'database'" json:"storage_provider"`
	Content         []byte    `gorm:"column:content;type:bytea;not null" json:"-"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (LeadAttachment) TableName() string {
	return models.S(models.TableLeadCaptureAttachments)
}
