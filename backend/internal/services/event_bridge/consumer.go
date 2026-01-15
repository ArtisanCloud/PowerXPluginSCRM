package event_bridge

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/event"
	ebmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/event_bridge"
)

type HandlerFunc func(ctx context.Context, ev event.Event) error

type Dispatcher struct {
	mu          sync.RWMutex
	handlers    map[event.Topic][]HandlerFunc
	idempotency *IdempotencyFilter
	logger      *logrus.Entry
}

func NewDispatcher(idempotency *IdempotencyFilter, logger *logrus.Entry) *Dispatcher {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	if idempotency == nil {
		idempotency = NewIdempotencyFilter(10000, logger)
	}
	return &Dispatcher{
		handlers:    map[event.Topic][]HandlerFunc{},
		idempotency: idempotency,
		logger:      logger,
	}
}

func (d *Dispatcher) Register(topic event.Topic, handler HandlerFunc) {
	if d == nil || handler == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[topic] = append(d.handlers[topic], handler)
}

func (d *Dispatcher) Dispatch(ctx context.Context, ev event.Event) error {
	if d == nil {
		return errors.New("dispatcher is nil")
	}

	if d.idempotency != nil && d.idempotency.SeenBefore(ev) {
		if d.logger != nil {
			d.logger.WithFields(logrus.Fields{
				"topic":       string(ev.Topic),
				"tenant_uuid": ev.Meta.TenantUUID,
				"trace_id":    ev.Meta.TraceID,
			}).Info("duplicate event skipped by idempotency filter")
		}
		return nil
	}

	d.mu.RLock()
	hs := append([]HandlerFunc(nil), d.handlers[ev.Topic]...)
	d.mu.RUnlock()

	start := time.Now()
	result := "success"
	for _, h := range hs {
		if err := h(ctx, ev); err != nil {
			result = "error"
			ebmetrics.RecordConsume(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), result)
			ebmetrics.ObserveLatencyMs(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), "consume", float64(time.Since(start).Milliseconds()))
			return err
		}
	}
	ebmetrics.RecordConsume(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), result)
	ebmetrics.ObserveLatencyMs(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), "consume", float64(time.Since(start).Milliseconds()))
	return nil
}
