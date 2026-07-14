package opportunity

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
)

type OpportunityPipelineTemplate struct {
	TemplateUUID string    `gorm:"column:template_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"template_uuid"`
	TemplateKey  string    `gorm:"column:template_key;type:varchar(64);not null;uniqueIndex:uq_opp_pipeline_template_key" json:"template_key"`
	GroupKey     string    `gorm:"column:group_key;type:varchar(64);not null" json:"group_key"`
	Name         string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Label        string    `gorm:"column:label;type:varchar(128);not null" json:"label"`
	Segment      string    `gorm:"column:segment;type:varchar(32);not null;default:''" json:"segment"`
	Description  string    `gorm:"column:description;type:text" json:"description,omitempty"`
	SortOrder    int       `gorm:"column:sort_order;type:int;not null;default:0" json:"sort_order"`
	IsActive     bool      `gorm:"column:is_active;type:boolean;not null;default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (OpportunityPipelineTemplate) TableName() string {
	return domainmodels.S(domainmodels.TableOpportunityPipelineTemplates)
}

type OpportunityPipelineTemplateStage struct {
	StageUUID      string    `gorm:"column:stage_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"stage_uuid"`
	TemplateKey    string    `gorm:"column:template_key;type:varchar(64);not null;index:idx_opp_pipeline_template_stage_order,priority:1;index:idx_opp_pipeline_template_stage_key,priority:1" json:"template_key"`
	StageKey       string    `gorm:"column:stage_key;type:varchar(64);not null;index:idx_opp_pipeline_template_stage_key,priority:2" json:"stage_key"`
	Label          string    `gorm:"column:label;type:varchar(128);not null" json:"label"`
	SortOrder      int       `gorm:"column:sort_order;type:int;not null;default:0;index:idx_opp_pipeline_template_stage_order,priority:2" json:"sort_order"`
	DefaultWinRate int       `gorm:"column:default_win_rate;type:int;not null;default:0" json:"default_win_rate"`
	SLADays        int       `gorm:"column:sla_days;type:int;not null;default:0" json:"sla_days"`
	StageType      string    `gorm:"column:stage_type;type:varchar(32);not null;default:'active'" json:"stage_type"`
	FixedStage     string    `gorm:"column:fixed_stage;type:varchar(32);not null" json:"fixed_stage"`
	IsActive       bool      `gorm:"column:is_active;type:boolean;not null;default:true" json:"is_active"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (OpportunityPipelineTemplateStage) TableName() string {
	return domainmodels.S(domainmodels.TableOpportunityPipelineTemplateStages)
}
