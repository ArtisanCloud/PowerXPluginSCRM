package lead_capture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrChannelCodeEventServiceNotReady = errors.New("channel code event service not ready")
	ErrInvalidChannelCodeEventPayload  = errors.New("invalid channel code event payload")
	ErrChannelCodeNotFound             = errors.New("channel code not found")
	ErrChannelCodeDisabled             = errors.New("channel code is disabled")
)

type ChannelCodeEventIngestRequest struct {
	Channel            string
	ChannelAccountUUID string
	CodeKey            string
	ExternalEventID    string
	EventType          string
	OccurredAt         string
	Payload            map[string]any
}

type ChannelCodeEventIngestResult struct {
	Event         *leadmodel.ChannelCodeEvent
	Created       bool
	IdempotentHit bool
	LeadUUID      string
	IsPrimary     bool
}

type ChannelCodeEventsQueryResult struct {
	CodeUUID string                        `json:"code_uuid"`
	Events   []*leadmodel.ChannelCodeEvent `json:"events"`
	Stats    ChannelCodeEventStats         `json:"stats"`
}

type ChannelCodeEventStats struct {
	TouchTotal  int64 `json:"touch_total"`
	IntakeTotal int64 `json:"intake_total"`
	DedupTotal  int64 `json:"dedup_total"`
}

type ChannelCodeEventService struct {
	eventRepo        leadrepo.ChannelCodeEventRepository
	channelCodeRepo  leadrepo.ChannelCodeRepository
	accountRepo      *socialrepo.AccountRepository
	attributionSvc   *AttributionService
	metrics          *leadobs.Metrics
	mu               sync.Mutex
	dedupHitCounters map[string]int64
}

func NewChannelCodeEventService(
	eventRepo leadrepo.ChannelCodeEventRepository,
	channelCodeRepo leadrepo.ChannelCodeRepository,
	accountRepo *socialrepo.AccountRepository,
	attributionSvc *AttributionService,
	metrics *leadobs.Metrics,
) *ChannelCodeEventService {
	return &ChannelCodeEventService{
		eventRepo:        eventRepo,
		channelCodeRepo:  channelCodeRepo,
		accountRepo:      accountRepo,
		attributionSvc:   attributionSvc,
		metrics:          metrics,
		dedupHitCounters: map[string]int64{},
	}
}

func (s *ChannelCodeEventService) IngestWebhook(ctx context.Context, req ChannelCodeEventIngestRequest) (*ChannelCodeEventIngestResult, error) {
	if s == nil || s.eventRepo == nil || s.channelCodeRepo == nil || s.accountRepo == nil || s.attributionSvc == nil {
		return nil, ErrChannelCodeEventServiceNotReady
	}
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	req.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(req.ChannelAccountUUID))
	req.CodeKey = strings.TrimSpace(req.CodeKey)
	req.ExternalEventID = strings.TrimSpace(req.ExternalEventID)
	req.EventType = strings.ToLower(strings.TrimSpace(req.EventType))
	if req.EventType == "" {
		req.EventType = "other"
	}
	if req.Channel == "" || req.ChannelAccountUUID == "" || req.CodeKey == "" || req.ExternalEventID == "" {
		return nil, ErrInvalidChannelCodeEventPayload
	}

	account, err := s.accountRepo.FindByUUID(ctx, req.ChannelAccountUUID)
	if err != nil {
		if errors.Is(err, socialrepo.ErrAccountNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidChannelCodeEventPayload
		}
		return nil, err
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(account.TenantUuid))
	appType := strings.ToLower(strings.TrimSpace(account.AppType))
	if tenantUUID == "" || appType == "" {
		return nil, ErrInvalidChannelCodeEventPayload
	}

	code, err := s.channelCodeRepo.GetByCodeKey(ctx, tenantUUID, req.Channel, req.CodeKey)
	if err != nil {
		if errors.Is(err, leadrepo.ErrRecordNotFound) {
			return nil, ErrChannelCodeNotFound
		}
		return nil, err
	}
	if strings.EqualFold(code.Status, leadmodel.ChannelCodeStatusDisabled) {
		return nil, ErrChannelCodeDisabled
	}

	occurredAt := time.Now().UTC()
	if raw := strings.TrimSpace(req.OccurredAt); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			return nil, ErrInvalidChannelCodeEventPayload
		}
		occurredAt = parsed.UTC()
	}

	idempotencyKey := buildChannelCodeEventIdempotencyKey(tenantUUID, req.Channel, req.ChannelAccountUUID, req.ExternalEventID)
	existing, err := s.eventRepo.GetByIdempotencyKey(ctx, tenantUUID, idempotencyKey)
	if err == nil && existing != nil {
		leadUUID := ""
		isPrimary := false
		attrs, _ := s.attributionSvc.ListByEventUUID(ctx, tenantUUID, existing.EventUUID)
		if len(attrs) > 0 && attrs[0] != nil {
			leadUUID = attrs[0].LeadUUID
			isPrimary = attrs[0].IsPrimary
		}
		s.recordDedupHit(tenantUUID, existing.CodeUUID)
		return &ChannelCodeEventIngestResult{
			Event:         existing,
			Created:       false,
			IdempotentHit: true,
			LeadUUID:      leadUUID,
			IsPrimary:     isPrimary,
		}, nil
	}
	if err != nil && !errors.Is(err, leadrepo.ErrRecordNotFound) {
		return nil, err
	}

	payloadJSON, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, ErrInvalidChannelCodeEventPayload
	}
	event := &leadmodel.ChannelCodeEvent{
		EventUUID:          uuid.NewString(),
		TenantUUID:         tenantUUID,
		Channel:            req.Channel,
		AppType:            appType,
		ChannelAccountUUID: req.ChannelAccountUUID,
		CodeUUID:           code.CodeUUID,
		ExternalEventID:    req.ExternalEventID,
		EventType:          req.EventType,
		IdempotencyKey:     idempotencyKey,
		OccurredAt:         occurredAt,
		Payload:            payloadJSON,
	}
	if err := s.eventRepo.Create(ctx, event); err != nil {
		if s.metrics != nil {
			s.metrics.RecordChannelCodeEventIngest(req.Channel, appType, "failed")
		}
		return nil, err
	}
	leadUUID, isPrimary, err := s.attributionSvc.AttributeByEvent(ctx, event, code)
	if err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.RecordChannelCodeEventIngest(req.Channel, appType, "success")
	}
	leadobs.EmitChannelCodeEventIngested(ctx, event.TenantUUID, event.CodeUUID, "", map[string]any{
		"channel":     event.Channel,
		"app_type":    event.AppType,
		"event_type":  event.EventType,
		"idempotency": event.IdempotencyKey,
		"external_id": event.ExternalEventID,
	})
	return &ChannelCodeEventIngestResult{
		Event:         event,
		Created:       true,
		IdempotentHit: false,
		LeadUUID:      leadUUID,
		IsPrimary:     isPrimary,
	}, nil
}

