package social_channel_governance

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

const (
	AuditEventAccountCreated     = "account_created"
	AuditEventAuthChanged        = "auth_changed"
	AuditEventMembersChanged     = "members_changed"
	AuditEventCapabilitiesChange = "capabilities_changed"
)

// AuditEvent captures governance changes for channel accounts.
type AuditEvent struct {
	EventUUID     string    `gorm:"column:event_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"event_uuid"`
	TenantUuid    string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_social_channel_audit_tenant" json:"tenant_uuid"`
	AccountUUID   string    `gorm:"column:account_uuid;type:uuid;not null;index:idx_social_channel_audit_account" json:"account_uuid"`
	EventType     string    `gorm:"column:event_type;type:varchar(64);not null" json:"event_type"`
	ActorUserUUID string    `gorm:"column:actor_user_uuid;type:uuid;not null" json:"actor_user_uuid"`
	OccurredAt    time.Time `gorm:"column:occurred_at;type:timestamptz;not null" json:"occurred_at"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (AuditEvent) TableName() string {
	return models.S(models.TableSocialChannelAuditEvents)
}
