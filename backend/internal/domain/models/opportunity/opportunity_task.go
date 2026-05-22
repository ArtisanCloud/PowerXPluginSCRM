package opportunity

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
)

type OpportunityTask struct {
	TaskUUID        string     `gorm:"column:task_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"task_uuid"`
	TenantUUID      string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_opp_task_tenant_opp_status,priority:1" json:"tenant_uuid"`
	OpportunityUUID string     `gorm:"column:opportunity_uuid;type:uuid;not null;index:idx_opp_task_tenant_opp_status,priority:2" json:"opportunity_uuid"`
	Title           string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	DueAt           *time.Time `gorm:"column:due_at;type:timestamptz;index:idx_opp_task_due_at" json:"due_at,omitempty"`
	Status          string     `gorm:"column:status;type:varchar(32);not null;default:'open';index:idx_opp_task_tenant_opp_status,priority:3" json:"status"`
	CreatedBy       string     `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy       string     `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (OpportunityTask) TableName() string {
	return domainmodels.S(domainmodels.TableOpportunityTasks)
}
