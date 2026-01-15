package integration

import (
	"time"

	"gorm.io/datatypes"
)

// ChangeApproval 表示一次配置变更的审批记录。
type ChangeApproval struct {
	ID          string         `gorm:"column:id;primaryKey" json:"id"`
	TargetType  string         `gorm:"column:target_type;not null" json:"target_type"`
	TargetID    string         `gorm:"column:target_id;not null" json:"target_id"`
	Payload     datatypes.JSON `gorm:"column:payload;not null" json:"payload"`
	Status      string         `gorm:"column:status;not null" json:"status"`
	SubmittedBy string         `gorm:"column:submitted_by;not null" json:"submitted_by"`
	SubmittedAt time.Time      `gorm:"column:submitted_at;not null" json:"submitted_at"`
	ReviewedBy  *string        `gorm:"column:reviewed_by" json:"reviewed_by"`
	ReviewedAt  *time.Time     `gorm:"column:reviewed_at" json:"reviewed_at"`
	Reason      string         `gorm:"column:reason" json:"reason"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName implements gorm's tabler interface.
func (ChangeApproval) TableName() string {
	return "integration_change_approvals"
}
