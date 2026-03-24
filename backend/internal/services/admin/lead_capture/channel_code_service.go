package lead_capture

import (
	"context"
	"errors"
	"strings"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	"github.com/google/uuid"
)

var (
	ErrChannelCodeServiceNotReady = errors.New("channel code service not ready")
	ErrInvalidChannelCodePayload  = errors.New("invalid channel code payload")
	ErrChannelCodeStatusInvalid   = errors.New("invalid channel code status")
	ErrChannelCodeAlreadyExists   = errors.New("channel code already exists")
)

type ChannelCodeCreateRequest struct {
	TenantUUID         string
	Channel            string
	AppType            string
	ChannelAccountUUID string
	CodeKey            string
	DisplayName        string
	TargetType         string
	TargetID           string
	ActorUserUUID      string
}

type ChannelCodeListRequest struct {
	TenantUUID         string
	Channel            string
	AppType            string
	ChannelAccountUUID string
	Status             string
	Limit              int
}

type ChannelCodeStatusUpdateRequest struct {
	TenantUUID    string
	CodeUUID      string
	Status        string
	ActorUserUUID string
}

type ChannelCodeService struct {
	channelCodeRepo leadrepo.ChannelCodeRepository
	metrics         *leadobs.Metrics
}

func NewChannelCodeService(channelCodeRepo leadrepo.ChannelCodeRepository, metrics *leadobs.Metrics) *ChannelCodeService {
	return &ChannelCodeService{channelCodeRepo: channelCodeRepo, metrics: metrics}
}

func (s *ChannelCodeService) Create(ctx context.Context, req ChannelCodeCreateRequest) (*leadmodel.ChannelCode, error) {
	if s == nil || s.channelCodeRepo == nil {
		return nil, ErrChannelCodeServiceNotReady
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	req.AppType = strings.ToLower(strings.TrimSpace(req.AppType))
	req.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(req.ChannelAccountUUID))
	req.CodeKey = strings.TrimSpace(req.CodeKey)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.TargetType = strings.ToLower(strings.TrimSpace(req.TargetType))
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.ActorUserUUID = strings.TrimSpace(req.ActorUserUUID)
	if req.TenantUUID == "" || req.Channel == "" || req.AppType == "" || req.ChannelAccountUUID == "" || req.CodeKey == "" || req.DisplayName == "" || req.TargetType == "" || req.TargetID == "" {
		return nil, ErrInvalidChannelCodePayload
	}
	if req.ActorUserUUID == "" {
		req.ActorUserUUID = leadobs.ResolveActorUserUUID(ctx, "system")
	}
	item := &leadmodel.ChannelCode{
		CodeUUID:           uuid.NewString(),
		TenantUUID:         req.TenantUUID,
		Channel:            req.Channel,
		AppType:            req.AppType,
		ChannelAccountUUID: req.ChannelAccountUUID,
		CodeKey:            req.CodeKey,
		DisplayName:        req.DisplayName,
		TargetType:         req.TargetType,
		TargetID:           req.TargetID,
		Status:             leadmodel.ChannelCodeStatusDraft,
		CreatedBy:          req.ActorUserUUID,
		UpdatedBy:          req.ActorUserUUID,
	}
	if err := s.channelCodeRepo.Create(ctx, item); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "uq_lead_capture_channel_codes_tenant_channel_code") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrChannelCodeAlreadyExists
		}
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.RecordChannelCodeConfigChange(item.Channel, item.AppType)
	}
	leadobs.EmitChannelCodeCreated(ctx, item.TenantUUID, item.CodeUUID, req.ActorUserUUID, map[string]any{
		"channel":  item.Channel,
		"app_type": item.AppType,
		"code_key": item.CodeKey,
	})
	return item, nil
}

func (s *ChannelCodeService) Get(ctx context.Context, tenantUUID, codeUUID string) (*leadmodel.ChannelCode, error) {
	if s == nil || s.channelCodeRepo == nil {
		return nil, ErrChannelCodeServiceNotReady
	}
	return s.channelCodeRepo.GetByCodeUUID(ctx, tenantUUID, codeUUID)
}

func (s *ChannelCodeService) List(ctx context.Context, req ChannelCodeListRequest) ([]*leadmodel.ChannelCode, error) {
	if s == nil || s.channelCodeRepo == nil {
		return nil, ErrChannelCodeServiceNotReady
	}
	items, err := s.channelCodeRepo.List(ctx, req.TenantUUID, leadrepo.ChannelCodeListFilter{
		Channel:            req.Channel,
		AppType:            req.AppType,
		ChannelAccountUUID: req.ChannelAccountUUID,
		Status:             req.Status,
		Limit:              req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *ChannelCodeService) UpdateStatus(ctx context.Context, req ChannelCodeStatusUpdateRequest) (*leadmodel.ChannelCode, error) {
	if s == nil || s.channelCodeRepo == nil {
		return nil, ErrChannelCodeServiceNotReady
	}
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if req.Status != leadmodel.ChannelCodeStatusActive && req.Status != leadmodel.ChannelCodeStatusDisabled {
		return nil, ErrChannelCodeStatusInvalid
	}
	actor := strings.TrimSpace(req.ActorUserUUID)
	if actor == "" {
		actor = leadobs.ResolveActorUserUUID(ctx, "system")
	}
	item, err := s.channelCodeRepo.UpdateStatus(ctx, req.TenantUUID, req.CodeUUID, req.Status, actor)
	if err != nil {
		return nil, err
	}
	leadobs.EmitChannelCodeStatusChanged(ctx, item.TenantUUID, item.CodeUUID, actor, map[string]any{
		"status": item.Status,
	})
	return item, nil
}
