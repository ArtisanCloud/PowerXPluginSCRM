package capability

import (
	"context"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/sirupsen/logrus"
)

const (
	EventTaskCreated           = "capability.review.task.created"
	EventTaskUpdated           = "capability.review.task.updated"
	EventTaskEscalated         = "capability.review.task.escalated"
	EventCommentAdded          = "capability.review.comment.added"
	EventCapabilityResubmitted = "capability.review.capability.resubmitted"
)

// Event represents a structured review notification.
type Event struct {
	Type         string
	CapabilityID string
	TaskID       string
	Status       string
	Message      string
	Deadline     time.Time
	Metadata     map[string]any
	Payload      map[string]any
	Channels     []string
}

// EmitReviewEvent sends the event to the shared logger for downstream sinks.
func EmitReviewEvent(ctx context.Context, log *logrus.Entry, evt Event) {
	if log == nil {
		log = logger.WithField("component", "capability_review_events")
	}
	fields := logrus.Fields{
		"event":         evt.Type,
		"capability_id": evt.CapabilityID,
		"task_id":       evt.TaskID,
		"status":        evt.Status,
		"message":       evt.Message,
	}
	if !evt.Deadline.IsZero() {
		fields["deadline"] = evt.Deadline
	}
	for k, v := range evt.Metadata {
		fields[k] = v
	}
	if len(evt.Channels) > 0 {
		fields["channels"] = evt.Channels
	}
	if len(evt.Payload) > 0 {
		fields["payload"] = evt.Payload
	}
	if ctx != nil {
		if reqID := ctx.Value("request_id"); reqID != nil {
			fields["trace_id"] = reqID
		}
	}
	log.WithFields(fields).Info("capability review event")
}
