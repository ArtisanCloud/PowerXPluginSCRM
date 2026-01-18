package social_channel_governance

import (
	"context"
	"errors"
	"sort"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	SocialObs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/social_channel_governance"
)

type ChannelAccountMemberUpdateRequest struct {
	OwnerUserUUID   *string
	MemberUserUUIDs []string
}

type ChannelAccountMemberService struct {
	repo *SocialRepo.AccountRepository
}

func NewChannelAccountMemberService(repo *SocialRepo.AccountRepository) *ChannelAccountMemberService {
	return &ChannelAccountMemberService{repo: repo}
}

func (s *ChannelAccountMemberService) UpdateChannelAccountMembers(ctx context.Context, tenantUUID, accountUUID string, req ChannelAccountMemberUpdateRequest) (*model.ChannelAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("channel account member repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if req.OwnerUserUUID != nil {
		clean := strings.ToLower(strings.TrimSpace(*req.OwnerUserUUID))
		if clean == "" {
			return nil, errors.New("owner_user_uuid is required")
		}
		req.OwnerUserUUID = &clean
	}
	cleaned := normalizeUUIDList(req.MemberUserUUIDs)
	if len(cleaned) == 0 {
		return nil, errors.New("member_user_uuids is required")
	}
	account, err := s.repo.UpdateChannelAccountMembers(ctx, tenantUUID, accountUUID, req.OwnerUserUUID, cleaned)
	if err != nil {
		return nil, err
	}
	actor := ""
	if req.OwnerUserUUID != nil {
		actor = *req.OwnerUserUUID
	}
	SocialObs.EmitChannelAccountMembersChanged(
		ctx,
		tenantUUID,
		accountUUID,
		SocialObs.ResolveActorUserUUID(ctx, actor),
		req.OwnerUserUUID,
		cleaned,
	)
	return account, nil
}

func normalizeUUIDList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		clean := strings.ToLower(strings.TrimSpace(v))
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	sort.Strings(out)
	return out
}
