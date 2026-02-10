package master

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/channel"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
)

// HandleCredentialInspectionEvent is a minimal "job -> consumer" migration example:
// instead of the job writing DB/log directly, the dispatcher calls this consumer handler.
func HandleCredentialInspectionEvent(logger *logrus.Entry) func(context.Context, event.Event) error {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}

	return func(ctx context.Context, ev event.Event) error {
		var payload channel.CredentialInspectionPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return err
		}

		logger.WithFields(logrus.Fields{
			"topic":          string(ev.Topic),
			"tenant_uuid":    ev.Meta.TenantUUID,
			"trace_id":       ev.Meta.TraceID,
			"channel_id":     payload.ChannelID,
			"status":         payload.Status,
			"credentialType": payload.CredentialType,
		}).Info("credential inspection event consumed")

		return nil
	}
}
