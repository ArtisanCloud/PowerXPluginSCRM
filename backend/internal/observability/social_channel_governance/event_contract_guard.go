package social_channel_governance

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type EventContractMeta struct {
	TenantUUID     string
	RequestID      string
	TraceID        string
	SourcePlugin   string
	OccurredAt     time.Time
	PayloadVersion string
}

type EventContractPayload map[string]any

func ValidateEventContract(topic string, meta EventContractMeta, payload EventContractPayload) error {
	if !strings.HasPrefix(topic, "powerx.") || !strings.Contains(topic, ".v") {
		return fmt.Errorf("invalid topic format: %s", topic)
	}
	if strings.TrimSpace(meta.TenantUUID) == "" {
		return errors.New("tenant_uuid is required")
	}
	if strings.TrimSpace(meta.RequestID) == "" || strings.TrimSpace(meta.TraceID) == "" {
		return errors.New("request_id and trace_id are required")
	}
	if strings.TrimSpace(meta.SourcePlugin) == "" || strings.TrimSpace(meta.PayloadVersion) == "" {
		return errors.New("source_plugin and payload_version are required")
	}
	if meta.OccurredAt.IsZero() {
		return errors.New("occurred_at is required")
	}
	for _, key := range []string{"password", "secret", "token", "access_key"} {
		if _, ok := payload[key]; ok {
			return fmt.Errorf("sensitive key %q is not allowed in event payload", key)
		}
	}
	return nil
}
