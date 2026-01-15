package event_bridge

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/event"
)

type LocalEmitter struct {
	queue  chan event.Event
	logger *logrus.Entry
}

func NewLocalEmitter(queueSize int, logger *logrus.Entry) *LocalEmitter {
	if queueSize <= 0 {
		queueSize = 1024
	}
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	return &LocalEmitter{
		queue:  make(chan event.Event, queueSize),
		logger: logger,
	}
}

func (e *LocalEmitter) Emit(ctx context.Context, ev event.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	select {
	case e.queue <- ev:
		e.logger.WithFields(logrus.Fields{
			"topic":       string(ev.Topic),
			"tenant_uuid": ev.Meta.TenantUUID,
			"trace_id":    ev.Meta.TraceID,
			"request_id":  ev.Meta.RequestID,
		}).Debug("event_bridge local emitter emitted")
		return nil
	default:
		e.logger.WithFields(logrus.Fields{
			"topic":       string(ev.Topic),
			"tenant_uuid": ev.Meta.TenantUUID,
			"trace_id":    ev.Meta.TraceID,
			"request_id":  ev.Meta.RequestID,
		}).Warn("event_bridge local emitter queue full; dropping event")
		return nil
	}
}

func (e *LocalEmitter) Drain() []event.Event {
	var out []event.Event
	for {
		select {
		case ev := <-e.queue:
			out = append(out, ev)
		default:
			return out
		}
	}
}
