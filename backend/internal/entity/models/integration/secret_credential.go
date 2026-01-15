package integration

import (
	"time"

	models "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	SecretStatusActive   = "ACTIVE"
	SecretStatusRotating = "ROTATING"
	SecretStatusRevoked  = "REVOKED"
)

// SecretCredential 描述外部 API 凭证的生命周期信息。
type SecretCredential struct {
	ID                string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid        string            `gorm:"column:tenant_uuid;type:uuid;not null" json:"tenant_uuid"`
	IntegrationType   string            `gorm:"column:integration_type;type:text;not null" json:"integration_type"`
	CurrentSecretRef  string            `gorm:"column:current_secret_ref;type:text" json:"current_secret_ref,omitempty"`
	PendingSecretRef  string            `gorm:"column:pending_secret_ref;type:text" json:"pending_secret_ref,omitempty"`
	RotationInterval  int               `gorm:"column:rotation_interval_days;type:int;not null;default:30" json:"rotation_interval_days"`
	LastRotatedAt     *time.Time        `gorm:"column:last_rotated_at;type:timestamptz" json:"last_rotated_at,omitempty"`
	NextRotationDueAt *time.Time        `gorm:"column:next_rotation_due_at;type:timestamptz" json:"next_rotation_due_at,omitempty"`
	Status            string            `gorm:"column:status;type:text;not null;default:'ACTIVE'" json:"status"`
	AuditLog          datatypes.JSON    `gorm:"column:audit_log;type:jsonb;default:'[]'::jsonb" json:"audit_log,omitempty"`
	Metadata          datatypes.JSONMap `gorm:"column:metadata;type:jsonb;default:'{}'::jsonb" json:"metadata"`
	CreatedAt         time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName returns the secrets table name.
func (SecretCredential) TableName() string {
	return models.S(models.TableIntegrationSecrets)
}
