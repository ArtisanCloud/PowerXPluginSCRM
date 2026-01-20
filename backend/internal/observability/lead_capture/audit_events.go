package lead_capture

import (
	"context"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/sirupsen/logrus"
)

const (
	AuditEventLeadCreated      = "lead.created"
	AuditEventLeadMerged       = "lead.merged"
	AuditEventLeadAssigned     = "lead.assigned"
	AuditEventLeadStatusChange = "lead.status.changed"
)

type AuditEvent struct {
	EventType     string
	TenantUUID    string
	LeadUUID      string
	ActorUserUUID string
	OccurredAt    time.Time
	Metadata      map[string]any
}

// ResolveActorUserUUID attempts to resolve actor id from context, then falls back.
func ResolveActorUserUUID(ctx context.Context, fallback string) string {
	if ctx != nil {
		if v := ctx.Value("actor_user_uuid"); v != nil {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
		if v := ctx.Value("user_uuid"); v != nil {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return fallback
}

// EmitAuditEvent logs a structured audit event for lead capture.
func EmitAuditEvent(ctx context.Context, event AuditEvent) {
	entry := logger.WithFields(logrus.Fields{
		"component":       "lead_capture_audit",
		"event":           event.EventType,
		"tenant_uuid":     event.TenantUUID,
		"lead_uuid":       event.LeadUUID,
		"actor_user_uuid": event.ActorUserUUID,
	})
	if ctx != nil {
		if requestID := ctx.Value("request_id"); requestID != nil {
			entry = entry.WithField("request_id", requestID)
		}
	}
	if !event.OccurredAt.IsZero() {
		entry = entry.WithField("occurred_at", event.OccurredAt)
	}
	for key, value := range event.Metadata {
		entry = entry.WithField(key, value)
	}
	entry.Info("lead capture audit event")
}

func EmitLeadCreated(ctx context.Context, tenantUUID, leadUUID, actorUUID string, metadata map[string]any) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventLeadCreated,
		TenantUUID:    tenantUUID,
		LeadUUID:      leadUUID,
		ActorUserUUID: actorUUID,
		OccurredAt:    time.Now().UTC(),
		Metadata:      metadata,
	})
}

func EmitLeadMerged(ctx context.Context, tenantUUID, leadUUID, actorUUID string, metadata map[string]any) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventLeadMerged,
		TenantUUID:    tenantUUID,
		LeadUUID:      leadUUID,
		ActorUserUUID: actorUUID,
		OccurredAt:    time.Now().UTC(),
		Metadata:      metadata,
	})
}

func EmitLeadAssigned(ctx context.Context, tenantUUID, leadUUID, actorUUID string, metadata map[string]any) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventLeadAssigned,
		TenantUUID:    tenantUUID,
		LeadUUID:      leadUUID,
		ActorUserUUID: actorUUID,
		OccurredAt:    time.Now().UTC(),
		Metadata:      metadata,
	})
}

func EmitLeadStatusChanged(ctx context.Context, tenantUUID, leadUUID, actorUUID string, metadata map[string]any) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventLeadStatusChange,
		TenantUUID:    tenantUUID,
		LeadUUID:      leadUUID,
		ActorUserUUID: actorUUID,
		OccurredAt:    time.Now().UTC(),
		Metadata:      metadata,
	})
}
