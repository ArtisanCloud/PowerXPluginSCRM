package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// IntegrationEnvelope 表示统一的 A2A 消息封装。
type IntegrationEnvelope struct {
	MessageID      uuid.UUID
	TraceID        uuid.UUID
	CorrelationID  uuid.UUID
	TenantUuid     string
	ToolScope      string
	IssuedAt       time.Time
	IdempotencyKey string
	PayloadRef     string
	Metadata       map[string]any
	Signature      string
}

// EnvelopeValidationError 描述单个字段的校验失败原因。
type EnvelopeValidationError struct {
	Field  string
	Reason string
}

func (e EnvelopeValidationError) Error() string {
	if e.Field == "" {
		return e.Reason
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// EnvelopeValidationErrors 聚合多个校验错误。
type EnvelopeValidationErrors []EnvelopeValidationError

// Error 实现 error 接口。
func (errs EnvelopeValidationErrors) Error() string {
	if len(errs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, "; ")
}

// HasError 检测是否包含错误。
func (errs EnvelopeValidationErrors) HasError() bool {
	return len(errs) > 0
}

// IsURL 判断 PayloadRef 是否为 URL。
func (e *IntegrationEnvelope) IsURL() bool {
	ref := strings.TrimSpace(e.PayloadRef)
	if ref == "" {
		return false
	}
	return strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://")
}

// Validate 检查 Envelope 关键字段是否满足约束。
// payloadThresholdBytes - 若 payload_ref 为内联 JSON，则限制其长度。
func (e *IntegrationEnvelope) Validate(now time.Time, payloadThresholdBytes int64) error {
	var errs EnvelopeValidationErrors

	if e.MessageID == uuid.Nil {
		errs = append(errs, EnvelopeValidationError{Field: "message_id", Reason: "must be a valid UUID"})
	}
	if e.TraceID == uuid.Nil {
		errs = append(errs, EnvelopeValidationError{Field: "trace_id", Reason: "must be a valid UUID"})
	}
	if e.CorrelationID == uuid.Nil {
		errs = append(errs, EnvelopeValidationError{Field: "correlation_id", Reason: "must be a valid UUID"})
	}
	if strings.TrimSpace(e.TenantUuid) == "" {
		errs = append(errs, EnvelopeValidationError{Field: "tenant_uuid", Reason: "required"})
	}
	if strings.TrimSpace(e.ToolScope) == "" {
		errs = append(errs, EnvelopeValidationError{Field: "tool_scope", Reason: "required"})
	}
	if e.IssuedAt.IsZero() {
		errs = append(errs, EnvelopeValidationError{Field: "issued_at", Reason: "required"})
	} else {
		if now.IsZero() {
			now = time.Now().UTC()
		}
		if e.IssuedAt.After(now.Add(5 * time.Minute)) {
			errs = append(errs, EnvelopeValidationError{Field: "issued_at", Reason: "cannot be more than 5 minutes in the future"})
		}
		if e.IssuedAt.Before(now.Add(-24 * time.Hour)) {
			errs = append(errs, EnvelopeValidationError{Field: "issued_at", Reason: "is too old"})
		}
	}

	if strings.TrimSpace(e.PayloadRef) == "" {
		errs = append(errs, EnvelopeValidationError{Field: "payload_ref", Reason: "required"})
	} else {
		ref := strings.TrimSpace(e.PayloadRef)
		if strings.HasPrefix(ref, "{") || strings.HasPrefix(ref, "[") {
			if !json.Valid([]byte(ref)) {
				errs = append(errs, EnvelopeValidationError{Field: "payload_ref", Reason: "inline payload must be valid JSON"})
			}
			if payloadThresholdBytes > 0 && int64(len([]byte(ref))) > payloadThresholdBytes {
				errs = append(errs, EnvelopeValidationError{Field: "payload_ref", Reason: "inline payload exceeds configured threshold"})
			}
		} else {
			if u, err := url.Parse(ref); err != nil || u.Scheme == "" || u.Host == "" {
				errs = append(errs, EnvelopeValidationError{Field: "payload_ref", Reason: "must be valid JSON or HTTPS URL"})
			} else if u.Scheme != "https" {
				errs = append(errs, EnvelopeValidationError{Field: "payload_ref", Reason: "URL must use HTTPS"})
			}
		}
	}

	if strings.TrimSpace(e.Signature) == "" {
		errs = append(errs, EnvelopeValidationError{Field: "signature", Reason: "required"})
	}

	if len(e.IdempotencyKey) > 255 {
		errs = append(errs, EnvelopeValidationError{Field: "idempotency_key", Reason: "must be <= 255 characters"})
	}

	if errs.HasError() {
		return errs
	}
	return nil
}

// Normalize 将部分字段标准化。
func (e *IntegrationEnvelope) Normalize() {
	e.TenantUuid = strings.TrimSpace(e.TenantUuid)
	e.ToolScope = strings.TrimSpace(e.ToolScope)
	e.IdempotencyKey = strings.TrimSpace(e.IdempotencyKey)
	e.PayloadRef = strings.TrimSpace(e.PayloadRef)
	e.Signature = strings.TrimSpace(e.Signature)

	if e.Metadata == nil {
		e.Metadata = map[string]any{}
	}
}

// ErrInvalidEnvelope 表示 Envelope 校验失败的通用错误。
var ErrInvalidEnvelope = errors.New("integration envelope invalid")

// ValidateOrError 执行 Normalize + Validate，并在失败时包装通用错误。
func (e *IntegrationEnvelope) ValidateOrError(now time.Time, payloadThresholdBytes int64) error {
	e.Normalize()
	if err := e.Validate(now, payloadThresholdBytes); err != nil {
		return errors.Join(ErrInvalidEnvelope, err)
	}
	return nil
}
