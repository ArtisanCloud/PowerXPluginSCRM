package tool_grant

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// UsageEvent captures ToolGrant lifecycle actions.
type UsageEvent struct {
	ID          string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid  string            `gorm:"column:tenant_uuid;type:uuid;not null" json:"tenant_uuid"`
	ToolGrantID string            `gorm:"column:toolgrant_id;type:text;not null" json:"toolgrant_id"`
	EventType   string            `gorm:"column:event_type;type:text;not null" json:"event_type"`
	Capability  string            `gorm:"column:capability;type:text;not null" json:"capability"`
	AgentID     string            `gorm:"column:agent_id;type:text;not null" json:"agent_id"`
	OccurredAt  time.Time         `gorm:"column:occurred_at;type:timestamptz;not null" json:"occurred_at"`
	Metadata    datatypes.JSONMap `gorm:"column:metadata;type:jsonb" json:"metadata,omitempty"`
}

func (*UsageEvent) TableName() string {
	return models.S(models.TableToolGrantUsageEvents)
}
