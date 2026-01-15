package operations

import (
	"context"

	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	"github.com/sirupsen/logrus"
)

// loggingIncidentDispatcher emits incident events to structured logs for auditability.
type loggingIncidentDispatcher struct {
	log *logrus.Entry
}

// NewLoggingIncidentDispatcher builds a dispatcher backed by the provided logger.
func NewLoggingIncidentDispatcher(log *logrus.Entry) IncidentDispatcher {
	if log == nil {
		return noopIncidentDispatcher{}
	}
	return &loggingIncidentDispatcher{log: log}
}

// DispatchIncidentEvent satisfies the IncidentDispatcher interface.
func (d *loggingIncidentDispatcher) DispatchIncidentEvent(ctx context.Context, eventType string, incident *opmodels.Incident, payload map[string]any) error {
	if d == nil || d.log == nil || incident == nil {
		return nil
	}
	fields := logrus.Fields{
		"incident_id": incident.ID,
		"event_type":  eventType,
		"severity":    incident.Severity,
	}
	for k, v := range payload {
		if _, exists := fields[k]; exists {
			fields[k+"_payload"] = v
			continue
		}
		fields[k] = v
	}
	d.log.WithContext(ctx).WithFields(fields).Info("operations incident event")
	return nil
}
