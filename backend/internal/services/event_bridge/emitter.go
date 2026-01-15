package event_bridge

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/event"
	ebmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/event_bridge"
)

type Emitter interface {
	Emit(ctx context.Context, e event.Event) error
}

type Factory struct {
	cfg    config.EventBridgeConfig
	logger *logrus.Entry

	taskBusProvider func(logger *logrus.Entry) (Emitter, error)
}

func NewFactory(cfg config.EventBridgeConfig, logger *logrus.Entry) *Factory {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	return &Factory{
		cfg:    cfg,
		logger: logger,
	}
}

func (f *Factory) WithTaskBusProvider(provider func(logger *logrus.Entry) (Emitter, error)) *Factory {
	f.taskBusProvider = provider
	return f
}

func (f *Factory) NewEmitter() (Emitter, error) {
	local := NewLocalEmitter(f.cfg.LocalQueueSize, f.logger)

	if !f.cfg.Enabled {
		return &instrumentedEmitter{inner: local}, nil
	}

	switch f.cfg.Mode {
	case "local":
		return &instrumentedEmitter{inner: local}, nil
	case "taskbus", "dual":
		if f.taskBusProvider == nil {
			if f.cfg.FallbackToLocal {
				f.logger.Warn("event_bridge taskbus provider not configured; falling back to local emitter")
				return &instrumentedEmitter{inner: local}, nil
			}
			return nil, errors.New("event_bridge taskbus provider not configured")
		}

		taskbus, err := f.taskBusProvider(f.logger)
		if err != nil {
			if f.cfg.FallbackToLocal {
				f.logger.WithError(err).Warn("event_bridge taskbus emitter unavailable; falling back to local emitter")
				return &instrumentedEmitter{inner: local}, nil
			}
			return nil, err
		}
		if f.cfg.Mode == "dual" {
			return &instrumentedEmitter{inner: &dualEmitter{primary: taskbus, secondary: local, logger: f.logger}}, nil
		}
		return &instrumentedEmitter{inner: taskbus}, nil
	default:
		return nil, errors.New("invalid event_bridge.mode")
	}
}

type dualEmitter struct {
	primary   Emitter
	secondary Emitter
	logger    *logrus.Entry
}

func (d *dualEmitter) Emit(ctx context.Context, e event.Event) error {
	err := d.primary.Emit(ctx, e)
	if err != nil {
		if d.logger != nil {
			d.logger.WithError(err).Warn("event_bridge dual emitter primary emit failed")
		}
		return err
	}
	if d.secondary == nil {
		return nil
	}
	if err2 := d.secondary.Emit(ctx, e); err2 != nil && d.logger != nil {
		d.logger.WithError(err2).Warn("event_bridge dual emitter secondary emit failed")
	}
	return nil
}

type instrumentedEmitter struct {
	inner Emitter
}

func (e *instrumentedEmitter) Emit(ctx context.Context, ev event.Event) error {
	if e == nil || e.inner == nil {
		return errors.New("event emitter not configured")
	}

	start := time.Now()
	err := e.inner.Emit(ctx, ev)
	latencyMs := float64(time.Since(start).Milliseconds())

	result := "success"
	if err != nil {
		result = "error"
	}

	ebmetrics.RecordEmit(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), result)
	ebmetrics.ObserveLatencyMs(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), "emit", latencyMs)
	return err
}

// Drain proxies LocalEmitter.Drain when the underlying emitter supports it.
// This is primarily used by tests to assert fallback behavior without relying on concrete types.
func (e *instrumentedEmitter) Drain() []event.Event {
	if e == nil || e.inner == nil {
		return nil
	}
	if d, ok := e.inner.(interface{ Drain() []event.Event }); ok {
		return d.Drain()
	}
	return nil
}
