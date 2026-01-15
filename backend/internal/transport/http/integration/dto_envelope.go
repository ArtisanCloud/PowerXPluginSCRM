package integration

import (
	"fmt"
	"time"

	domain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	"github.com/google/uuid"
)

// DispatchRequest 对应 OpenAPI 定义的 IntegrationEnvelope。
type DispatchRequest struct {
	MessageID      string         `json:"message_id"`
	TraceID        string         `json:"trace_id"`
	CorrelationID  string         `json:"correlation_id"`
	TenantUuid     string         `json:"tenant_uuid"`
	ToolScope      string         `json:"tool_scope"`
	IssuedAt       string         `json:"issued_at"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	PayloadRef     string         `json:"payload_ref"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Signature      string         `json:"signature"`
}

// ToDomain 将 HTTP 请求转换为领域对象。
func (r *DispatchRequest) ToDomain() (*domain.IntegrationEnvelope, error) {
	if r == nil {
		return nil, fmt.Errorf("dispatch request is nil")
	}

	messageID, err := uuid.Parse(r.MessageID)
	if err != nil {
		return nil, fmt.Errorf("message_id: %w", err)
	}

	traceID, err := uuid.Parse(r.TraceID)
	if err != nil {
		return nil, fmt.Errorf("trace_id: %w", err)
	}

	correlationID, err := uuid.Parse(r.CorrelationID)
	if err != nil {
		return nil, fmt.Errorf("correlation_id: %w", err)
	}

	issuedAt, err := time.Parse(time.RFC3339, r.IssuedAt)
	if err != nil {
		return nil, fmt.Errorf("issued_at: %w", err)
	}

	env := &domain.IntegrationEnvelope{
		MessageID:      messageID,
		TraceID:        traceID,
		CorrelationID:  correlationID,
		TenantUuid:     r.TenantUuid,
		ToolScope:      r.ToolScope,
		IssuedAt:       issuedAt.UTC(),
		IdempotencyKey: r.IdempotencyKey,
		PayloadRef:     r.PayloadRef,
		Metadata:       r.Metadata,
		Signature:      r.Signature,
	}

	env.Normalize()
	return env, nil
}
