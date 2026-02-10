package org_sync

import (
	"context"
	"errors"
	"strings"

	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/org_sync"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
)

// SourceMemberService handles source member queries.
type SourceMemberService struct {
	repo        *orgrepo.SourceMemberRepository
	profileRepo *orgrepo.SourceMemberProfileRepository
}

func NewSourceMemberService(repo *orgrepo.SourceMemberRepository, profileRepo *orgrepo.SourceMemberProfileRepository) *SourceMemberService {
	return &SourceMemberService{repo: repo, profileRepo: profileRepo}
}

func (s *SourceMemberService) List(ctx context.Context, tenantUUID, sourceAccountUUID, channelAccountUUID string, status *string, q *string, sourceUnitUUID *string, sourceUnitUUIDs []string) ([]*dto.SourceMemberView, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("source member repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || (sourceAccountUUID == "" && channelAccountUUID == "") {
		return nil, repository.ErrTenantUuidRequired
	}
	var members []*model.SourceMember
	var err error
	if len(sourceUnitUUIDs) > 0 {
		members, err = s.repo.ListByUnitIDs(ctx, tenantUUID, sourceAccountUUID, sourceUnitUUIDs, status, nil)
	} else if sourceUnitUUID != nil && strings.TrimSpace(*sourceUnitUUID) != "" {
		members, err = s.repo.ListByUnit(ctx, tenantUUID, sourceAccountUUID, *sourceUnitUUID, status, nil)
	} else if channelAccountUUID != "" {
		members, err = s.repo.ListByChannelAccount(ctx, tenantUUID, channelAccountUUID, status, nil)
	} else {
		members, err = s.repo.ListByAccount(ctx, tenantUUID, sourceAccountUUID, status, nil)
	}
	if err != nil {
		return nil, err
	}
	profiles := map[string]*model.SourceMemberProfile{}
	if s.profileRepo != nil && len(members) > 0 {
		ids := make([]string, 0, len(members))
		for _, member := range members {
			if member == nil || strings.TrimSpace(member.SourceMemberUUID) == "" {
				continue
			}
			ids = append(ids, member.SourceMemberUUID)
		}
		if len(ids) > 0 {
			rows, err := s.profileRepo.ListBySourceMembers(ctx, tenantUUID, ids)
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				if row == nil || strings.TrimSpace(row.SourceMemberUUID) == "" {
					continue
				}
				profiles[row.SourceMemberUUID] = row
			}
		}
	}
	items := make([]*dto.SourceMemberView, 0, len(members))
	keyword := ""
	if q != nil {
		keyword = strings.ToLower(strings.TrimSpace(*q))
	}
	for _, member := range members {
		if member == nil {
			continue
		}
		profile := profiles[member.SourceMemberUUID]
		name := strings.TrimSpace(member.Name)
		phone := strings.TrimSpace(member.Phone)
		email := strings.TrimSpace(member.Email)
		var profileView *dto.SourceMemberProfileView
		if profile != nil {
			if strings.TrimSpace(profile.Name) != "" {
				name = strings.TrimSpace(profile.Name)
			}
			if strings.TrimSpace(profile.Phone) != "" {
				phone = strings.TrimSpace(profile.Phone)
			}
			if strings.TrimSpace(profile.Email) != "" {
				email = strings.TrimSpace(profile.Email)
			}
			updated := profile.UpdatedAt
			profileView = &dto.SourceMemberProfileView{
				Name:             strings.TrimSpace(profile.Name),
				Phone:            strings.TrimSpace(profile.Phone),
				Email:            strings.TrimSpace(profile.Email),
				BizMail:          strings.TrimSpace(profile.BizMail),
				Position:         strings.TrimSpace(profile.Position),
				MainDepartmentID: strings.TrimSpace(profile.MainDepartmentID),
				Address:          strings.TrimSpace(profile.Address),
				AvatarURL:        strings.TrimSpace(profile.AvatarURL),
				UpdatedAt:        &updated,
			}
		}
		if name == "" {
			name = strings.TrimSpace(member.ExternalMemberID)
		}
		view := &dto.SourceMemberView{
			SourceMemberUUID:   member.SourceMemberUUID,
			TenantUUID:         member.TenantUUID,
			SourceAccountUUID:  member.SourceAccountUUID,
			ChannelAccountUUID: member.ChannelAccountUUID,
			ExternalMemberID:   member.ExternalMemberID,
			Name:               name,
			Phone:              phone,
			Email:              email,
			ProfileStatus:      member.ProfileStatus,
			Status:             member.Status,
			Profile:            profileView,
			CreatedAt:          &member.CreatedAt,
			UpdatedAt:          &member.UpdatedAt,
		}
		if keyword != "" {
			if !matchSourceMemberView(view, keyword) {
				continue
			}
		}
		items = append(items, view)
	}
	return items, nil
}

func matchSourceMemberView(view *dto.SourceMemberView, keyword string) bool {
	if view == nil {
		return false
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return true
	}
	candidates := []string{
		view.Name,
		view.Phone,
		view.Email,
		view.ExternalMemberID,
	}
	if view.Profile != nil {
		candidates = append(
			candidates,
			view.Profile.Name,
			view.Profile.Phone,
			view.Profile.Email,
			view.Profile.BizMail,
			view.Profile.Position,
			view.Profile.Address,
		)
	}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(strings.TrimSpace(candidate)), keyword) {
			return true
		}
	}
	return false
}
