package org_sync

import (
	"context"
	"errors"
	"sort"
	"strings"

	iamentity "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/org_sync"
	"gorm.io/gorm"
)

type MainOrgMemberView struct {
	MainMemberID      string   `json:"main_member_id"`
	MainMemberName    string   `json:"main_member_name"`
	BindingStatus     string   `json:"binding_status"`
	BoundCount        int      `json:"bound_count"`
	ChannelAccounts   []string `json:"channel_accounts"`
	ExternalMemberIDs []string `json:"external_member_ids"`
}

// MainViewService builds a main org view with source identities.
type MainViewService struct {
	db *gorm.DB
}

func NewMainViewService(db *gorm.DB) *MainViewService {
	return &MainViewService{db: db}
}

func (s *MainViewService) List(ctx context.Context, tenantUUID, query string) ([]MainOrgMemberView, error) {
	if s == nil || s.db == nil {
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
		Table(iamentity.Member{}.TableName()+" m").
		Select("m.id::text AS main_member_id, COALESCE(NULLIF(m.display_name, ''), NULLIF(u.display_name, ''), m.username) AS main_member_name").
		Joins("JOIN "+iamentity.User{}.TableName()+" u ON u.id = m.user_id").
		Where("m.tenant_uuid = ?", tenantUUID)
	if keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		memberQry = memberQry.Where("lower(COALESCE(NULLIF(m.display_name, ''), NULLIF(u.display_name, ''), m.username)) LIKE ?", like)
	}
	if err := memberQry.Scan(&members).Error; err != nil {
		return nil, err
	}
	rows := []struct {
		MainMemberID       string
		MainMemberName     string
		ChannelAccountUUID string
		ExternalMemberID   string
	}{}
	qry := s.db.WithContext(ctx).
		Table(orgmodel.MemberBinding{}.TableName()+" mb").
		Select("mb.main_member_id, COALESCE(m.display_name, u.display_name) AS main_member_name, mb.channel_account_uuid, mb.external_member_id").
		Joins("JOIN "+iamentity.Member{}.TableName()+" m ON m.id::text = mb.main_member_id").
		Joins("JOIN "+iamentity.User{}.TableName()+" u ON u.id = m.user_id").
		Where("mb.tenant_uuid = ?", tenantUUID)
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
			MainMemberID:      row.MainMemberID,
			MainMemberName:    name,
			BindingStatus:     "unbound",
			BoundCount:        0,
			ChannelAccounts:   []string{},
			ExternalMemberIDs: []string{},
		}
	}
	for _, row := range rows {
		if strings.TrimSpace(row.MainMemberID) == "" {
			continue
		}
		entry, ok := viewMap[row.MainMemberID]
		if !ok {
			entry = &MainOrgMemberView{
				MainMemberID:      row.MainMemberID,
				MainMemberName:    row.MainMemberName,
				BindingStatus:     "unbound",
				BoundCount:        0,
				ChannelAccounts:   []string{},
				ExternalMemberIDs: []string{},
			}
			viewMap[row.MainMemberID] = entry
		}
		if row.ChannelAccountUUID != "" {
			entry.ChannelAccounts = appendUnique(entry.ChannelAccounts, row.ChannelAccountUUID)
		}
		if row.ExternalMemberID != "" {
			entry.ExternalMemberIDs = appendUnique(entry.ExternalMemberIDs, row.ExternalMemberID)
		}
	}
	out := make([]MainOrgMemberView, 0, len(viewMap))
	for _, item := range viewMap {
		item.BoundCount = len(item.ExternalMemberIDs)
		if item.BoundCount > 0 {
			item.BindingStatus = "bound"
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
