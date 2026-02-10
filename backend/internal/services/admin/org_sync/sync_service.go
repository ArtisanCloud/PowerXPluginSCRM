package org_sync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	socialModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	orgobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/org_sync"
	orgdriver "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync/driver"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SyncService handles source org sync triggers.
type SyncService struct {
	repo              *orgrepo.SourceAccountRepository
	unitRepo          *orgrepo.SourceUnitRepository
	memberRepo        *orgrepo.SourceMemberRepository
	unitMappingRepo   *orgrepo.UnitMappingRepository
	memberMappingRepo *orgrepo.MemberMappingRepository
	logRepo           *orgrepo.SyncLogRepository
	driverRegistry    *orgdriver.Registry
	publisher         fwwsbus.Publisher
}

func NewSyncService(
	repo *orgrepo.SourceAccountRepository,
	unitRepo *orgrepo.SourceUnitRepository,
	memberRepo *orgrepo.SourceMemberRepository,
	unitMappingRepo *orgrepo.UnitMappingRepository,
	memberMappingRepo *orgrepo.MemberMappingRepository,
	logRepo *orgrepo.SyncLogRepository,
	publisher fwwsbus.Publisher,
) *SyncService {
	registry := orgdriver.NewRegistry()
	registry.Register("wechat", "wecom", &orgdriver.WeComDriver{})
	return &SyncService{
		repo:              repo,
		unitRepo:          unitRepo,
		memberRepo:        memberRepo,
		unitMappingRepo:   unitMappingRepo,
		memberMappingRepo: memberMappingRepo,
		logRepo:           logRepo,
		driverRegistry:    registry,
		publisher:         publisher,
	}
}

