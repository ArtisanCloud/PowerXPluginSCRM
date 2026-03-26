package acquisition

import (
	"context"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
)

type GroupLiveCodeService struct {
	repo acqrepo.GroupLiveCodeRepository
}

func NewGroupLiveCodeService(repo acqrepo.GroupLiveCodeRepository) *GroupLiveCodeService {
	return &GroupLiveCodeService{repo: repo}
}

func (s *GroupLiveCodeService) List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return []*acqmodel.GroupLiveCode{}, nil
	}
	items, err := s.repo.List(ctx, tenantUUID, limit)
	if err != nil {
		return nil, err
	}
	return items, nil
}