func (s *ChannelCodeEventService) ListByCodeUUID(ctx context.Context, tenantUUID, codeUUID string, limit int) (*ChannelCodeEventsQueryResult, error) {
	if s == nil || s.eventRepo == nil || s.attributionSvc == nil {
		return nil, ErrChannelCodeEventServiceNotReady
	}
	events, err := s.eventRepo.ListByCodeUUID(ctx, tenantUUID, codeUUID, limit)
	if err != nil {
		return nil, err
	}
	touchTotal, err := s.eventRepo.CountByCodeUUID(ctx, tenantUUID, codeUUID)
	if err != nil {
		return nil, err
	}
	intakeTotal, err := s.attributionSvc.CountByCodeUUID(ctx, tenantUUID, codeUUID)
	if err != nil {
		return nil, err
	}
	return &ChannelCodeEventsQueryResult{
		CodeUUID: codeUUID,
		Events:   events,
		Stats: ChannelCodeEventStats{
			TouchTotal:  touchTotal,
			IntakeTotal: intakeTotal,
			DedupTotal:  s.getDedupHit(tenantUUID, codeUUID),
		},
	}, nil
}

func (s *ChannelCodeEventService) recordDedupHit(tenantUUID, codeUUID string) {
	if s == nil {
		return
	}
	key := fmt.Sprintf("%s:%s", strings.ToLower(strings.TrimSpace(tenantUUID)), strings.ToLower(strings.TrimSpace(codeUUID)))
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dedupHitCounters == nil {
		s.dedupHitCounters = map[string]int64{}
	}
	s.dedupHitCounters[key]++
}

func (s *ChannelCodeEventService) getDedupHit(tenantUUID, codeUUID string) int64 {
	if s == nil {
		return 0
	}
	key := fmt.Sprintf("%s:%s", strings.ToLower(strings.TrimSpace(tenantUUID)), strings.ToLower(strings.TrimSpace(codeUUID)))
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dedupHitCounters[key]
}

func buildChannelCodeEventIdempotencyKey(tenantUUID, channel, channelAccountUUID, externalEventID string) string {
	return strings.ToLower(strings.TrimSpace(tenantUUID)) + ":" +
		strings.ToLower(strings.TrimSpace(channel)) + ":" +
		strings.ToLower(strings.TrimSpace(channelAccountUUID)) + ":" +
		strings.TrimSpace(externalEventID)
}
