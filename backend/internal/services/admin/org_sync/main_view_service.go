package org_sync

import (
	"context"
	"errors"
	"sort"
	"strings"

	iammodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	orgobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/org_sync"
	"gorm.io/gorm"
)

type MainOrgMemberView struct {
	MainMemberID     string   `json:"main_member_id"`
	MainMemberName   string   `json:"main_member_name"`
	MappingStatus    string   `json:"mapping_status"`
	MappedCount      int      `json:"mapped_count"`
	SourceAccounts   []string `json:"source_accounts"`
	SourceMemberUUID []string `json:"source_member_uuids"`
}

// MainViewService builds a main org view with source identities.
type MainViewService struct {
	db                *gorm.DB
	memberMappingRepo *orgrepo.MemberMappingRepository
}

func NewMainViewService(db *gorm.DB, memberMappingRepo *orgrepo.MemberMappingRepository) *MainViewService {
	return &MainViewService{db: db, memberMappingRepo: memberMappingRepo}
}

func (s *MainViewService) List(ctx context.Context, tenantUUID, query string) ([]MainOrgMemberView, error) {
	if s == nil || s.db == nil || s.memberMappingRepo == nil {
		return nil, errors.New("main view service dependencies not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	keyword := strings.TrimSpace(query)
	members := []struct {
		MainMemberID   string
		MainMemberName string
	}{}
	memberQry := s.db.WithContext(ctx).
		Table(iammodel.Member{}.TableName() + " m").
		Select("m.id::text AS main_member_id, COALESCE(NULLIF(m.display_name, ''), NULLIF(u.display_name, ''), m.username) AS main_member_name").
		Joins("JOIN " + iammodel.User{}.TableName() + " u ON u.id = m.user_id").
		Where("m.tenant_uuid = ?", tenantUUID)
	if keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		memberQry = memberQry.Where("lower(COALESCE(NULLIF(m.display_name, ''), NULLIF(u.display_name, ''), m.username)) LIKE ?", like)
	}
	if err := memberQry.Scan(&members).Error; err != nil {
		return nil, err
	}
	rows := []struct {
		MainMemberID      string
		MainMemberName    string
		SourceAccountUUID string
		SourceMemberUUID  string
	}{}
	qry := s.db.WithContext(ctx).
		Table(orgmodel.MemberMapping{}.TableName()+" mm").
		Select("mm.main_member_id, COALESCE(m.display_name, u.display_name) AS main_member_name, sm.source_account_uuid, mm.source_member_uuid").
		Joins("JOIN "+orgmodel.SourceMember{}.TableName()+" sm ON sm.source_member_uuid = mm.source_member_uuid").
		Joins("JOIN "+iammodel.Member{}.TableName()+" m ON m.id::text = mm.main_member_id").
		Joins("JOIN "+iammodel.User{}.TableName()+" u ON u.id = m.user_id").
		Where("mm.tenant_uuid = ? AND mm.mapping_status = ?", tenantUUID, orgmodel.MappingStatusConfirmed)
	if keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		qry = qry.Where("lower(COALESCE(m.display_name, u.display_name)) LIKE ?", like)
	}
	if err := qry.Scan(&rows).Error; err != nil {
		return nil, err
	}
	viewMap := make(map[string]*MainOrgMemberView)
	for _, row := range members {
		if strings.TrimSpace(row.MainMemberID) == "" {
			continue
		}
		name := strings.TrimSpace(row.MainMemberName)
		if name == "" {
			name = row.MainMemberID
		}
		viewMap[row.MainMemberID] = &MainOrgMemberView{
			MainMemberID:     row.MainMemberID,
			MainMemberName:   name,
			MappingStatus:    "unmapped",
			MappedCount:      0,
			SourceAccounts:   []string{},
			SourceMemberUUID: []string{},
		}
	}
	for _, row := range rows {
		if strings.TrimSpace(row.MainMemberID) == "" {
			continue
		}
		entry, ok := viewMap[row.MainMemberID]
		if !ok {
			entry = &MainOrgMemberView{
				MainMemberID:     row.MainMemberID,
				MainMemberName:   row.MainMemberName,
				MappingStatus:    "unmapped",
				MappedCount:      0,
				SourceAccounts:   []string{},
				SourceMemberUUID: []string{},
			}
			viewMap[row.MainMemberID] = entry
		}
		if row.SourceAccountUUID != "" {
			entry.SourceAccounts = appendUnique(entry.SourceAccounts, row.SourceAccountUUID)
		}
		if row.SourceMemberUUID != "" {
			entry.SourceMemberUUID = appendUnique(entry.SourceMemberUUID, row.SourceMemberUUID)
		}
	}
	out := make([]MainOrgMemberView, 0, len(viewMap))
	for _, item := range viewMap {
		item.MappedCount = len(item.SourceMemberUUID)
		if item.MappedCount > 0 {
			item.MappingStatus = "mapped"
		}
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(out[i].MainMemberName)) < strings.ToLower(strings.TrimSpace(out[j].MainMemberName))
	})
	orgobs.EmitMainViewQueried(ctx, tenantUUID, orgobs.ResolveActorUserUUID(ctx, ""), keyword, len(out))
	return out, nil
}

func appendUnique(list []string, value string) []string {
	for _, item := range list {
		if item == value {
			return list
		}
	}
	return append(list, value)
}
