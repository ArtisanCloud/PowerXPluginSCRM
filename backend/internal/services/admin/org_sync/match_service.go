package org_sync

import (
	"context"
	"errors"
	"strconv"
	"strings"

	iammodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	"gorm.io/gorm"
)

type UnitSuggestion struct {
	SourceUnitUUID string `json:"source_unit_uuid"`
	MainUnitID     string `json:"main_unit_id"`
}

type MemberSuggestion struct {
	SourceMemberUUID string `json:"source_member_uuid"`
	SourceName       string `json:"source_name"`
	Phone            string `json:"phone,omitempty"`
	Email            string `json:"email,omitempty"`
	MainMemberID     string `json:"main_member_id"`
	MainMemberName   string `json:"main_member_name"`
	MatchedBy        string `json:"matched_by"`
}

type MappingSuggestions struct {
	UnitSuggestions   []UnitSuggestion   `json:"unit_suggestions"`
	MemberSuggestions []MemberSuggestion `json:"member_suggestions"`
}

// MatchService provides mapping suggestions.
type MatchService struct {
	db                *gorm.DB
	sourceMemberRepo  *orgrepo.SourceMemberRepository
	memberMappingRepo *orgrepo.MemberMappingRepository
}

func NewMatchService(db *gorm.DB, sourceMemberRepo *orgrepo.SourceMemberRepository, memberMappingRepo *orgrepo.MemberMappingRepository) *MatchService {
	return &MatchService{db: db, sourceMemberRepo: sourceMemberRepo, memberMappingRepo: memberMappingRepo}
}

func (s *MatchService) SuggestMappings(ctx context.Context, tenantUUID, sourceAccountUUID string) (*MappingSuggestions, error) {
	if s == nil || s.sourceMemberRepo == nil || s.memberMappingRepo == nil {
		return nil, errors.New("match service dependencies not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	sourceMembers, err := s.sourceMemberRepo.ListByAccount(ctx, tenantUUID, sourceAccountUUID, nil, nil)
	if err != nil {
		return nil, err
	}
	if len(sourceMembers) == 0 {
		return &MappingSuggestions{UnitSuggestions: []UnitSuggestion{}, MemberSuggestions: []MemberSuggestion{}}, nil
	}
	existing, err := s.existingMemberMappings(ctx, tenantUUID, sourceMembers)
	if err != nil {
		return nil, err
	}
	memberSuggestions := make([]MemberSuggestion, 0)
	for _, member := range sourceMembers {
		if member == nil {
			continue
		}
		if _, ok := existing[member.SourceMemberUUID]; ok {
			continue
		}
		matched, matchedBy, err := s.matchMainMember(ctx, tenantUUID, member)
		if err != nil || matched == nil {
			continue
		}
		memberSuggestions = append(memberSuggestions, MemberSuggestion{
			SourceMemberUUID: member.SourceMemberUUID,
			SourceName:       member.Name,
			Phone:            member.Phone,
			Email:            member.Email,
			MainMemberID:     strconv.FormatUint(matched.ID, 10),
			MainMemberName:   matched.DisplayName,
			MatchedBy:        matchedBy,
		})
	}
	return &MappingSuggestions{UnitSuggestions: []UnitSuggestion{}, MemberSuggestions: memberSuggestions}, nil
}

func (s *MatchService) existingMemberMappings(ctx context.Context, tenantUUID string, sourceMembers []*model.SourceMember) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	if s == nil || s.memberMappingRepo == nil || s.memberMappingRepo.DB == nil {
		return result, nil
	}
	uuids := make([]string, 0, len(sourceMembers))
	for _, member := range sourceMembers {
		if member == nil || strings.TrimSpace(member.SourceMemberUUID) == "" {
			continue
		}
		uuids = append(uuids, strings.TrimSpace(member.SourceMemberUUID))
	}
	if len(uuids) == 0 {
		return result, nil
	}
	var rows []struct {
		SourceMemberUUID string
	}
	if err := s.memberMappingRepo.DB.WithContext(ctx).
		Model(&model.MemberMapping{}).
		Select("source_member_uuid").
		Where("tenant_uuid = ? AND source_member_uuid IN ?", tenantUUID, uuids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if strings.TrimSpace(row.SourceMemberUUID) == "" {
			continue
		}
		result[row.SourceMemberUUID] = struct{}{}
	}
	return result, nil
}

type mainMemberMatch struct {
	ID          uint64
	DisplayName string
}

func (s *MatchService) matchMainMember(ctx context.Context, tenantUUID string, member *model.SourceMember) (*mainMemberMatch, string, error) {
	if s == nil || s.memberMappingRepo == nil || s.memberMappingRepo.DB == nil || member == nil {
		return nil, "", nil
	}
	db := s.memberMappingRepo.DB.WithContext(ctx)
	phone := strings.TrimSpace(member.Phone)
	if phone != "" {
		matched, err := findMemberByPhone(db, tenantUUID, phone)
		if err != nil {
			return nil, "", err
		}
		if matched != nil {
			return matched, "phone", nil
		}
	}
	email := strings.ToLower(strings.TrimSpace(member.Email))
	if email != "" {
		matched, err := findMemberByEmail(db, tenantUUID, email)
		if err != nil {
			return nil, "", err
		}
		if matched != nil {
			return matched, "email", nil
		}
	}
	return nil, "", nil
}

func findMemberByPhone(db *gorm.DB, tenantUUID, phone string) (*mainMemberMatch, error) {
	if db == nil {
		return nil, nil
	}
	var out mainMemberMatch
	query := db.Table(iammodel.Member{}.TableName()+" m").
		Select("m.id, COALESCE(m.display_name, u.display_name) AS display_name").
		Joins("JOIN "+iammodel.User{}.TableName()+" u ON u.id = m.user_id").
		Where("m.tenant_uuid = ? AND u.phone = ?", tenantUUID, phone)
	if err := query.Limit(1).Scan(&out).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if out.ID == 0 {
		return nil, nil
	}
	return &out, nil
}

func findMemberByEmail(db *gorm.DB, tenantUUID, email string) (*mainMemberMatch, error) {
	if db == nil {
		return nil, nil
	}
	var out mainMemberMatch
	query := db.Table(iammodel.Member{}.TableName()+" m").
		Select("m.id, COALESCE(m.display_name, u.display_name) AS display_name").
		Joins("JOIN "+iammodel.User{}.TableName()+" u ON u.id = m.user_id").
		Where("m.tenant_uuid = ? AND lower(u.email) = ?", tenantUUID, strings.ToLower(email))
	if err := query.Limit(1).Scan(&out).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if out.ID == 0 {
		return nil, nil
	}
	return &out, nil
}