func (s *SyncService) TriggerSync(ctx context.Context, tenantUUID, sourceAccountUUID string) (*model.SourceAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("source account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	account, err := s.repo.GetByUUID(ctx, tenantUUID, sourceAccountUUID)
	if err != nil {
		if errors.Is(err, orgrepo.ErrSourceAccountNotFound) {
			account, err = s.ensureSourceAccount(ctx, tenantUUID, sourceAccountUUID)
		}
		if err != nil {
			return nil, err
		}
	}
	channelAccount, err := s.loadChannelAccount(ctx, tenantUUID, sourceAccountUUID)
	if err != nil {
		return nil, err
	}
	driverContext := orgdriver.AccountContext{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: channelAccount.AccountUUID,
		ChannelCode:        channelAccount.ChannelCode,
		AppType:            channelAccount.AppType,
		AccountID:          channelAccount.AccountID,
		DisplayName:        channelAccount.DisplayName,
		Credentials:        credentialsToMap(channelAccount.Credentials),
	}
	drv, err := s.resolveDriver(driverContext)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	syncLogUUID := ""
	if s.logRepo != nil && s.logRepo.DB != nil {
		logRecord := &model.SyncLog{
			TenantUUID:         tenantUUID,
			SourceAccountUUID:  sourceAccountUUID,
			ChannelAccountUUID: channelAccount.AccountUUID,
			Status:             model.SyncStatusRunning,
			Message:            "同步中",
			Stage:              "init",
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := s.logRepo.DB.WithContext(ctx).Create(logRecord).Error; err == nil {
			syncLogUUID = logRecord.SyncLogUUID
		}
	}
	updateSyncLog := func(updates map[string]any) {
		if s.logRepo == nil || syncLogUUID == "" {
			return
		}
		if updates == nil {
			return
		}
		if _, ok := updates["updated_at"]; !ok {
			updates["updated_at"] = time.Now().UTC()
		}
		_ = s.logRepo.UpdateByUUID(ctx, tenantUUID, syncLogUUID, updates)
	}
	progressReporter := func(current, total int, stage string) {
		percent := 0
		if total > 0 {
			percent = int(float64(current) / float64(total) * 100.0)
			if percent > 100 {
				percent = 100
			}
			if percent < 0 {
				percent = 0
			}
		}
		updateSyncLog(map[string]any{
			"progress_total":   total,
			"progress_current": current,
			"progress_percent": percent,
			"stage":            stage,
		})
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, stage, "同步中", current, total, percent, 0)
	}
	status := model.SyncStatusSuccess
	message := "同步完成"
	var (
		unitPayloads   []orgdriver.SourceUnitDTO
		memberPayloads []orgdriver.SourceMemberDTO
	)
	startTime := time.Now()
	s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "init", "同步中", 0, 0, 0, 0)
	updateSyncLog(map[string]any{
		"status": model.SyncStatusRunning,
		"stage":  "fetch_units",
	})
	s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "fetch_units", "同步中", 0, 0, 0, 0)
	unitPayloads, err = drv.FetchUnits(ctx, driverContext)
	if err != nil {
		status = model.SyncStatusFailed
		message = err.Error()
	}
	if status == model.SyncStatusSuccess {
		updateSyncLog(map[string]any{"stage": "fetch_members"})
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "fetch_members", "同步中", 0, 0, 0, 0)
		memberCtx := orgdriver.WithProgressReporter(ctx, progressReporter)
		memberPayloads, err = drv.FetchMembers(memberCtx, driverContext)
		if err != nil {
			status = model.SyncStatusFailed
			message = err.Error()
		}
	}
	duration := time.Since(startTime).Milliseconds()
	if status == model.SyncStatusSuccess {
		updateSyncLog(map[string]any{"stage": "persist"})
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "persist", "同步中", 0, 0, 0, 0)
	}
	updates := map[string]any{
		"last_sync_at":      now,
		"last_sync_status":  status,
		"last_sync_message": message,
		"updated_at":        now,
	}
	unitTotal := int64(len(unitPayloads))
	memberTotal := int64(len(memberPayloads))
	unitNew, unitUpdated := s.classifyUnitChanges(ctx, tenantUUID, sourceAccountUUID, unitPayloads)
	memberNew, memberUpdated := s.classifyMemberChanges(ctx, tenantUUID, sourceAccountUUID, memberPayloads)
	unitPending, unitConflict, memberPending, memberConflict := s.collectMappingStats(ctx, tenantUUID, sourceAccountUUID, channelAccount.AccountUUID)
	logUpdates := map[string]any{
		"status":           status,
		"message":          message,
		"units_total":      int(unitTotal),
		"members_total":    int(memberTotal),
		"units_new":        int(unitNew),
		"members_new":      int(memberNew),
		"units_updated":    int(unitUpdated),
		"members_updated":  int(memberUpdated),
		"units_pending":    int(unitPending),
		"members_pending":  int(memberPending),
		"units_conflict":   int(unitConflict),
		"members_conflict": int(memberConflict),
		"duration_ms":      duration,
		"updated_at":       now,
	}
	if status == model.SyncStatusSuccess {
		logUpdates["progress_total"] = int(memberTotal)
		logUpdates["progress_current"] = int(memberTotal)
		logUpdates["progress_percent"] = 100
		logUpdates["stage"] = "done"
	} else {
		logUpdates["stage"] = "failed"
	}
	if err := s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.SourceAccount{}).
			Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return orgrepo.ErrSourceAccountNotFound
		}
		if status == model.SyncStatusSuccess {
			if err := upsertSourceUnits(ctx, tx, tenantUUID, sourceAccountUUID, channelAccount.AccountUUID, unitPayloads); err != nil {
				return err
			}
			if err := upsertSourceMembers(ctx, tx, tenantUUID, sourceAccountUUID, channelAccount.AccountUUID, memberPayloads); err != nil {
				return err
			}
			if err := upsertSourceMemberUnits(ctx, tx, tenantUUID, sourceAccountUUID, channelAccount.AccountUUID, memberPayloads); err != nil {
				return err
			}
		}
		if s.logRepo != nil {
			if syncLogUUID != "" {
				if err := tx.Model(&model.SyncLog{}).
					Where("tenant_uuid = ? AND sync_log_uuid = ?", tenantUUID, syncLogUUID).
					Updates(logUpdates).Error; err != nil {
					return err
				}
			} else {
				logRecord := &model.SyncLog{
					TenantUUID:         tenantUUID,
					SourceAccountUUID:  sourceAccountUUID,
					ChannelAccountUUID: channelAccount.AccountUUID,
					Status:             status,
					Message:            message,
					UnitsTotal:         int(unitTotal),
					MembersTotal:       int(memberTotal),
					UnitsNew:           int(unitNew),
					MembersNew:         int(memberNew),
					UnitsUpdated:       int(unitUpdated),
					MembersUpdated:     int(memberUpdated),
					UnitsPending:       int(unitPending),
					MembersPending:     int(memberPending),
					UnitsConflict:      int(unitConflict),
					MembersConflict:    int(memberConflict),
					DurationMs:         duration,
					Stage:              "done",
					ProgressTotal:      int(memberTotal),
					ProgressCurrent:    int(memberTotal),
					ProgressPercent:    100,
					CreatedAt:          now,
					UpdatedAt:          now,
				}
				if err := tx.Create(logRecord).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if status == model.SyncStatusSuccess {
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, status, "done", message, int(memberTotal), int(memberTotal), 100, duration)
	} else {
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, status, "failed", message, 0, int(memberTotal), 0, duration)
	}
	account.LastSyncAt = &now
	account.LastSyncStatus = status
	account.LastSyncMessage = message
	account.UpdatedAt = now
	orgobs.EmitSourceSyncTriggered(ctx, tenantUUID, orgobs.ResolveActorUserUUID(ctx, ""), sourceAccountUUID)
	return account, nil
}

