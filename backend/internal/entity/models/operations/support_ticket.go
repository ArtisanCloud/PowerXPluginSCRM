package operations

import (
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// SupportTicket represents a support case filed by tenants or vendors.
type SupportTicket struct {
	ID              string            `gorm:"primaryKey;type:uuid" json:"id"`
	PluginID        string            `gorm:"column:plugin_id;index:idx_support_tickets_scope" json:"plugin_id"`
	TenantUuid      string            `gorm:"column:tenant_uuid;index:idx_support_tickets_scope" json:"tenant_uuid"`
	ChannelID       *string           `gorm:"column:channel_id" json:"channel_id,omitempty"`
	ExternalRef     *string           `gorm:"column:external_ref" json:"external_ref,omitempty"`
	Subject         string            `gorm:"column:subject" json:"subject"`
	Description     string            `gorm:"column:description" json:"description"`
	Priority        string            `gorm:"column:priority" json:"priority"`
	Status          string            `gorm:"column:status" json:"status"`
	RequestedBy     datatypes.JSONMap `gorm:"column:requested_by" json:"requested_by"`
	AssignedTeam    *string           `gorm:"column:assigned_team" json:"assigned_team,omitempty"`
	AssignedTo      *string           `gorm:"column:assigned_to" json:"assigned_to,omitempty"`
	FirstResponseAt *time.Time        `gorm:"column:first_response_at" json:"first_response_at,omitempty"`
	ResolvedAt      *time.Time        `gorm:"column:resolved_at" json:"resolved_at,omitempty"`
	ClosedAt        *time.Time        `gorm:"column:closed_at" json:"closed_at,omitempty"`
	CSATScore       *float64          `gorm:"column:csat_score" json:"csat_score,omitempty"`
	ResolutionCode  *string           `gorm:"column:resolution_code" json:"resolution_code,omitempty"`
	ReopenCount     int               `gorm:"column:reopen_count" json:"reopen_count"`
	CreatedAt       time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns table name for SupportTicket.
func (SupportTicket) TableName() string {
	return basemodels.S(basemodels.TableOperationsSupportTickets)
}

// SupportTicketEvent records the lifecycle events for a ticket.
type SupportTicketEvent struct {
	ID            int64             `gorm:"primaryKey;autoIncrement" json:"id"`
	TicketID      string            `gorm:"column:ticket_id;index" json:"ticket_id"`
	EventType     string            `gorm:"column:event_type" json:"event_type"`
	Payload       datatypes.JSONMap `gorm:"column:payload" json:"payload"`
	EmittedAt     time.Time         `gorm:"column:emitted_at" json:"emitted_at"`
	WebhookStatus string            `gorm:"column:webhook_status" json:"webhook_status"`
	RetryCount    int               `gorm:"column:retry_count" json:"retry_count"`
	CreatedAt     time.Time         `gorm:"column:created_at" json:"created_at"`
}

// TableName returns table name for events.
func (SupportTicketEvent) TableName() string {
	return basemodels.S(basemodels.TableOperationsSupportTicketEvents)
}
