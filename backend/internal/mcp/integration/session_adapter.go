package integration

import (
	"context"
	"errors"
	"strings"

	domain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	integrationService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/integration"
	"github.com/sirupsen/logrus"
)

// SessionAdapter 为 MCP 会话提供 DispatchService 适配。
type SessionAdapter struct {
	dispatch *integrationService.DispatchService
	logger   *logrus.Entry
}

// NewSessionAdapter 构造 MCP 适配器。
func NewSessionAdapter(dispatch *integrationService.DispatchService, logger *logrus.Entry) *SessionAdapter {
	if logger == nil {
		logger = logrus.WithField("component", "integration.mcp.adapter")
	}
	return &SessionAdapter{
		dispatch: dispatch,
		logger:   logger,
	}
}

// ValidateHandshake 校验 ToolScope / Session 前置条件。
func (a *SessionAdapter) ValidateHandshake(toolScope string) error {
	if strings.TrimSpace(toolScope) == "" {
		return errors.New("tool_scope is required for MCP handshake")
	}
	return nil
}

// DispatchEnvelope 通过 MCP 适配器转发请求。
func (a *SessionAdapter) DispatchEnvelope(ctx context.Context, envelope *domain.IntegrationEnvelope) (*integrationService.DispatchOutcome, error) {
	if a.dispatch == nil {
		return nil, errors.New("dispatch service unavailable")
	}
	return a.dispatch.Dispatch(ctx, "MCP", "/integration/dispatch", "CALL", envelope)
}