func (s *SyncService) collectMappingStats(ctx context.Context, tenantUUID, sourceAccountUUID, channelAccountUUID string) (int64, int64, int64, int64) {
	if s == nil {
		return 0, 0, 0, 0
	}
	var (
		unitsPending    int64
		unitsConflict   int64
		membersPending  int64
		membersConflict int64
	)
	if s.unitMappingRepo != nil {
		if channelAccountUUID != "" {
			if count, err := s.unitMappingRepo.CountByStatusForChannelAccount(ctx, tenantUUID, channelAccountUUID, model.MappingStatusPending); err == nil {
				unitsPending = count
			}
			if count, err := s.unitMappingRepo.CountByStatusForChannelAccount(ctx, tenantUUID, channelAccountUUID, model.MappingStatusConflict); err == nil {
				unitsConflict = count
			}
		} else {
			if count, err := s.unitMappingRepo.CountByStatusForAccount(ctx, tenantUUID, sourceAccountUUID, model.MappingStatusPending); err == nil {
				unitsPending = count
			}
			if count, err := s.unitMappingRepo.CountByStatusForAccount(ctx, tenantUUID, sourceAccountUUID, model.MappingStatusConflict); err == nil {
				unitsConflict = count
			}
		}
	}
	if s.memberMappingRepo != nil {
		if channelAccountUUID != "" {
			if count, err := s.memberMappingRepo.CountByStatusForChannelAccount(ctx, tenantUUID, channelAccountUUID, model.MappingStatusPending); err == nil {
				membersPending = count
			}
			if count, err := s.memberMappingRepo.CountByStatusForChannelAccount(ctx, tenantUUID, channelAccountUUID, model.MappingStatusConflict); err == nil {
				membersConflict = count
			}
		} else {
			if count, err := s.memberMappingRepo.CountByStatusForAccount(ctx, tenantUUID, sourceAccountUUID, model.MappingStatusPending); err == nil {
				membersPending = count
			}
			if count, err := s.memberMappingRepo.CountByStatusForAccount(ctx, tenantUUID, sourceAccountUUID, model.MappingStatusConflict); err == nil {
				membersConflict = count
			}
		}
	}
	return unitsPending, unitsConflict, membersPending, membersConflict
}

func (s *SyncService) resolveDriver(account orgdriver.AccountContext) (orgdriver.OrgSyncDriver, error) {
	if s == nil || s.driverRegistry == nil {
		return nil, fmt.Errorf("driver registry not configured")
	}
	return s.driverRegistry.Resolve(account.ChannelCode, account.AppType)
}

func (s *SyncService) loadChannelAccount(ctx context.Context, tenantUUID, accountUUID string) (*socialModel.ChannelAccount, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	var channelAccount socialModel.ChannelAccount
	if err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).
		First(&channelAccount).Error; err != nil {
		return nil, err
	}
	return &channelAccount, nil
}

func credentialsToMap(input map[string]interface{}) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		if value == nil {
			continue
		}
		out[key] = fmt.Sprintf("%v", value)
	}
	return out
}

