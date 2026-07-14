package opportunity

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
)

type OpportunityStageConfig struct {
	ConfigUUID        string    `gorm:"column:config_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"config_uuid"`
	TenantUUID        string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_opp_stage_config_tenant_order,priority:1" json:"tenant_uuid"`
	PipelineGroupUUID string    `gorm:"column:pipeline_group_uuid;type:uuid;index:idx_opp_stage_config_group_order,priority:1;index:idx_opp_stage_config_group_key,priority:1" json:"pipeline_group_uuid,omitempty"`
	StageKey          string    `gorm:"column:stage_key;type:varchar(64);not null;index:idx_opp_stage_config_tenant_key,priority:2;index:idx_opp_stage_config_group_key,priority:2" json:"stage_key"`
	Label             string    `gorm:"column:label;type:varchar(128);not null" json:"label"`
	SortOrder         int       `gorm:"column:sort_order;type:int;not null;default:0;index:idx_opp_stage_config_tenant_order,priority:2;index:idx_opp_stage_config_group_order,priority:2" json:"sort_order"`
	DefaultWinRate    int       `gorm:"column:default_win_rate;type:int;not null;default:0" json:"default_win_rate"`
	SLADays           int       `gorm:"column:sla_days;type:int;not null;default:0" json:"sla_days"`
	StageType         string    `gorm:"column:stage_type;type:varchar(32);not null;default:'active'" json:"stage_type"`
	FixedStage        string    `gorm:"column:fixed_stage;type:varchar(32);not null" json:"fixed_stage"`
	IsActive          bool      `gorm:"column:is_active;type:boolean;not null;default:true" json:"is_active"`
	MigrationPolicy   string    `gorm:"column:migration_policy;type:varchar(32);not null;default:'map_to_fixed'" json:"migration_policy"`
	CreatedBy         string    `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy         string    `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt         time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (OpportunityStageConfig) TableName() string {
	return domainmodels.S(domainmodels.TableOpportunityStageConfigs)
}
