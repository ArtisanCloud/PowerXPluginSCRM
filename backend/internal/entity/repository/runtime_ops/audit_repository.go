package runtime_ops

import "context"

// AuditRepository persists runtime audit events.
type AuditRepository interface {
	CreateEvent(ctx context.Context, evt *RuntimeAuditEvent) error
}

// RuntimeAuditEvent placeholder type for scaffolding.
type RuntimeAuditEvent struct{}
