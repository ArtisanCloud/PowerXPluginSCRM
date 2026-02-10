package social_channel_governance

import (
	"context"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/sirupsen/logrus"
)

const (
	AuditEventAccountCreated     = model.AuditEventAccountCreated
	AuditEventAuthChanged        = model.AuditEventAuthChanged
	AuditEventMembersChanged     = model.AuditEventMembersChanged
	AuditEventCapabilitiesChange = model.AuditEventCapabilitiesChange
)

type AuditEvent struct {
	EventType     string
	TenantUUID    string
	AccountUUID   string
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

// EmitAuditEvent logs a structured audit event for social channel governance.
func EmitAuditEvent(ctx context.Context, event AuditEvent) {
	entry := logger.WithFields(logrus.Fields{
		"component":       "social_channel_governance_audit",
		"event":           event.EventType,
		"tenant_uuid":     event.TenantUUID,
		"account_uuid":    event.AccountUUID,
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
	entry.Info("social channel governance audit event")
}

func EmitChannelAccountCreated(ctx context.Context, tenantUUID, accountUUID, actorUUID, status string) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventAccountCreated,
		TenantUUID:    tenantUUID,
		AccountUUID:   accountUUID,
		ActorUserUUID: actorUUID,
		OccurredAt:    time.Now().UTC(),
		Metadata: map[string]any{
			"status": status,
		},
	})
}

func EmitChannelAccountMembersChanged(ctx context.Context, tenantUUID, accountUUID, actorUUID string, ownerUUID *string, members []string) {
	metadata := map[string]any{
		"member_user_uuids": members,
	}
	if ownerUUID != nil {
		metadata["owner_member_uuid"] = *ownerUUID
	}
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventMembersChanged,
		TenantUUID:    tenantUUID,
		AccountUUID:   accountUUID,
		ActorUserUUID: actorUUID,
		OccurredAt:    time.Now().UTC(),
		Metadata:      metadata,
	})
}

func EmitChannelAccountCapabilitiesChanged(ctx context.Context, tenantUUID, accountUUID, actorUUID string, capabilities map[string]bool) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventCapabilitiesChange,
		TenantUUID:    tenantUUID,
		AccountUUID:   accountUUID,
		ActorUserUUID: actorUUID,
		OccurredAt:    time.Now().UTC(),
		Metadata: map[string]any{
			"capabilities": capabilities,
		},
	})
}
