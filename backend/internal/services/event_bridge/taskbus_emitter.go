package event_bridge

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/event"
)

// TaskBusProvider is a factory for producing a TaskBus-backed emitter.
// 实际实现依赖 framework runtime/host SDK，本仓库仅提供接口占位以支持 DI 与测试注入。
type TaskBusProvider interface {
	NewEmitter(logger *logrus.Entry) (Emitter, error)
}

// NewTaskBusEmitterAdapter returns a provider function compatible with Factory.WithTaskBusProvider.
// 当前为占位：如果没有注入真实 provider，将返回明确错误。
func NewTaskBusEmitterAdapter(provider TaskBusProvider) func(logger *logrus.Entry) (Emitter, error) {
	return func(logger *logrus.Entry) (Emitter, error) {
		if provider == nil {
			return nil, errors.New("taskbus provider is nil")
		}
		return provider.NewEmitter(logger)
	}
}

// Ensure TaskBus providers can be used as emitters in tests if needed.
type noopTaskBusEmitter struct {
	err error
}

func (e noopTaskBusEmitter) Emit(ctx context.Context, ev event.Event) error {
	if e.err != nil {
		return e.err
	}
	return errors.New("taskbus emitter not configured")
}
