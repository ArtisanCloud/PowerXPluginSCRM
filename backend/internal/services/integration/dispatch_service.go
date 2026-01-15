package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	domain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	idrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	obsintegration "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/integration"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
)

// HostInvoker 定义调用宿主接口的抽象。
type HostInvoker interface {
	Invoke(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error)
}

// HostInvocationResult 描述宿主调用的结果。
type HostInvocationResult struct {
	Status   string          `json:"status"`
	Payload  json.RawMessage `json:"payload,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}

// DispatchOutcome 封装 service 返回给 Handler 的结果。
type DispatchOutcome struct {
	Status        string
	TraceID       string
	CorrelationID string
	Replay        bool
	Latency       time.Duration
	Payload       json.RawMessage
	Metadata      map[string]any
}

// DispatchService orchestrates envelope validation, idempotency, GrantMatrix checks and host invocation.
type DispatchService struct {
	cfg            *config.Config
	grants         *GrantMatrixService
	idempotency    *idrepo.IdempotencyRepository
	invoker        HostInvoker
	logger         *logrus.Entry
	now            func() time.Time
	payloadLimiter int64
}

// NewDispatchService builds a new service instance.
func NewDispatchService(
	cfg *config.Config,
	grants *GrantMatrixService,
	idempotency *idrepo.IdempotencyRepository,
	invoker HostInvoker,
	logger *logrus.Entry,
) *DispatchService {
	if logger == nil {
		logger = logrus.WithField("component", "integration.dispatch_service")
	}
	payloadThreshold := int64(1 << 20) // default 1 MB
	if cfg != nil {
		payloadThreshold = cfg.IntegrationPayloadThreshold()
	}

	return &DispatchService{
		cfg:            cfg,
		grants:         grants,
		idempotency:    idempotency,
		invoker:        invoker,
		logger:         logger,
		now:            time.Now,
		payloadLimiter: payloadThreshold,
	}
}

// Dispatch validates and executes an integration request.
func (s *DispatchService) Dispatch(ctx context.Context, channel, resource, action string, envelope *domain.IntegrationEnvelope) (*DispatchOutcome, error) {
	start := s.now().UTC()
	channel = strings.ToUpper(strings.TrimSpace(channel))

	if envelope == nil {
		return nil, errors.New("envelope cannot be nil")
	}

	if err := envelope.ValidateOrError(start, s.payloadLimiter); err != nil {
		obsintegration.RecordEnvelope(channel, "invalid")
		return nil, err
	}

	if s.grants == nil {
		obsintegration.RecordEnvelope(channel, "error")
		return nil, errors.New("grant matrix service not configured")
	}

	if _, err := s.grants.EnsureAccess(ctx, envelope.ToolScope, channel, resource, action); err != nil {
		obsintegration.RecordEnvelope(channel, "denied")
		return nil, err
	}

	outcome := &DispatchOutcome{
		Status:        "accepted",
		TraceID:       envelope.TraceID.String(),
		CorrelationID: envelope.CorrelationID.String(),
	}

	replay, err := s.handleIdempotencyClaim(ctx, channel, resource, envelope, outcome)
	if err != nil {
		obsintegration.RecordEnvelope(channel, "error")
		return nil, err
	}
	outcome.Replay = replay

	if replay {
		obsintegration.RecordEnvelope(channel, "accepted")
		return outcome, nil
	}

	if s.invoker == nil {
		obsintegration.RecordEnvelope(channel, "error")
		return nil, errors.New("dispatch invoker not configured")
	}

	result, err := s.invoker.Invoke(ctx, envelope)
	if err != nil {
		obsintegration.RecordEnvelope(channel, "error")
		return nil, err
	}
	if result != nil {
		if len(result.Payload) > 0 {
			outcome.Payload = result.Payload
		}
		if len(result.Metadata) > 0 {
			outcome.Metadata = result.Metadata
		}
	}

	if result != nil && strings.TrimSpace(result.Status) != "" {
		outcome.Status = result.Status
	}
	outcome.Latency = time.Since(start)
	if outcome.Latency < 0 {
		outcome.Latency = 0
	}

	if err := s.persistIdempotentResponse(ctx, channel, envelope, outcome, result); err != nil {
		s.logger.WithError(err).
			WithField("tenant_uuid", envelope.TenantUuid).
			WithField("tool_scope", envelope.ToolScope).
			Warn("failed to persist idempotent response")
	}

	obsintegration.RecordEnvelope(channel, "accepted")
	return outcome, nil
}

func (s *DispatchService) handleIdempotencyClaim(
	ctx context.Context,
	channel, resource string,
	envelope *domain.IntegrationEnvelope,
	outcome *DispatchOutcome,
) (bool, error) {
	if envelope.IdempotencyKey == "" || s.idempotency == nil {
		return false, nil
	}

	key := buildIdempotencyKey(envelope.TenantUuid, envelope.ToolScope, envelope.IdempotencyKey)
	metadata := datatypes.JSONMap{
		"trace_id":       envelope.TraceID.String(),
		"correlation_id": envelope.CorrelationID.String(),
		"channel":        channel,
	}

	record := &domain.IdempotencyRecord{
		Key:         key,
		TenantUuid:  envelope.TenantUuid,
		Scope:       envelope.ToolScope,
		Operation:   fmt.Sprintf("%s:%s", channel, normalizeResource(resource)),
		PayloadHash: hashPayloadRef(envelope.PayloadRef),
		Metadata:    metadata,
	}

	res, err := s.idempotency.Claim(ctx, record)
	if err != nil {
		obsintegration.RecordIdempotency("error")
		return false, err
	}
	if res == nil || res.Record == nil {
		obsintegration.RecordIdempotency("error")
		return false, errors.New("idempotency claim returned no record")
	}

	if res.Status == idrepo.ClaimStatusExisting {
		obsintegration.RecordIdempotency("hit")
		outcome.Replay = true
		outcome.Status = getStoredStatus(res.Record.Response, outcome.Status)
		if latency := getStoredLatency(res.Record.Response); latency > 0 {
			outcome.Latency = latency
		}
		s.logger.WithFields(logrus.Fields{
			"idempotency_key": envelope.IdempotencyKey,
			"tenant_uuid":     envelope.TenantUuid,
			"tool_scope":      envelope.ToolScope,
			"channel":         channel,
		}).Info("integration dispatch replayed from idempotency record")
		return true, nil
	}

	obsintegration.RecordIdempotency("miss")
	return false, nil
}

func (s *DispatchService) persistIdempotentResponse(
	ctx context.Context,
	channel string,
	envelope *domain.IntegrationEnvelope,
	outcome *DispatchOutcome,
	result *HostInvocationResult,
) error {
	if envelope.IdempotencyKey == "" || s.idempotency == nil {
		return nil
	}

	payload := map[string]any{
		"status":     outcome.Status,
		"latency_ms": outcome.Latency.Milliseconds(),
		"channel":    channel,
	}
	if result != nil {
		if len(result.Payload) > 0 {
			payload["payload"] = json.RawMessage(result.Payload)
		}
		if len(result.Metadata) > 0 {
			payload["metadata"] = result.Metadata
		}
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, saveErr := s.idempotency.SaveResponse(
		ctx,
		buildIdempotencyKey(envelope.TenantUuid, envelope.ToolScope, envelope.IdempotencyKey),
		json.RawMessage(raw),
		map[string]any{
			"status": outcome.Status,
		},
	)
	return saveErr
}

func buildIdempotencyKey(tenantID, scope, key string) string {
	return strings.Join([]string{strings.TrimSpace(tenantID), strings.TrimSpace(scope), strings.TrimSpace(key)}, "|")
}

func hashPayloadRef(payloadRef string) string {
	sum := sha256.Sum256([]byte(payloadRef))
	return hex.EncodeToString(sum[:])
}

func getStoredStatus(data datatypes.JSON, defaultStatus string) string {
	if len(data) == 0 {
		return defaultStatus
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return defaultStatus
	}
	if status, ok := payload["status"].(string); ok && status != "" {
		return status
	}
	return defaultStatus
}

func getStoredLatency(data datatypes.JSON) time.Duration {
	if len(data) == 0 {
		return 0
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0
	}
	if v, ok := payload["latency_ms"]; ok {
		switch value := v.(type) {
		case float64:
			return time.Duration(value) * time.Millisecond
		case int64:
			return time.Duration(value) * time.Millisecond
		case json.Number:
			if parsed, err := value.Int64(); err == nil {
				return time.Duration(parsed) * time.Millisecond
			}
		}
	}
	return 0
}
