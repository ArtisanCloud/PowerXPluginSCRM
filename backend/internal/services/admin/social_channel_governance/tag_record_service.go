package social_channel_governance

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
)

type TagRecordService struct {
	repo *socialrepo.TagRecordRepository
}

func NewTagRecordService(repo *socialrepo.TagRecordRepository) *TagRecordService {
	return &TagRecordService{repo: repo}
}

func (s *TagRecordService) List(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	limit int,
) ([]model.SyncTagRecord, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("tag record service unavailable")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	channelAccountUUID = strings.TrimSpace(strings.ToLower(channelAccountUUID))
	return s.repo.ListByChannel(ctx, tenantUUID, channelAccountUUID, limit)
}
