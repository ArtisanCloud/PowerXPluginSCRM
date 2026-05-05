package opportunity

import (
	"time"

	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
	"gorm.io/datatypes"
)

type OpportunityActivity struct {
	ActivityUUID     string         `gorm:"column:activity_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"activity_uuid"`
	TenantUUID       string         `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_opp_act_tenant_opp_created,priority:1;index:idx_opp_act_tenant_type_created,priority:1" json:"tenant_uuid"`
	OpportunityUUID  string         `gorm:"column:opportunity_uuid;type:uuid;not null;index:idx_opp_act_tenant_opp_created,priority:2" json:"opportunity_uuid"`
	ActivityType     string         `gorm:"column:activity_type;type:varchar(32);not null;index:idx_opp_act_tenant_type_created,priority:2" json:"activity_type"`
	FromStage        string         `gorm:"column:from_stage;type:varchar(32)" json:"from_stage,omitempty"`
	ToStage          string         `gorm:"column:to_stage;type:varchar(32)" json:"to_stage,omitempty"`
	Payload          datatypes.JSON `gorm:"column:payload;type:jsonb;not null;default:'{}'::jsonb" json:"payload"`
	OperatorUserUUID string         `gorm:"column:operator_user_uuid;type:uuid;not null" json:"operator_user_uuid"`
	RequestID        string         `gorm:"column:request_id;type:varchar(128)" json:"request_id,omitempty"`
	CreatedAt        time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now();index:idx_opp_act_tenant_opp_created,priority:3;index:idx_opp_act_tenant_type_created,priority:3" json:"created_at"`
}

func (OpportunityActivity) TableName() string {
	return domainmodels.S(domainmodels.TableOpportunityActivities)
}
