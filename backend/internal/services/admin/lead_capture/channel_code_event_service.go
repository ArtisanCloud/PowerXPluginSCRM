package lead_capture

import (
	"context"
	"errors"
	"strings"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
)

var (
	ErrChannelCodeEventServiceNotReady = errors.New("channel code event service not ready")
)

type ChannelCodeEventService struct {
	eventRepo leadrepo.ChannelCodeEventRepository
	metrics   *leadobs.Metrics
}

func NewChannelCodeEventService(eventRepo leadrepo.ChannelCodeEventRepository, metrics *leadobs.Metrics) *ChannelCodeEventService {
	return &ChannelCodeEventService{eventRepo: eventRepo, metrics: metrics}
}

func (s *ChannelCodeEventService) Ingest(ctx context.Context, input *leadmodel.ChannelCodeEvent) error {
	if s == nil || s.eventRepo == nil {
		return ErrChannelCodeEventServiceNotReady
	}
	if input == nil {
		return errors.New("channel code event is required")
	}
	input.Channel = strings.ToLower(strings.TrimSpace(input.Channel))
	input.AppType = strings.ToLower(strings.TrimSpace(input.AppType))
	input.EventType = strings.ToLower(strings.TrimSpace(input.EventType))
	if input.EventType == "" {
		input.EventType = "other"
	}
	if err := s.eventRepo.Create(ctx, input); err != nil {
		if s.metrics != nil {
			s.metrics.RecordChannelCodeEventIngest(input.Channel, input.AppType, "failed")
		}
		return err
	}
	if s.metrics != nil {
		s.metrics.RecordChannelCodeEventIngest(input.Channel, input.AppType, "success")
	}
	leadobs.EmitChannelCodeEventIngested(ctx, input.TenantUUID, input.CodeUUID, "", map[string]any{
		"channel":     input.Channel,
		"app_type":    input.AppType,
		"event_type":  input.EventType,
		"idempotency": input.IdempotencyKey,
		"external_id": input.ExternalEventID,
	})
	return nil
}

func (s *ChannelCodeEventService) GetByIdempotencyKey(ctx context.Context, tenantUUID, idempotencyKey string) (*leadmodel.ChannelCodeEvent, error) {
	if s == nil || s.eventRepo == nil {
		return nil, ErrChannelCodeEventServiceNotReady
	}
	return s.eventRepo.GetByIdempotencyKey(ctx, tenantUUID, idempotencyKey)
}
