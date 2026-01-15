package integration

import (
	"context"

	domain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	"github.com/sirupsen/logrus"
)

type noopInvoker struct {
	logger *logrus.Entry
}

// NewNoopInvoker 返回默认的宿主调用占位实现。
func NewNoopInvoker(logger *logrus.Entry) HostInvoker {
	return &noopInvoker{logger: logger}
}

func (n *noopInvoker) Invoke(_ context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if n.logger != nil && envelope != nil {
		n.logger.WithFields(logrus.Fields{
			"tenant_uuid": envelope.TenantUuid,
			"tool_scope":  envelope.ToolScope,
		}).Debug("noop host invoker executed")
	}
	return &HostInvocationResult{
		Status: "accepted",
	}, nil
}
