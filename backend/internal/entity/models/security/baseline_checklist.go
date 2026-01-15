package security

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// BaselineChecklist represents the versioned baseline control manifest.
type BaselineChecklist struct {
	ID        string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Version   string            `gorm:"column:version;type:text;not null" json:"version"`
	Controls  datatypes.JSONMap `gorm:"column:controls;type:jsonb;not null" json:"controls"`
	CreatedAt time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	RetiredAt *time.Time        `gorm:"column:retired_at;type:timestamptz" json:"retired_at,omitempty"`
}

// TableName returns the fully qualified table name.
func (*BaselineChecklist) TableName() string {
	return models.S(models.TableSecurityBaselineChecklists)
}
