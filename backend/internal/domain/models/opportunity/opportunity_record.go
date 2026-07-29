package opportunity

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
	"gorm.io/datatypes"
)

type OpportunityRecord struct {
	OpportunityUUID   string         `gorm:"column:opportunity_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"opportunity_uuid"`
	TenantUUID        string         `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_opp_tenant_stage,priority:1;index:idx_opp_tenant_owner_stage,priority:1;index:idx_opp_tenant_lead,priority:1;index:idx_opp_tenant_expected_close,priority:1" json:"tenant_uuid"`
	LeadUUID          string         `gorm:"column:lead_uuid;type:uuid;not null;index:idx_opp_tenant_lead,priority:2" json:"lead_uuid"`
	Title             string         `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Stage             string         `gorm:"column:stage;type:varchar(32);not null;default:'open';index:idx_opp_tenant_stage,priority:2;index:idx_opp_tenant_owner_stage,priority:3" json:"stage"`
	PipelineGroupUUID string         `gorm:"column:pipeline_group_uuid;type:uuid;index:idx_opp_tenant_pipeline_stage,priority:2" json:"pipeline_group_uuid,omitempty"`
	CurrentStageUUID  string         `gorm:"column:current_stage_uuid;type:uuid;index:idx_opp_tenant_pipeline_stage,priority:3" json:"current_stage_uuid,omitempty"`
	Amount            *float64       `gorm:"column:amount;type:numeric(18,2)" json:"amount,omitempty"`
	Currency          string         `gorm:"column:currency;type:varchar(8);not null;default:'CNY'" json:"currency"`
	Probability       int            `gorm:"column:probability;type:int;not null;default:0" json:"probability"`
	OwnerUserUUID     string         `gorm:"column:owner_user_uuid;type:text;not null;index:idx_opp_tenant_owner_stage,priority:2" json:"owner_user_uuid"`
	SourceChannel     string         `gorm:"column:source_channel;type:varchar(64)" json:"source_channel,omitempty"`
	SourceAppType     string         `gorm:"column:source_app_type;type:varchar(64)" json:"source_app_type,omitempty"`
	SourceAccountUUID *string        `gorm:"column:source_account_uuid;type:uuid" json:"source_account_uuid,omitempty"`
	ExternalUserID    string         `gorm:"column:external_userid;type:varchar(128)" json:"external_userid,omitempty"`
	ExpectedCloseAt   *time.Time     `gorm:"column:expected_close_at;type:timestamptz;index:idx_opp_tenant_expected_close,priority:2" json:"expected_close_at,omitempty"`
	WonAt             *time.Time     `gorm:"column:won_at;type:timestamptz" json:"won_at,omitempty"`
	LostAt            *time.Time     `gorm:"column:lost_at;type:timestamptz" json:"lost_at,omitempty"`
	LostReason        string         `gorm:"column:lost_reason;type:text" json:"lost_reason,omitempty"`
	RiskFlags         datatypes.JSON `gorm:"column:risk_flags;type:jsonb;not null;default:'[]'::jsonb" json:"risk_flags"`
	CreatedBy         string         `gorm:"column:created_by;type:varchar(64);not null" json:"created_by"`
	UpdatedBy         string         `gorm:"column:updated_by;type:varchar(64);not null" json:"updated_by"`
	CreatedAt         time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (OpportunityRecord) TableName() string {
	return domainmodels.S(domainmodels.TableOpportunityRecords)
}
