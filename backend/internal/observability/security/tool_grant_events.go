package security

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
)

// EmitToolGrantEvent writes lifecycle events to structured log.
func EmitToolGrantEvent(logger *logrus.Entry, event string, tenantID string, metadata map[string]interface{}) {
	if logger == nil {
		return
	}
	payload := map[string]interface{}{
		"event":     event,
		"tenant_uuid": tenantID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range metadata {
		payload[k] = v
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		logger.WithError(err).Warn("failed to marshal toolgrant event metadata")
		return
	}
	logger.WithField("toolgrant_event", string(raw)).Info("toolgrant lifecycle event")
}
