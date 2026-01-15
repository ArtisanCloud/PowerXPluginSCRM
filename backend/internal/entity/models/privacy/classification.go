package privacy

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// Data classification categories.
const (
	CategoryPII         = "PII"
	CategoryBusiness    = "BUSINESS"
	CategoryLog         = "LOG"
	CategoryAIInput     = "AI_INPUT"
	CategoryAIOutput    = "AI_OUTPUT"
	LawfulConsent       = "CONSENT"
	LawfulContract      = "CONTRACT"
	LawfulLegal         = "LEGAL_OBLIGATION"
	LawfulInterest      = "LEGITIMATE_INTEREST"
	LawfulPublicTask    = "PUBLIC_TASK"
	LawfulVitalInterest = "VITAL_INTEREST"
)

// DataClassification maps tenant datasets to classification tiers and lawful basis.
type DataClassification struct {
	ID              string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid      string            `gorm:"column:tenant_uuid;type:uuid;not null" json:"tenant_uuid"`
	AssetKey        string            `gorm:"column:asset_key;type:text;not null" json:"asset_key"`
	Category        string            `gorm:"column:category;type:text;not null" json:"category"`
	LawfulBasis     string            `gorm:"column:lawful_basis;type:text;not null" json:"lawful_basis"`
	RetentionPolicy datatypes.JSONMap `gorm:"column:retention_policy;type:jsonb" json:"retention_policy,omitempty"`
	Purpose         string            `gorm:"column:purpose;type:text;not null" json:"purpose"`
	CreatedAt       time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (*DataClassification) TableName() string {
	return models.S(models.TablePrivacyDataClassifications)
}
