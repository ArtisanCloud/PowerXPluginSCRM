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
	"github.com/sirupsen/logrus"
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
	sourceUnitRepo    *orgrepo.SourceUnitRepository
	unitMappingRepo   *orgrepo.UnitMappingRepository
	sourceMemberRepo  *orgrepo.SourceMemberRepository
	memberMappingRepo *orgrepo.MemberMappingRepository
}

func NewMatchService(
	db *gorm.DB,
	sourceUnitRepo *orgrepo.SourceUnitRepository,
	unitMappingRepo *orgrepo.UnitMappingRepository,
	sourceMemberRepo *orgrepo.SourceMemberRepository,
	memberMappingRepo *orgrepo.MemberMappingRepository,
) *MatchService {
	return &MatchService{
		db:                db,
		sourceUnitRepo:    sourceUnitRepo,
		unitMappingRepo:   unitMappingRepo,
		sourceMemberRepo:  sourceMemberRepo,
		memberMappingRepo: memberMappingRepo,
	}
}

func (s *MatchService) SuggestMappings(ctx context.Context, tenantUUID, sourceAccountUUID, channelAccountUUID string) (*MappingSuggestions, error) {
	if s == nil || s.sourceUnitRepo == nil || s.unitMappingRepo == nil || s.sourceMemberRepo == nil || s.memberMappingRepo == nil {
		return nil, errors.New("match service dependencies not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || (sourceAccountUUID == "" && channelAccountUUID == "") {
		return nil, repository.ErrTenantUuidRequired
	}
	var (
		sourceUnits   []*model.SourceUnit
		sourceMembers []*model.SourceMember
		err           error
	)
	if channelAccountUUID != "" {
		sourceUnits, err = s.sourceUnitRepo.ListByChannelAccount(ctx, tenantUUID, channelAccountUUID, nil)
		if err != nil {
			return nil, err
		}
		sourceMembers, err = s.sourceMemberRepo.ListByChannelAccount(ctx, tenantUUID, channelAccountUUID, nil, nil)
	} else {
		sourceUnits, err = s.sourceUnitRepo.ListByAccount(ctx, tenantUUID, sourceAccountUUID, nil)
		if err != nil {
			return nil, err
		}
		sourceMembers, err = s.sourceMemberRepo.ListByAccount(ctx, tenantUUID, sourceAccountUUID, nil, nil)
	}
	if err != nil {
		return nil, err
	}
	if len(sourceUnits) == 0 && len(sourceMembers) == 0 {
		logrus.WithFields(logrus.Fields{
			"component":            "org_sync_match",
			"tenant_uuid":          tenantUUID,
			"source_account_uuid":  sourceAccountUUID,
			"channel_account_uuid": channelAccountUUID,
			"source_units_total":   0,
			"source_members_total": 0,
			"unit_suggestions":     0,
			"member_suggestions":   0,
			"message":              "no source units/members found for suggestion input",
		}).Info("org sync mapping suggestions diagnostics")
		return &MappingSuggestions{UnitSuggestions: []UnitSuggestion{}, MemberSuggestions: []MemberSuggestion{}}, nil
	}
	unitSuggestions, err := s.suggestUnitMappings(ctx, tenantUUID, sourceUnits)
	if err != nil {
		return nil, err
	}
	if len(sourceMembers) == 0 {
		return &MappingSuggestions{UnitSuggestions: unitSuggestions, MemberSuggestions: []MemberSuggestion{}}, nil
	}
	existing, err := s.existingMemberMappings(ctx, tenantUUID, sourceMembers)
	if err != nil {
		return nil, err
	}
	nameIndex, err := s.buildMainMemberNameIndex(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	memberSuggestions := make([]MemberSuggestion, 0)
	memberSkippedExisting := 0
	memberSkippedLimited := 0
	memberSkippedNoPhoneEmail := 0
	memberNoMatch := 0
	memberMatchedByPhone := 0
	memberMatchedByEmail := 0
	memberMatchedByName := 0
	for _, member := range sourceMembers {
		if member == nil {
			continue
		}
		if _, ok := existing[member.SourceMemberUUID]; ok {
			memberSkippedExisting++
			continue
		}
		if strings.EqualFold(strings.TrimSpace(member.ProfileStatus), model.ProfileStatusLimited) {
			memberSkippedLimited++
			continue
		}
		if strings.TrimSpace(member.Phone) == "" && strings.TrimSpace(member.Email) == "" {
			memberSkippedNoPhoneEmail++
		}
		matched, matchedBy, err := s.matchMainMember(ctx, tenantUUID, member, nameIndex)
		if err != nil || matched == nil {
			memberNoMatch++
			continue
		}
		if matchedBy == "phone" {
			memberMatchedByPhone++
		}
		if matchedBy == "email" {
			memberMatchedByEmail++
		}
		if matchedBy == "name_exact" {
			memberMatchedByName++
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
	logrus.WithFields(logrus.Fields{
		"component":                      "org_sync_match",
		"tenant_uuid":                    tenantUUID,
		"source_account_uuid":            sourceAccountUUID,
		"channel_account_uuid":           channelAccountUUID,
		"source_units_total":             len(sourceUnits),
		"source_members_total":           len(sourceMembers),
		"unit_suggestions":               len(unitSuggestions),
		"member_suggestions":             len(memberSuggestions),
		"member_skipped_existing":        memberSkippedExisting,
		"member_skipped_profile_limited": memberSkippedLimited,
		"member_skipped_no_phone_email":  memberSkippedNoPhoneEmail,
		"member_no_match":                memberNoMatch,
		"member_matched_by_phone":        memberMatchedByPhone,
		"member_matched_by_email":        memberMatchedByEmail,
		"member_matched_by_name":         memberMatchedByName,
	}).Info("org sync mapping suggestions diagnostics")
	return &MappingSuggestions{UnitSuggestions: unitSuggestions, MemberSuggestions: memberSuggestions}, nil
}

func (s *MatchService) suggestUnitMappings(ctx context.Context, tenantUUID string, sourceUnits []*model.SourceUnit) ([]UnitSuggestion, error) {
	if s == nil || s.db == nil || s.unitMappingRepo == nil {
		return []UnitSuggestion{}, nil
	}
	existing, err := s.existingUnitMappings(ctx, tenantUUID, sourceUnits)
	if err != nil {
		return nil, err
	}
	var mainUnits []iammodel.Department
	if err := s.db.WithContext(ctx).
		Where("tenant_uuid = ?", tenantUUID).
		Find(&mainUnits).Error; err != nil {
		return nil, err
	}
	if len(mainUnits) == 0 {
		return []UnitSuggestion{}, nil
	}
	mainByID := make(map[uint64]iammodel.Department, len(mainUnits))
	mainNameToIDs := make(map[string][]uint64, len(mainUnits))
	for _, item := range mainUnits {
		mainByID[item.ID] = item
		nameKey := normalizeOrgName(item.Name)
		if nameKey == "" {
			continue
		}
		mainNameToIDs[nameKey] = append(mainNameToIDs[nameKey], item.ID)
	}
	sourceByExternal := make(map[string]*model.SourceUnit, len(sourceUnits))
	for _, item := range sourceUnits {
		if item == nil {
			continue
		}
		externalID := strings.TrimSpace(item.ExternalUnitID)
		if externalID != "" {
			sourceByExternal[externalID] = item
		}
	}
	out := make([]UnitSuggestion, 0)
	unitSkippedExisting := 0
	unitSkippedNoName := 0
	unitNoMainNameMatch := 0
	unitAmbiguous := 0
	unitParentNarrowed := 0
	for _, source := range sourceUnits {
		if source == nil {
			continue
		}
		if _, ok := existing[source.SourceUnitUUID]; ok {
			unitSkippedExisting++
			continue
		}
		nameKey := normalizeOrgName(source.Name)
		if nameKey == "" {
			unitSkippedNoName++
			continue
		}
		candidateIDs := append([]uint64(nil), mainNameToIDs[nameKey]...)
		if len(candidateIDs) == 0 {
			unitNoMainNameMatch++
			continue
		}
		// 父级同名收敛，减少重名误匹配
		if source.ParentExternalUnitID != nil && strings.TrimSpace(*source.ParentExternalUnitID) != "" {
			parentExternalID := strings.TrimSpace(*source.ParentExternalUnitID)
			parentSource := sourceByExternal[parentExternalID]
			parentNameKey := ""
			if parentSource != nil {
				parentNameKey = normalizeOrgName(parentSource.Name)
			}
			if parentNameKey != "" {
				filtered := make([]uint64, 0, len(candidateIDs))
				for _, id := range candidateIDs {
					parentID := mainByID[id].ParentID
					if parentID == nil {
						continue
					}
					parentMain, ok := mainByID[*parentID]
					if !ok {
						continue
					}
					if normalizeOrgName(parentMain.Name) == parentNameKey {
						filtered = append(filtered, id)
					}
				}
				if len(filtered) > 0 {
					unitParentNarrowed++
					candidateIDs = filtered
				}
			}
		}
		if len(candidateIDs) != 1 {
			unitAmbiguous++
			continue
		}
		out = append(out, UnitSuggestion{
			SourceUnitUUID: source.SourceUnitUUID,
			MainUnitID:     strconv.FormatUint(candidateIDs[0], 10),
		})
	}
	logrus.WithFields(logrus.Fields{
		"component":                 "org_sync_match",
		"tenant_uuid":               tenantUUID,
		"source_units_total":        len(sourceUnits),
		"unit_suggestions":          len(out),
		"unit_skipped_existing":     unitSkippedExisting,
		"unit_skipped_no_name":      unitSkippedNoName,
		"unit_no_main_name_match":   unitNoMainNameMatch,
		"unit_ambiguous_candidates": unitAmbiguous,
		"unit_parent_narrowed":      unitParentNarrowed,
	}).Info("org sync unit suggestion diagnostics")
	return out, nil
}

func (s *MatchService) existingUnitMappings(ctx context.Context, tenantUUID string, sourceUnits []*model.SourceUnit) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	if s == nil || s.unitMappingRepo == nil || s.unitMappingRepo.DB == nil {
		return result, nil
	}
	uuids := make([]string, 0, len(sourceUnits))
	for _, unit := range sourceUnits {
		if unit == nil || strings.TrimSpace(unit.SourceUnitUUID) == "" {
			continue
		}
		uuids = append(uuids, strings.TrimSpace(unit.SourceUnitUUID))
	}
	if len(uuids) == 0 {
		return result, nil
	}
	var rows []struct {
		SourceUnitUUID string
	}
	if err := s.unitMappingRepo.DB.WithContext(ctx).
		Model(&model.UnitMapping{}).
		Select("source_unit_uuid").
		Where("tenant_uuid = ? AND source_unit_uuid IN ?", tenantUUID, uuids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		key := strings.TrimSpace(row.SourceUnitUUID)
		if key == "" {
			continue
		}
		result[key] = struct{}{}
	}
	return result, nil
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

func (s *MatchService) matchMainMember(ctx context.Context, tenantUUID string, member *model.SourceMember, nameIndex map[string][]mainMemberMatch) (*mainMemberMatch, string, error) {
	if s == nil || s.memberMappingRepo == nil || s.memberMappingRepo.DB == nil || member == nil {
		return nil, "", nil
	}
	if strings.EqualFold(strings.TrimSpace(member.ProfileStatus), model.ProfileStatusLimited) {
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
	nameKey := normalizePersonName(member.Name)
	if nameKey != "" {
		candidates := nameIndex[nameKey]
		if len(candidates) == 1 {
			item := candidates[0]
			return &item, "name_exact", nil
		}
	}
	return nil, "", nil
}

func (s *MatchService) buildMainMemberNameIndex(ctx context.Context, tenantUUID string) (map[string][]mainMemberMatch, error) {
	index := make(map[string][]mainMemberMatch)
	if s == nil || s.db == nil {
		return index, nil
	}
	rows := make([]mainMemberMatch, 0)
	if err := s.db.WithContext(ctx).
		Table(iammodel.Member{}.TableName()+" m").
		Select("m.id, COALESCE(NULLIF(m.display_name, ''), NULLIF(u.display_name, ''), m.username) AS display_name").
		Joins("JOIN "+iammodel.User{}.TableName()+" u ON u.id = m.user_id").
		Where("m.tenant_uuid = ?", tenantUUID).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		key := normalizePersonName(row.DisplayName)
		if key == "" {
			continue
		}
		index[key] = append(index[key], row)
	}
	return index, nil
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

func normalizeText(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func normalizePersonName(v string) string {
	clean := strings.TrimSpace(strings.ToLower(v))
	replacer := strings.NewReplacer(" ", "", "\u3000", "", "-", "", "_", "", ".", "", "·", "", "•", "", "（", "", "）", "", "(", "", ")", "")
	return replacer.Replace(clean)
}

func normalizeOrgName(v string) string {
	clean := normalizeText(v)
	replacer := strings.NewReplacer(" ", "", "\u3000", "", "-", "", "_", "", ".", "", "（", "", "）", "", "(", "", ")", "")
	clean = replacer.Replace(clean)
	clean = strings.TrimSuffix(clean, "有限公司")
	clean = strings.TrimSuffix(clean, "有限责任公司")
	clean = strings.TrimSuffix(clean, "集团")
	clean = strings.TrimSuffix(clean, "部门")
	clean = strings.TrimSuffix(clean, "部")
	clean = strings.TrimSuffix(clean, "中心")
	clean = strings.TrimSuffix(clean, "组")
	return strings.TrimSpace(clean)
}