func upsertSourceUnits(ctx context.Context, tx *gorm.DB, tenantUUID, sourceAccountUUID, channelAccountUUID string, units []orgdriver.SourceUnitDTO) error {
	if len(units) == 0 {
		return nil
	}
	rows := make([]*model.SourceUnit, 0, len(units))
	now := time.Now().UTC()
	for _, unit := range units {
		if strings.TrimSpace(unit.ExternalUnitID) == "" {
			continue
		}
		name := strings.TrimSpace(unit.Name)
		if name == "" {
			name = unit.ExternalUnitID
		}
		status := strings.TrimSpace(unit.Status)
		if status == "" {
			status = "active"
		}
		order := unit.Order
		rows = append(rows, &model.SourceUnit{
			TenantUUID:           tenantUUID,
			SourceAccountUUID:    sourceAccountUUID,
			ChannelAccountUUID:   channelAccountUUID,
			ExternalUnitID:       strings.TrimSpace(unit.ExternalUnitID),
			ParentExternalUnitID: unit.ParentExternalUnitID,
			Name:                 name,
			Order:                order,
			Status:               status,
			UpdatedAt:            now,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "source_account_uuid"}, {Name: "external_unit_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"channel_account_uuid", "parent_external_unit_id", "name", "order", "status", "updated_at"}),
		}).
		Create(&rows).Error
}

func upsertSourceMembers(ctx context.Context, tx *gorm.DB, tenantUUID, sourceAccountUUID, channelAccountUUID string, members []orgdriver.SourceMemberDTO) error {
	if len(members) == 0 {
		return nil
	}
	rows := make([]*model.SourceMember, 0, len(members))
	profileCandidates := make(map[string]*model.SourceMemberProfile)
	now := time.Now().UTC()
	for _, member := range members {
		if strings.TrimSpace(member.ExternalMemberID) == "" {
			continue
		}
		name := strings.TrimSpace(member.Name)
		if name == "" {
			name = member.ExternalMemberID
		}
		profileStatus := strings.TrimSpace(member.ProfileStatus)
		if profileStatus == "" {
			profileStatus = model.ProfileStatusFull
		}
		status := strings.TrimSpace(member.Status)
		if status == "" {
			status = "active"
		}
		if profileStatus == model.ProfileStatusFull {
			profileCandidates[strings.TrimSpace(member.ExternalMemberID)] = &model.SourceMemberProfile{
				TenantUUID:         tenantUUID,
				ChannelAccountUUID: channelAccountUUID,
				ExternalMemberID:   strings.TrimSpace(member.ExternalMemberID),
				Name:               name,
				Phone:              strings.TrimSpace(member.Phone),
				Email:              strings.TrimSpace(member.Email),
				BizMail:            strings.TrimSpace(member.BizMail),
				Position:           strings.TrimSpace(member.Position),
				Address:            strings.TrimSpace(member.Address),
				MainDepartmentID:   strings.TrimSpace(member.MainDepartmentID),
				AvatarURL:          strings.TrimSpace(member.AvatarURL),
				UpdatedAt:          now,
			}
		}
		rows = append(rows, &model.SourceMember{
			TenantUUID:         tenantUUID,
			SourceAccountUUID:  sourceAccountUUID,
			ChannelAccountUUID: channelAccountUUID,
			ExternalMemberID:   strings.TrimSpace(member.ExternalMemberID),
			Name:               name,
			Phone:              strings.TrimSpace(member.Phone),
			Email:              strings.TrimSpace(member.Email),
			ProfileStatus:      profileStatus,
			Status:             status,
			UpdatedAt:          now,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "source_account_uuid"}, {Name: "external_member_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"channel_account_uuid", "name", "phone", "email", "profile_status", "status", "updated_at"}),
		}).
		Create(&rows).Error; err != nil {
		return err
	}
	if len(profileCandidates) == 0 {
		return nil
	}
	externalIDs := make([]string, 0, len(profileCandidates))
	for externalID := range profileCandidates {
		if strings.TrimSpace(externalID) == "" {
			continue
		}
		externalIDs = append(externalIDs, externalID)
	}
	if len(externalIDs) == 0 {
		return nil
	}
	var mapping []struct {
		SourceMemberUUID string
		ExternalMemberID string
	}
	if err := tx.WithContext(ctx).
		Model(&model.SourceMember{}).
		Select("source_member_uuid, external_member_id").
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_member_id IN ?", tenantUUID, sourceAccountUUID, externalIDs).
		Find(&mapping).Error; err != nil {
		return err
	}
	profiles := make([]*model.SourceMemberProfile, 0, len(mapping))
	for _, row := range mapping {
		profile := profileCandidates[row.ExternalMemberID]
		if profile == nil || strings.TrimSpace(row.SourceMemberUUID) == "" {
			continue
		}
		profile.SourceMemberUUID = row.SourceMemberUUID
		profiles = append(profiles, profile)
	}
	if len(profiles) == 0 {
		return nil
	}
	return tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "source_member_uuid"}},
			DoUpdates: clause.AssignmentColumns([]string{"channel_account_uuid", "external_member_id", "name", "phone", "email", "biz_mail", "position", "address", "main_department_id", "avatar_url", "updated_at"}),
		}).
		Create(&profiles).Error
}

