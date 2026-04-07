package social_channel_governance

import (
	"context"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"gorm.io/datatypes"
)

const DefaultConflictStrategy = "remote_first"

type ConflictResolutionService struct {
	repo *socialrepo.SyncFoundationRepository
}

func NewConflictResolutionService(repo *socialrepo.SyncFoundationRepository) *ConflictResolutionService {
	return &ConflictResolutionService{repo: repo}
}

func (s *ConflictResolutionService) Enqueue(ctx context.Context, tenantUUID, domain, entityType, entityKey string, localValue, remoteValue map[string]any) error {
	if s == nil || s.repo == nil {
		return nil
	}
	conflict := &model.SyncConflict{
		TenantUUID:         strings.TrimSpace(strings.ToLower(tenantUUID)),
		Domain:             strings.TrimSpace(domain),
		EntityType:         strings.TrimSpace(entityType),
		EntityKey:          strings.TrimSpace(entityKey),
		ResolutionStrategy: DefaultConflictStrategy,
		Status:             "open",
		LocalValue:         datatypes.JSONMap(localValue),
		RemoteValue:        datatypes.JSONMap(remoteValue),
	}
	return s.repo.SaveConflict(ctx, conflict)
}
