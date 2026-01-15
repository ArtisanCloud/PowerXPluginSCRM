package security

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const DefaultAuditLogPath = "logs/audit.log"

// AuditWriter appends structured audit events to a designated writer.
type AuditWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewFileAuditWriter creates (or opens) the audit log file and returns a writer.
func NewFileAuditWriter(path string) (*AuditWriter, error) {
	if path == "" {
		path = DefaultAuditLogPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create audit log directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, fmt.Errorf("open audit log file: %w", err)
	}
	return &AuditWriter{writer: f}, nil
}

// NewAuditWriter wraps an arbitrary writer for audit output.
func NewAuditWriter(w io.Writer) *AuditWriter {
	return &AuditWriter{writer: w}
}

// Emit writes the supplied audit payload as a JSON line.
func (w *AuditWriter) Emit(event AuditEvent) error {
	if w == nil || w.writer == nil {
		return fmt.Errorf("audit writer not configured")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	data := struct {
		Timestamp  time.Time              `json:"timestamp"`
		Event      string                 `json:"event"`
		TenantUuid string                 `json:"tenant_uuid,omitempty"`
		Actor      string                 `json:"actor,omitempty"`
		Metadata   map[string]interface{} `json:"metadata,omitempty"`
	}{
		Timestamp:  event.Timestamp,
		Event:      event.Event,
		TenantUuid: event.TenantUuid,
		Actor:      event.Actor,
		Metadata:   event.Metadata,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.writer.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}

// Close releases the underlying writer if it implements io.Closer.
func (w *AuditWriter) Close() error {
	if closer, ok := w.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// AuditEvent describes the structured payload persisted by AuditWriter.
type AuditEvent struct {
	Event      string                 `json:"event"`
	TenantUuid string                 `json:"tenant_uuid,omitempty"`
	Actor      string                 `json:"actor,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// EmitLifecycleSuccess helper for lifecycle events.
func (w *AuditWriter) EmitLifecycleSuccess(tenantID, eventType, actor string, metadata map[string]interface{}) error {
	return w.Emit(AuditEvent{
		Event:      eventType,
		TenantUuid: tenantID,
		Actor:      actor,
		Metadata:   metadata,
	})
}
