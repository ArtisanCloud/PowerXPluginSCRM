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
	ErrChannelCodeServiceNotReady = errors.New("channel code service not ready")
)

type ChannelCodeService struct {
	channelCodeRepo leadrepo.ChannelCodeRepository
	metrics         *leadobs.Metrics
}

func NewChannelCodeService(channelCodeRepo leadrepo.ChannelCodeRepository, metrics *leadobs.Metrics) *ChannelCodeService {
	return &ChannelCodeService{channelCodeRepo: channelCodeRepo, metrics: metrics}
}

func (s *ChannelCodeService) Create(ctx context.Context, input *leadmodel.ChannelCode) error {
	if s == nil || s.channelCodeRepo == nil {
		return ErrChannelCodeServiceNotReady
	}
	if input == nil {
		return errors.New("channel code is required")
	}
	input.Channel = strings.ToLower(strings.TrimSpace(input.Channel))
	input.AppType = strings.ToLower(strings.TrimSpace(input.AppType))
	if input.Status == "" {
		input.Status = leadmodel.ChannelCodeStatusDraft
	}
	if err := s.channelCodeRepo.Create(ctx, input); err != nil {
		return err
	}
	if s.metrics != nil {
		s.metrics.RecordChannelCodeConfigChange(input.Channel, input.AppType)
	}
	leadobs.EmitChannelCodeCreated(ctx, input.TenantUUID, input.CodeUUID, input.CreatedBy, map[string]any{
		"channel":  input.Channel,
		"app_type": input.AppType,
	})
	return nil
}

func (s *ChannelCodeService) Get(ctx context.Context, tenantUUID, codeUUID string) (*leadmodel.ChannelCode, error) {
	if s == nil || s.channelCodeRepo == nil {
		return nil, ErrChannelCodeServiceNotReady
	}
	return s.channelCodeRepo.GetByCodeUUID(ctx, tenantUUID, codeUUID)
}
