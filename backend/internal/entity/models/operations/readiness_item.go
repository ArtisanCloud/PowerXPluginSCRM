package operations

import (
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// ReadinessChecklistItem represents an operations readiness checklist entry.
type ReadinessChecklistItem struct {
	ID          string     `gorm:"primaryKey;type:uuid" json:"id"`
	PluginID    string     `gorm:"column:plugin_id;index:idx_operations_readiness_type" json:"plugin_id"`
	Type        string     `gorm:"column:type;index:idx_operations_readiness_type" json:"type"`
	ItemKey     string     `gorm:"column:item_key" json:"item_key"`
	Description string     `gorm:"column:description" json:"description"`
	Status      string     `gorm:"column:status" json:"status"`
	OwnerRole   string     `gorm:"column:owner_role" json:"owner_role"`
	DueDate     *time.Time `gorm:"column:due_date" json:"due_date,omitempty"`
	CompletedAt *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	Notes       string     `gorm:"column:notes" json:"notes"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName implements gorm tablename.
func (ReadinessChecklistItem) TableName() string {
	return basemodels.S(basemodels.TableOperationsReadinessItems)
}
