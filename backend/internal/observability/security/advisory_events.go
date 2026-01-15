package security

import (
	"encoding/json"
	"time"

	secmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/security"
	"github.com/sirupsen/logrus"
)

const (
	eventVulnerabilityDetected   = "plugin.vulnerability.detected"
	eventVulnerabilityRemediated = "plugin.vulnerability.remediated"
)

// EmitAdvisoryDetected records advisory intake events for SLA tracking.
func EmitAdvisoryDetected(logger *logrus.Entry, advisory *secmodel.Advisory, metadata map[string]interface{}) {
	emitAdvisoryEvent(logger, eventVulnerabilityDetected, advisory, metadata)
}

// EmitAdvisoryRemediated records remediation completion for observability sinks.
func EmitAdvisoryRemediated(logger *logrus.Entry, advisory *secmodel.Advisory, metadata map[string]interface{}) {
	emitAdvisoryEvent(logger, eventVulnerabilityRemediated, advisory, metadata)
}

// QueueAdvisoryNotification simulates queuing outbound notifications (marketplace/webhook/email).
func QueueAdvisoryNotification(logger *logrus.Entry, advisory *secmodel.Advisory, channel string, payload map[string]interface{}) {
	if logger == nil || advisory == nil {
		return
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	payload["channel"] = channel
	payload["reference"] = advisory.Reference
	payload["advisory_id"] = advisory.ID
	payload["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	raw, err := json.Marshal(payload)
	if err != nil {
		logger.WithError(err).Warn("failed to marshal advisory notification payload")
		return
	}
	logger.WithField("advisory_notification", string(raw)).Info("queued vulnerability advisory notification")
}

func emitAdvisoryEvent(logger *logrus.Entry, event string, advisory *secmodel.Advisory, metadata map[string]interface{}) {
	if logger == nil || advisory == nil {
		return
	}
	payload := map[string]interface{}{
		"event":        event,
		"advisory_id":  advisory.ID,
		"reference":    advisory.Reference,
		"severity":     advisory.Severity,
		"status":       advisory.Status,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"sla_deadline": nil,
	}
	if advisory.SlaDeadline != nil {
		payload["sla_deadline"] = advisory.SlaDeadline.UTC().Format(time.RFC3339)
	}
	if advisory.PatchedInVersion != "" {
		payload["patched_in_version"] = advisory.PatchedInVersion
	}
	for k, v := range metadata {
		payload[k] = v
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		logger.WithError(err).Warn("failed to marshal advisory event payload")
		return
	}
	logger.WithField("advisory_event", string(raw)).Info("vulnerability advisory event")
}
