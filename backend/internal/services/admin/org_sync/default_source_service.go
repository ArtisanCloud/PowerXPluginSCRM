package org_sync

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
)

// DefaultSourceAccountService manages the default org sync source account per channel/app_type.
type DefaultSourceAccountService struct {
	accountRepo *socialrepo.AccountRepository
}

func NewDefaultSourceAccountService(accountRepo *socialrepo.AccountRepository) *DefaultSourceAccountService {
	return &DefaultSourceAccountService{accountRepo: accountRepo}
}

func (s *DefaultSourceAccountService) SetDefault(ctx context.Context, tenantUUID, accountUUID string) (*model.ChannelAccount, error) {
	if s == nil || s.accountRepo == nil {
		return nil, errors.New("default source account service not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	return s.accountRepo.SetOrgSyncDefault(ctx, tenantUUID, accountUUID)
}
