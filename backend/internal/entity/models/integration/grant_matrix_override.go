package integration

import (
	"time"

	models "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// GrantMatrixOverride 存储 GrantMatrix 的数据库覆盖项。
type GrantMatrixOverride struct {
	ID          string         `gorm:"column:id;primaryKey" json:"id"`
	Scope       string         `gorm:"column:scope;not null" json:"scope"`
	Channel     string         `gorm:"column:channel;not null" json:"channel"`
	Resource    string         `gorm:"column:resource;not null" json:"resource"`
	Action      string         `gorm:"column:action;not null" json:"action"`
	Constraints datatypes.JSON `gorm:"column:constraints;not null" json:"constraints"`
	Status      string         `gorm:"column:status;not null" json:"status"`
	Version     int            `gorm:"column:version;not null" json:"version"`
	ApprovedBy  string         `gorm:"column:approved_by" json:"approved_by"`
	ApprovedAt  *time.Time     `gorm:"column:approved_at" json:"approved_at"`
	CreatedAt   time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at" json:"updated_at"`
}

func (GrantMatrixOverride) TableName() string {
	return models.S(models.TableIntegrationGrantMatrixOverrides)
}
