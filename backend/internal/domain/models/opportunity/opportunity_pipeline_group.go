package opportunity

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
)

type OpportunityPipelineGroup struct {
	GroupUUID   string    `gorm:"column:group_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"group_uuid"`
	TenantUUID  string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_opp_pipeline_group_tenant_key,priority:1;index:idx_opp_pipeline_group_tenant_default,priority:1" json:"tenant_uuid"`
	GroupKey    string    `gorm:"column:group_key;type:varchar(64);not null;index:idx_opp_pipeline_group_tenant_key,priority:2" json:"group_key"`
	Name        string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Description string    `gorm:"column:description;type:text" json:"description,omitempty"`
	IsDefault   bool      `gorm:"column:is_default;type:boolean;not null;default:false;index:idx_opp_pipeline_group_tenant_default,priority:2" json:"is_default"`
	IsActive    bool      `gorm:"column:is_active;type:boolean;not null;default:true" json:"is_active"`
	SortOrder   int       `gorm:"column:sort_order;type:int;not null;default:0" json:"sort_order"`
	CreatedBy   string    `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy   string    `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (OpportunityPipelineGroup) TableName() string {
	return domainmodels.S(domainmodels.TableOpportunityPipelineGroups)
}
