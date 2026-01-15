package runtime_ops

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/runtime_ops"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	runtimeRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/runtime_ops"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MCPSessionService coordinates REGISTER/ACK/CAPABILITY_SYNC flow.
type MCPSessionService struct {
	repo   *runtimeRepo.MCPSessionRepository
	audits *repository.BaseRepository[model.RuntimeAuditEvent]
}

// NewMCPSessionService constructs the MCP session service.
func NewMCPSessionService(db *gorm.DB) *MCPSessionService {
	service := &MCPSessionService{}
	if db != nil {
		service.repo = runtimeRepo.NewMCPSessionRepository(db)
		service.audits = repository.NewBaseRepository[model.RuntimeAuditEvent](db)
	}
	return service
}

// Register is a placeholder for MCP session registration logic.

func (s *MCPSessionService) Register(ctx context.Context, session *model.MCPSession) (*model.MCPSession, error) {
	if session == nil {
		return nil, gorm.ErrInvalidData
	}
	tenantUUID, err := authx.RequireTenantUUID(ctx)
	if err != nil {
		return nil, err
	}
	if s.repo == nil {
		return nil, gorm.ErrInvalidDB
	}
	assignmentID := strings.TrimSpace(session.RuntimeAssignmentID)
	if assignmentID == "" {
		return nil, gorm.ErrInvalidData
	}
	if strings.TrimSpace(session.ID) == "" {
		session.ID = uuid.NewString()
	}
	session.RuntimeAssignmentID = assignmentID
	session.TenantUuid = tenantUUID
	session.State = strings.ToUpper(strings.TrimSpace(session.State))
	if session.State == "" {
		session.State = "REGISTERED"
	}
	session.JWTID = strings.TrimSpace(session.JWTID)
	session.CapabilitiesHash = strings.TrimSpace(session.CapabilitiesHash)
	session.MissedHeartbeats = 0
	now := time.Now().UTC()
	session.LastPingAt = &model.DBTime{Time: now}
	stored, err := s.repo.Create(ctx, session)
	if err != nil {
		return nil, err
	}
	_ = s.RecordLifecycle(ctx, stored.TenantUuid, "mcp.session.registered", stored)
	return stored, nil
}

// RecordAudit writes an audit event for session lifecycle.
func (s *MCPSessionService) RecordAudit(ctx context.Context, evt *model.RuntimeAuditEvent) error {
	if evt == nil {
		return gorm.ErrInvalidData
	}
	if s.audits == nil {
		return gorm.ErrInvalidDB
	}
	_, err := s.audits.Create(ctx, evt)
	return err
}

// Acknowledge transitions a session into READY or custom state and updates capability hash.
func (s *MCPSessionService) Acknowledge(ctx context.Context, sessionID, state, capabilitiesHash string) (*model.MCPSession, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, gorm.ErrInvalidData
	}
	if _, err := authx.RequireTenantUUID(ctx); err != nil {
		return nil, err
	}
	if s.repo == nil {
		return nil, gorm.ErrInvalidDB
	}
	fields := map[string]interface{}{
		"state": strings.ToUpper(strings.TrimSpace(state)),
	}
	if strings.TrimSpace(state) == "" {
		fields["state"] = "READY"
	}
	if strings.TrimSpace(capabilitiesHash) != "" {
		fields["capabilities_hash"] = capabilitiesHash
	}
	if _, ok := fields["capabilities_hash"]; !ok {
		fields["updated_at"] = time.Now().UTC()
	}
	updated, err := s.repo.UpdateFields(ctx, sessionID, fields)
	if err != nil {
		return nil, err
	}
	_ = s.RecordLifecycle(ctx, updated.TenantUuid, "mcp.session.ack", updated)
	return updated, nil
}

// TouchHeartbeat records last ping timestamp and missed heartbeat count.
func (s *MCPSessionService) TouchHeartbeat(ctx context.Context, sessionID string, missed int) (*model.MCPSession, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, gorm.ErrInvalidData
	}
	if _, err := authx.RequireTenantUUID(ctx); err != nil {
		return nil, err
	}
	if s.repo == nil {
		return nil, gorm.ErrInvalidDB
	}
	if missed < 0 {
		missed = 0
	}
	fields := map[string]interface{}{
		"missed_heartbeats": missed,
		"last_ping_at":      time.Now().UTC(),
	}
	updated, err := s.repo.UpdateFields(ctx, sessionID, fields)
	if err != nil {
		return nil, err
	}
	if missed > 0 {
		_ = s.RecordLifecycle(ctx, updated.TenantUuid, "mcp.session.heartbeat.missed", map[string]any{
			"session_id":        sessionID,
			"missed_heartbeats": missed,
		})
	}
	return updated, nil
}

// Close marks the session as closed and records audit event.
func (s *MCPSessionService) Close(ctx context.Context, sessionID, reason string) (*model.MCPSession, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, gorm.ErrInvalidData
	}
	if _, err := authx.RequireTenantUUID(ctx); err != nil {
		return nil, err
	}
	if s.repo == nil {
		return nil, gorm.ErrInvalidDB
	}
	fields := map[string]interface{}{
		"state":     "CLOSED",
		"closed_at": time.Now().UTC(),
	}
	updated, err := s.repo.UpdateFields(ctx, sessionID, fields)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"session_id": sessionID,
	}
	if strings.TrimSpace(reason) != "" {
		payload["reason"] = reason
	}
	_ = s.RecordLifecycle(ctx, updated.TenantUuid, "mcp.session.closed", payload)
	return updated, nil
}

// RecordLifecycle helper wraps RecordAudit with JSON payload.
func (s *MCPSessionService) RecordLifecycle(ctx context.Context, tenantUUID, event string, payload interface{}) error {
	if s == nil || s.audits == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte("{}")
	}
	return s.RecordAudit(ctx, &model.RuntimeAuditEvent{
		PluginID:   app.PluginID,
		TenantUUID: tenantUUID,
		EventType:  event,
		Payload:    string(data),
		OccurredAt: time.Now().UTC(),
	})
}
