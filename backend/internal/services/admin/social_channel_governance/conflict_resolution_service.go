package social_channel_governance

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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

func (s *ConflictResolutionService) List(ctx context.Context, tenantUUID, domain, status string, limit int) ([]model.SyncConflict, error) {
	if s == nil || s.repo == nil {
		return []model.SyncConflict{}, nil
	}
	return s.repo.ListConflicts(ctx, tenantUUID, domain, status, limit)
}

func (s *ConflictResolutionService) Replay(ctx context.Context, tenantUUID, conflictUUID, resolvedBy string) (*model.SyncConflict, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("conflict service unavailable")
	}
	out, err := s.repo.ReplayConflict(ctx, tenantUUID, conflictUUID, resolvedBy)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, socialrepo.ErrSyncConflictNotFound
		}
		return nil, err
	}
	return out, nil
}
