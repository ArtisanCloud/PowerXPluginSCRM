package event

import (
	"encoding/json"
	"time"
)

type Topic string

type Meta struct {
	TenantUUID     string    `json:"tenant_uuid" yaml:"tenant_uuid"`
	RequestID      string    `json:"request_id" yaml:"request_id"`
	SourcePlugin   string    `json:"source_plugin" yaml:"source_plugin"`
	TraceID        string    `json:"trace_id" yaml:"trace_id"`
	OccurredAt     time.Time `json:"occurred_at" yaml:"occurred_at"`
	PayloadVersion string    `json:"payload_version" yaml:"payload_version"`
}

type Event struct {
	Topic   Topic           `json:"topic" yaml:"topic"`
	Meta    Meta            `json:"meta" yaml:"meta"`
	Payload json.RawMessage `json:"payload" yaml:"payload"`
}

type Subscription struct {
	Topic Topic `json:"topic" yaml:"topic"`
}
