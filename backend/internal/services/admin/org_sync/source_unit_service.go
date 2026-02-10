package org_sync

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
)

// SourceUnitService handles source unit queries.
type SourceUnitService struct {
	repo *orgrepo.SourceUnitRepository
}

func NewSourceUnitService(repo *orgrepo.SourceUnitRepository) *SourceUnitService {
	return &SourceUnitService{repo: repo}
}

func (s *SourceUnitService) List(ctx context.Context, tenantUUID, sourceAccountUUID, channelAccountUUID string, status *string) ([]*model.SourceUnit, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("source unit repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || (sourceAccountUUID == "" && channelAccountUUID == "") {
		return nil, repository.ErrTenantUuidRequired
	}
	if channelAccountUUID != "" {
		return s.repo.ListByChannelAccount(ctx, tenantUUID, channelAccountUUID, status)
	}
	return s.repo.ListByAccount(ctx, tenantUUID, sourceAccountUUID, status)
}
