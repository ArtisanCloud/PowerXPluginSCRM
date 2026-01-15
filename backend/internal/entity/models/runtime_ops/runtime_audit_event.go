package runtime_ops

import "time"

// RuntimeAuditEvent represents an audit event recorded by runtime ops.
type RuntimeAuditEvent struct {
	ID         string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PluginID   string    `gorm:"column:plugin_id;type:text;not null" json:"plugin_id"`
	TenantUUID string    `gorm:"column:tenant_uuid;type:uuid" json:"tenant_uuid,omitempty"`
	EventType  string    `gorm:"column:event_type;type:text;not null" json:"event_type"`
	Payload    string    `gorm:"column:payload;type:jsonb" json:"payload"`
	OccurredAt time.Time `gorm:"column:occurred_at;type:timestamptz;not null;default:now()" json:"occurred_at"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}
