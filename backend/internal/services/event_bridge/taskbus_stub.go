package event_bridge

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/event"
)

// TaskBusStub is an in-process TaskBus-like emitter used for integration tests.
// It supports publish -> dispatch -> consumer handlers without external dependencies.
type TaskBusStub struct {
	mu       sync.RWMutex
	handlers map[event.Topic][]func(context.Context, event.Event) error
	logger   *logrus.Entry
}

func NewTaskBusStub(logger *logrus.Entry) *TaskBusStub {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	return &TaskBusStub{
		handlers: map[event.Topic][]func(context.Context, event.Event) error{},
		logger:   logger,
	}
}

func (b *TaskBusStub) Subscribe(topic event.Topic, handler func(context.Context, event.Event) error) {
	if b == nil || handler == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], handler)
}

func (b *TaskBusStub) Emit(ctx context.Context, ev event.Event) error {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	handlers := append([]func(context.Context, event.Event) error(nil), b.handlers[ev.Topic]...)
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, ev); err != nil {
			if b.logger != nil {
				b.logger.WithError(err).Warn("taskbus stub handler failed")
			}
			return err
		}
	}
	return nil
}
