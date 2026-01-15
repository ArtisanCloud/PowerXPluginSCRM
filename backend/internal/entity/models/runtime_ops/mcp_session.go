package runtime_ops

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MCPSession models the MCP connection for a plugin instance.
type MCPSession struct {
	ID                  string     `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	RuntimeAssignmentID string     `gorm:"column:runtime_assignment_id;type:uuid;not null" json:"runtime_assignment_id"`
	TenantUuid          string     `gorm:"column:tenant_uuid;type:uuid;not null" json:"tenant_uuid"`
	State               string     `gorm:"column:state;type:text;not null" json:"state"`
	JWTID               string     `gorm:"column:jwt_id;type:text" json:"jwt_id,omitempty"`
	CapabilitiesHash    string     `gorm:"column:capabilities_hash;type:text" json:"capabilities_hash,omitempty"`
	MissedHeartbeats    int        `gorm:"column:missed_heartbeats;type:int;not null;default:0" json:"missed_heartbeats"`
	LastPingAt          *DBTime    `gorm:"column:last_ping_at;type:timestamptz" json:"last_ping_at,omitempty"`
	ClosedAt            *DBTime    `gorm:"column:closed_at;type:timestamptz" json:"closed_at,omitempty"`
	CreatedAt           DBTime     `gorm:"column:created_at;type:timestamptz" json:"created_at"`
	UpdatedAt           DBTime     `gorm:"column:updated_at;type:timestamptz" json:"updated_at"`
}

func (s *MCPSession) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(s.ID) == "" {
		s.ID = uuid.NewString()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = DBTime{Time: time.Now().UTC()}
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = DBTime{Time: s.CreatedAt.Time}
	}
	return nil
}

func (s *MCPSession) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = DBTime{Time: time.Now().UTC()}
	return nil
}
