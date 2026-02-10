package org_sync

import (
	"context"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/sirupsen/logrus"
)

const (
	AuditEventSourceSyncTriggered = "org_sync.source.sync.triggered"
	AuditEventMappingsConfirmed   = "org_sync.mappings.confirmed"
	AuditEventMainViewQueried     = "org_sync.main_view.queried"
)

type AuditEvent struct {
	EventType     string
	TenantUUID    string
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

func EmitAuditEvent(ctx context.Context, event AuditEvent) {
	entry := logger.WithFields(logrus.Fields{
		"component":       "org_sync_audit",
		"event":           event.EventType,
		"tenant_uuid":     event.TenantUUID,
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
	entry.Info("org sync audit event")
}

func EmitSourceSyncTriggered(ctx context.Context, tenantUUID, actorUUID, sourceAccountUUID string) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventSourceSyncTriggered,
		TenantUUID:    tenantUUID,
		ActorUserUUID: actorUUID,
		OccurredAt:    time.Now().UTC(),
		Metadata: map[string]any{
			"source_account_uuid": sourceAccountUUID,
		},
	})
}

func EmitMappingsConfirmed(ctx context.Context, tenantUUID, actorUUID string, unitCount, memberCount int) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventMappingsConfirmed,
		TenantUUID:    tenantUUID,
		ActorUserUUID: actorUUID,
		OccurredAt:    time.Now().UTC(),
		Metadata: map[string]any{
			"unit_mappings":   unitCount,
			"member_mappings": memberCount,
		},
	})
}

func EmitMainViewQueried(ctx context.Context, tenantUUID, actorUUID string, keyword string, total int) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventMainViewQueried,
		TenantUUID:    tenantUUID,
		ActorUserUUID: actorUUID,
		OccurredAt:    time.Now().UTC(),
		Metadata: map[string]any{
			"keyword": keyword,
			"total":   total,
		},
	})
}