func upsertSourceMemberUnits(ctx context.Context, tx *gorm.DB, tenantUUID, sourceAccountUUID, channelAccountUUID string, members []orgdriver.SourceMemberDTO) error {
	if len(members) == 0 {
		return nil
	}
	extMemberIDs := make([]string, 0, len(members))
	extUnitSet := map[string]struct{}{}
	memberDeptMap := map[string][]string{}
	memberDeptOrderMap := map[string]map[string]int{}
	for _, member := range members {
		memberID := strings.TrimSpace(member.ExternalMemberID)
		if memberID == "" {
			continue
		}
		if len(member.DepartmentIDs) == 0 {
			continue
		}
		deptIDs := make([]string, 0, len(member.DepartmentIDs))
		seen := map[string]struct{}{}
		for _, deptID := range member.DepartmentIDs {
			clean := strings.TrimSpace(deptID)
			if clean == "" {
				continue
			}
			if _, ok := seen[clean]; ok {
				continue
			}
			seen[clean] = struct{}{}
			deptIDs = append(deptIDs, clean)
			extUnitSet[clean] = struct{}{}
		}
		if len(deptIDs) == 0 {
			continue
		}
		memberDeptMap[memberID] = deptIDs
		if member.DepartmentOrders != nil {
			orderMap := map[string]int{}
			for deptID, order := range member.DepartmentOrders {
				clean := strings.TrimSpace(deptID)
				if clean == "" {
					continue
				}
				orderMap[clean] = order
			}
			if len(orderMap) > 0 {
				memberDeptOrderMap[memberID] = orderMap
			}
		}
		extMemberIDs = append(extMemberIDs, memberID)
	}
	if len(extMemberIDs) == 0 || len(extUnitSet) == 0 {
		return nil
	}
	extUnitIDs := make([]string, 0, len(extUnitSet))
	for id := range extUnitSet {
		extUnitIDs = append(extUnitIDs, id)
	}
	var unitRows []struct {
		SourceUnitUUID string
		ExternalUnitID string
	}
	if err := tx.WithContext(ctx).
		Model(&model.SourceUnit{}).
		Select("source_unit_uuid, external_unit_id").
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_unit_id IN ?", tenantUUID, sourceAccountUUID, extUnitIDs).
		Find(&unitRows).Error; err != nil {
		return err
	}
	unitMap := map[string]string{}
	for _, row := range unitRows {
		if strings.TrimSpace(row.SourceUnitUUID) == "" || strings.TrimSpace(row.ExternalUnitID) == "" {
			continue
		}
		unitMap[row.ExternalUnitID] = row.SourceUnitUUID
	}
	if len(unitMap) == 0 {
		return nil
	}
	var memberRows []struct {
		SourceMemberUUID string
		ExternalMemberID string
	}
	if err := tx.WithContext(ctx).
		Model(&model.SourceMember{}).
		Select("source_member_uuid, external_member_id").
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_member_id IN ?", tenantUUID, sourceAccountUUID, extMemberIDs).
		Find(&memberRows).Error; err != nil {
		return err
	}
	memberMap := map[string]string{}
	for _, row := range memberRows {
		if strings.TrimSpace(row.SourceMemberUUID) == "" || strings.TrimSpace(row.ExternalMemberID) == "" {
			continue
		}
		memberMap[row.ExternalMemberID] = row.SourceMemberUUID
	}
	if len(memberMap) == 0 {
		return nil
	}
	rows := make([]*model.SourceMemberUnit, 0, len(memberRows)*2)
	now := time.Now().UTC()
	for externalMemberID, deptIDs := range memberDeptMap {
		memberUUID := memberMap[externalMemberID]
		if memberUUID == "" {
			continue
		}
		deptOrderMap := memberDeptOrderMap[externalMemberID]
		for _, deptID := range deptIDs {
			unitUUID := unitMap[deptID]
			if unitUUID == "" {
				continue
			}
			order := 0
			if deptOrderMap != nil {
				if value, ok := deptOrderMap[deptID]; ok {
					order = value
				}
			}
			rows = append(rows, &model.SourceMemberUnit{
				TenantUUID:         tenantUUID,
				SourceAccountUUID:  sourceAccountUUID,
				ChannelAccountUUID: channelAccountUUID,
				SourceMemberUUID:   memberUUID,
				SourceUnitUUID:     unitUUID,
				ExternalMemberID:   externalMemberID,
				ExternalUnitID:     deptID,
				Order:              order,
				UpdatedAt:          now,
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID).
		Delete(&model.SourceMemberUnit{}).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Create(&rows).Error
}

func (s *SyncService) classifyUnitChanges(ctx context.Context, tenantUUID, sourceAccountUUID string, units []orgdriver.SourceUnitDTO) (int64, int64) {
	if s == nil || s.repo == nil || s.repo.DB == nil || len(units) == 0 {
		return 0, 0
	}
	ids := make([]string, 0, len(units))
	for _, unit := range units {
		if strings.TrimSpace(unit.ExternalUnitID) != "" {
			ids = append(ids, strings.TrimSpace(unit.ExternalUnitID))
		}
	}
	if len(ids) == 0 {
		return 0, 0
	}
	var existing int64
	if err := s.repo.DB.WithContext(ctx).
		Model(&model.SourceUnit{}).
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_unit_id IN ?", tenantUUID, sourceAccountUUID, ids).
		Count(&existing).Error; err != nil {
		return 0, int64(len(ids))
	}
	newCount := int64(len(ids)) - existing
	if newCount < 0 {
		newCount = 0
	}
	return newCount, existing
}

func (s *SyncService) classifyMemberChanges(ctx context.Context, tenantUUID, sourceAccountUUID string, members []orgdriver.SourceMemberDTO) (int64, int64) {
	if s == nil || s.repo == nil || s.repo.DB == nil || len(members) == 0 {
		return 0, 0
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		if strings.TrimSpace(member.ExternalMemberID) != "" {
			ids = append(ids, strings.TrimSpace(member.ExternalMemberID))
		}
	}
	if len(ids) == 0 {
		return 0, 0
	}
	var existing int64
	if err := s.repo.DB.WithContext(ctx).
		Model(&model.SourceMember{}).
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_member_id IN ?", tenantUUID, sourceAccountUUID, ids).
		Count(&existing).Error; err != nil {
		return 0, int64(len(ids))
	}
	newCount := int64(len(ids)) - existing
	if newCount < 0 {
		newCount = 0
	}
	return newCount, existing
}

func (s *SyncService) ensureSourceAccount(ctx context.Context, tenantUUID, channelAccountUUID string) (*model.SourceAccount, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, errors.New("source account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	existing, err := s.repo.FindByChannelAccount(ctx, tenantUUID, channelAccountUUID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, orgrepo.ErrSourceAccountNotFound) {
		return nil, err
	}
	var channelAccount socialModel.ChannelAccount
	if err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, channelAccountUUID).
		First(&channelAccount).Error; err != nil {
		return nil, err
	}
	record := &model.SourceAccount{
		SourceAccountUUID:  channelAccount.AccountUUID,
		TenantUUID:         tenantUUID,
		Provider:           channelAccount.ChannelCode,
		AppType:            channelAccount.AppType,
		ChannelAccountUUID: &channelAccount.AccountUUID,
		DisplayName:        channelAccount.DisplayName,
		Status:             model.SourceAccountStatusActive,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		return tx.Create(record).Error
	}); err != nil {
		return nil, err
	}
	return record, nil
}
