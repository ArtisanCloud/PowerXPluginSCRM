package social_channel_governance

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	SocialObs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/social_channel_governance"
	"gorm.io/datatypes"
)

type ChannelAccountCapabilityService struct {
	repo *SocialRepo.AccountRepository
}

func NewChannelAccountCapabilityService(repo *SocialRepo.AccountRepository) *ChannelAccountCapabilityService {
	return &ChannelAccountCapabilityService{repo: repo}
}

func (s *ChannelAccountCapabilityService) UpdateChannelAccountCapabilities(ctx context.Context, tenantUUID, accountUUID string, capabilities map[string]bool) (*model.ChannelAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("channel account capability repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if capabilities == nil {
		return nil, errors.New("capabilities is required")
	}

	account, err := s.repo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, SocialRepo.ErrAccountNotFound
	}
	if account.OwnerUserUUID == "" {
		for _, enabled := range capabilities {
			if enabled {
				return nil, errors.New("owner_user_uuid is required before enabling capabilities")
			}
		}
	}

	payload := datatypes.JSONMap{}
	for key, value := range capabilities {
		clean := strings.TrimSpace(key)
		if clean == "" {
			continue
		}
		payload[clean] = value
	}
	updated, err := s.repo.UpdateChannelAccountCapabilities(ctx, tenantUUID, accountUUID, payload)
	if err != nil {
		return nil, err
	}
	SocialObs.EmitChannelAccountCapabilitiesChanged(
		ctx,
		tenantUUID,
		accountUUID,
		SocialObs.ResolveActorUserUUID(ctx, ""),
		capabilities,
	)
	return updated, nil
}
