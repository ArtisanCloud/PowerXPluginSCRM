package runtime_ops

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	domain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/runtime_ops"
	idrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	mcpintegration "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/mcp/integration"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/mcp/stream"
	runtimeops "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/runtime_ops"
	integrationService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/integration"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SessionsHandler handles MCP session admin endpoints.
type SessionsHandler struct {
	svc     *runtimeops.MCPSessionService
	adapter *mcpintegration.SessionAdapter
	broker  *stream.Broker
	logger  *logrus.Entry
}

// NewSessionsHandler constructs handler with dependencies.
func NewSessionsHandler(deps *app.Deps) *SessionsHandler {
	var (
		db     *gorm.DB
		logger *logrus.Entry
	)
	if deps != nil {
		db = deps.DB
		logger = deps.RuntimeLogger(deps.Ctx, "mcp_session_handler", nil)
	}
	dispatch := integrationService.BuildDispatchService(deps, logger)
	return &SessionsHandler{
		svc:     runtimeops.NewMCPSessionService(db),
		adapter: mcpintegration.NewSessionAdapter(dispatch, logger),
		broker:  stream.DefaultBroker(),
		logger:  logger,
	}
}

// Register handles MCP REGISTER requests.
func (h *SessionsHandler) Register(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "session service unavailable", nil)
		return
	}
	var req registerSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid register request")
		return
	}
	session := &model.MCPSession{
		RuntimeAssignmentID: req.RuntimeAssignmentID,
		State:               req.State,
		JWTID:               req.JWTID,
		CapabilitiesHash:    req.CapabilitiesHash,
	}
	stored, err := h.svc.Register(c.Request.Context(), session)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	h.publishEvent(stored.ID, "session.registered", stored)
	contracts.ResponseSuccess(c, stored)
}

// Ack handles ACK/READY transitions.
func (h *SessionsHandler) Ack(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "session service unavailable", nil)
		return
	}
	sessionID := c.Param("sessionID")
	var req ackSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid ack payload")
		return
	}
	updated, err := h.svc.Acknowledge(c.Request.Context(), sessionID, req.State, req.CapabilitiesHash)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	h.publishEvent(updated.ID, "session.ready", updated)
	contracts.ResponseSuccess(c, updated)
}

// Heartbeat updates heartbeat metadata.
func (h *SessionsHandler) Heartbeat(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "session service unavailable", nil)
		return
	}
	sessionID := c.Param("sessionID")
	var req heartbeatRequest
	_ = c.ShouldBindJSON(&req)
	updated, err := h.svc.TouchHeartbeat(c.Request.Context(), sessionID, req.MissedHeartbeats)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	h.publishEvent(updated.ID, "session.heartbeat", map[string]any{
		"missed_heartbeats": req.MissedHeartbeats,
		"last_ping_at":      updated.LastPingAt,
	})
	contracts.ResponseSuccess(c, updated)
}

// Close terminates the session lifecycle.
func (h *SessionsHandler) Close(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "session service unavailable", nil)
		return
	}
	sessionID := c.Param("sessionID")
	var req closeRequest
	_ = c.ShouldBindJSON(&req)
	updated, err := h.svc.Close(c.Request.Context(), sessionID, req.Reason)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	h.publishEvent(updated.ID, "session.closed", map[string]any{
		"reason": req.Reason,
	})
	contracts.ResponseSuccess(c, updated)
}

// Invoke dispatches MCP envelope via SessionAdapter.
func (h *SessionsHandler) Invoke(c *gin.Context) {
	if h.adapter == nil {
		contracts.ResponseServiceUnavailable(c, "dispatch service unavailable", nil)
		return
	}
	sessionID := c.Param("sessionID")
	var req invokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid invoke payload")
		return
	}
	envelope, err := req.toDomain()
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	if err := req.ensureSessionMatches(sessionID, envelope); err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}

	outcome, err := h.adapter.DispatchEnvelope(c.Request.Context(), envelope)
	if err != nil {
		h.handleDispatchError(c, err)
		return
	}
	h.publishEvent(sessionID, "invoke.completed", map[string]any{
		"session_id":     sessionID,
		"status":         outcome.Status,
		"trace_id":       outcome.TraceID,
		"correlation_id": outcome.CorrelationID,
	})

	response := gin.H{
		"status":         outcome.Status,
		"trace_id":       outcome.TraceID,
		"correlation_id": outcome.CorrelationID,
		"latency_ms":     outcome.Latency.Milliseconds(),
		"replay":         outcome.Replay,
	}
	if len(outcome.Payload) > 0 {
		response["payload"] = json.RawMessage(outcome.Payload)
	}
	if len(outcome.Metadata) > 0 {
		response["metadata"] = outcome.Metadata
	}

	contracts.ResponseSuccess(c, response)
}

