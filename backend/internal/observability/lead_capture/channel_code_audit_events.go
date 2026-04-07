package lead_capture

import "context"

const (
	AuditEventChannelCodeCreated      = "lead.channel_code.created"
	AuditEventChannelCodeStatusChange = "lead.channel_code.status.changed"
	AuditEventWelcomeConfigSaved      = "lead.channel_code.welcome.saved"
	AuditEventWelcomeSyncPublished    = "lead.channel_code.welcome.published"
	AuditEventChannelCodeEventIngest  = "lead.channel_code.event.ingested"
)

func EmitChannelCodeCreated(ctx context.Context, tenantUUID, codeUUID, actorUUID string, metadata map[string]any) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventChannelCodeCreated,
		TenantUUID:    tenantUUID,
		LeadUUID:      codeUUID,
		ActorUserUUID: actorUUID,
		Metadata:      metadata,
	})
}

func EmitChannelCodeStatusChanged(ctx context.Context, tenantUUID, codeUUID, actorUUID string, metadata map[string]any) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventChannelCodeStatusChange,
		TenantUUID:    tenantUUID,
		LeadUUID:      codeUUID,
		ActorUserUUID: actorUUID,
		Metadata:      metadata,
	})
}

func EmitWelcomeConfigSaved(ctx context.Context, tenantUUID, codeUUID, actorUUID string, metadata map[string]any) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventWelcomeConfigSaved,
		TenantUUID:    tenantUUID,
		LeadUUID:      codeUUID,
		ActorUserUUID: actorUUID,
		Metadata:      metadata,
	})
}

func EmitWelcomeSyncPublished(ctx context.Context, tenantUUID, codeUUID, actorUUID string, metadata map[string]any) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventWelcomeSyncPublished,
		TenantUUID:    tenantUUID,
		LeadUUID:      codeUUID,
		ActorUserUUID: actorUUID,
		Metadata:      metadata,
	})
}

func EmitChannelCodeEventIngested(ctx context.Context, tenantUUID, codeUUID, actorUUID string, metadata map[string]any) {
	EmitAuditEvent(ctx, AuditEvent{
		EventType:     AuditEventChannelCodeEventIngest,
		TenantUUID:    tenantUUID,
		LeadUUID:      codeUUID,
		ActorUserUUID: actorUUID,
		Metadata:      metadata,
	})
}
