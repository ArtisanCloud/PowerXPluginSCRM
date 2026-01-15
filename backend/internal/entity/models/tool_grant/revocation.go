package tool_grant

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// Revocation records a revoked ToolGrant lease.
type Revocation struct {
	ID          string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid  string    `gorm:"column:tenant_uuid;type:uuid;not null" json:"tenant_uuid"`
	ToolGrantID string    `gorm:"column:toolgrant_id;type:text;not null" json:"toolgrant_id"`
	RevokedAt   time.Time `gorm:"column:revoked_at;type:timestamptz;not null" json:"revoked_at"`
	RevokedBy   string    `gorm:"column:revoked_by;type:text;not null" json:"revoked_by"`
	Reason      string    `gorm:"column:reason;type:text" json:"reason,omitempty"`
	TtlExpiry   time.Time `gorm:"column:ttl_expiry;type:timestamptz;not null" json:"ttl_expiry"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

func (*Revocation) TableName() string {
	return models.S(models.TableToolGrantRevocations)
}
