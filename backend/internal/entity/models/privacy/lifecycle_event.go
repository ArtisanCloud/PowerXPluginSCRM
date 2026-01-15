package privacy

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// Lifecycle event types and statuses.
const (
	LifecycleEventRetentionStart = "RETENTION_START"
	LifecycleEventRetentionPurge = "RETENTION_PURGE"
	LifecycleEventExport         = "EXPORT"
	LifecycleEventErasure        = "ERASURE"
	LifecycleEventConsentRevoke  = "CONSENT_REVOKE"
	LifecycleEventConsentRenew   = "CONSENT_RENEW"

	LifecycleStatusPending   = "PENDING"
	LifecycleStatusSucceeded = "SUCCEEDED"
	LifecycleStatusFailed    = "FAILED"
)

// LifecycleEvent stores evidence for retention, export, and erasure actions.
type LifecycleEvent struct {
	ID         string         `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid string         `gorm:"column:tenant_uuid;type:uuid;not null" json:"tenant_uuid"`
	EventType  string         `gorm:"column:event_type;type:text;not null" json:"event_type"`
	AssetKey   string         `gorm:"column:asset_key;type:text;not null" json:"asset_key"`
	Payload    datatypes.JSON `gorm:"column:payload;type:jsonb" json:"payload,omitempty"`
	OccurredAt time.Time      `gorm:"column:occurred_at;type:timestamptz;not null;default:now()" json:"occurred_at"`
	RecordedBy string         `gorm:"column:recorded_by;type:text;not null" json:"recorded_by"`
	Status     string         `gorm:"column:status;type:text;not null;default:'PENDING'" json:"status"`
	CreatedAt  time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (*LifecycleEvent) TableName() string {
	return models.S(models.TablePrivacyLifecycleEvents)
}