func (h *SessionsHandler) handleDispatchError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidEnvelope):
		contracts.ResponseBadRequest(c, err.Error())
	case errors.Is(err, integrationService.ErrGrantMatrixDenied):
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeForbidden, err.Error())
	case errors.Is(err, idrepo.ErrIdempotencyUnavailable):
		contracts.ResponseServiceUnavailable(c, "idempotency backend unavailable", nil)
	default:
		if h.logger != nil {
			h.logger.WithError(err).Warn("mcp invoke failed")
		}
		contracts.ResponseInternalError(c, err)
	}
}

func (h *SessionsHandler) publishEvent(sessionID, eventType string, payload interface{}) {
	if h.broker == nil || sessionID == "" {
		return
	}
	h.broker.Publish(stream.Event{
		SessionID: sessionID,
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	})
}

type registerSessionRequest struct {
	RuntimeAssignmentID string `json:"runtime_assignment_id" binding:"required"`
	State               string `json:"state"`
	JWTID               string `json:"jwt_id"`
	CapabilitiesHash    string `json:"capabilities_hash"`
}

type ackSessionRequest struct {
	State            string `json:"state"`
	CapabilitiesHash string `json:"capabilities_hash"`
}

type heartbeatRequest struct {
	MissedHeartbeats int `json:"missed_heartbeats"`
}

type closeRequest struct {
	Reason string `json:"reason"`
}

type invokeRequest struct {
	MessageID      string         `json:"message_id" binding:"required"`
	TraceID        string         `json:"trace_id" binding:"required"`
	CorrelationID  string         `json:"correlation_id" binding:"required"`
	TenantUuid     string         `json:"tenant_uuid" binding:"required"`
	ToolScope      string         `json:"tool_scope" binding:"required"`
	IssuedAt       string         `json:"issued_at" binding:"required"`
	IdempotencyKey string         `json:"idempotency_key"`
	PayloadRef     string         `json:"payload_ref" binding:"required"`
	Metadata       map[string]any `json:"metadata"`
	Signature      string         `json:"signature" binding:"required"`
}

func (r *invokeRequest) toDomain() (*domain.IntegrationEnvelope, error) {
	if r == nil {
		return nil, errors.New("invoke payload is empty")
	}
	msgID, err := uuid.Parse(r.MessageID)
	if err != nil {
		return nil, err
	}
	traceID, err := uuid.Parse(r.TraceID)
	if err != nil {
		return nil, err
	}
	correlationID, err := uuid.Parse(r.CorrelationID)
	if err != nil {
		return nil, err
	}
	issuedAt, err := time.Parse(time.RFC3339, r.IssuedAt)
	if err != nil {
		return nil, err
	}
	env := &domain.IntegrationEnvelope{
		MessageID:      msgID,
		TraceID:        traceID,
		CorrelationID:  correlationID,
		TenantUuid:     r.TenantUuid,
		ToolScope:      r.ToolScope,
		IssuedAt:       issuedAt.UTC(),
		IdempotencyKey: r.IdempotencyKey,
		PayloadRef:     r.PayloadRef,
		Metadata:       r.Metadata,
		Signature:      r.Signature,
	}
	env.Normalize()
	return env, nil
}

func (r *invokeRequest) ensureSessionMatches(sessionID string, envelope *domain.IntegrationEnvelope) error {
	if envelope == nil {
		return errors.New("envelope missing")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("session_id missing from path")
	}
	return nil
}
