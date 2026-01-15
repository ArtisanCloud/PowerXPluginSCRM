package template

// internal/entity/models/template/template.go

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// Template represents a reusable snippet that can be shared across the Base plugin.
type Template struct {
	models.BaseModel
	Name           string     `gorm:"type:varchar(255);not null;comment:模板名称" json:"name"`
	Description    string     `gorm:"type:text;comment:模板描述" json:"description"`
	Content        string     `gorm:"type:text;comment:模板内容" json:"content"`
	Status         string     `gorm:"type:varchar(50);not null;default:'draft';comment:发布状态(draft/published/archived) " json:"status"`
	ReviewStatus   string     `gorm:"type:varchar(50);not null;default:'pending';comment:审核状态(pending/approved/rejected)" json:"review_status"`
	ReviewComment  string     `gorm:"type:text;comment:审核备注" json:"review_comment"`
	ReviewedBy     string     `gorm:"type:varchar(100);comment:审核人" json:"reviewed_by"`
	ReviewedAt     *time.Time `gorm:"comment:审核时间" json:"reviewed_at"`
	PublishChannel string     `gorm:"type:varchar(120);comment:发布渠道" json:"publish_channel"`
	PublishedAt    *time.Time `gorm:"comment:发布时间" json:"published_at"`
	CleanupReason  string     `gorm:"type:varchar(255);comment:清理原因" json:"cleanup_reason"`
	CleanedAt      *time.Time `gorm:"comment:清理完成时间" json:"cleaned_at"`
}

func (t *Template) TableName() string {
	return models.S(models.TableTemplate)
}
