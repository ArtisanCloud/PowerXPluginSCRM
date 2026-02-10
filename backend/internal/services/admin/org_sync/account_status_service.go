package org_sync

import (
	"context"
	"errors"
	"strings"
	"time"

	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	"gorm.io/gorm"
)

// AccountStatusService updates mappings when a source account is disabled.
type AccountStatusService struct {
	memberMappingRepo *orgrepo.MemberMappingRepository
	unitMappingRepo   *orgrepo.UnitMappingRepository
}

func NewAccountStatusService(
	memberMappingRepo *orgrepo.MemberMappingRepository,
	unitMappingRepo *orgrepo.UnitMappingRepository,
) *AccountStatusService {
	return &AccountStatusService{memberMappingRepo: memberMappingRepo, unitMappingRepo: unitMappingRepo}
}

func (s *AccountStatusService) MarkMappingsDisabled(ctx context.Context, tenantUUID, channelAccountUUID string) (int64, int64, error) {
	if s == nil || s.memberMappingRepo == nil || s.unitMappingRepo == nil {
		return 0, 0, errors.New("account status service dependencies not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return 0, 0, repository.ErrTenantUuidRequired
	}
	sourceAccounts, err := s.lookupSourceAccounts(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return 0, 0, err
	}
	if len(sourceAccounts) == 0 {
		return 0, 0, nil
	}
	now := time.Now().UTC()
	var unitAffected int64
	var memberAffected int64
	if err := s.unitMappingRepo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&orgmodel.UnitMapping{}).
			Where("tenant_uuid = ? AND source_unit_uuid IN (SELECT source_unit_uuid FROM "+orgmodel.SourceUnit{}.TableName()+" WHERE tenant_uuid = ? AND source_account_uuid IN ?)", tenantUUID, tenantUUID, sourceAccounts).
			Updates(map[string]any{
				"mapping_status": orgmodel.MappingStatusDisabled,
				"updated_at":     now,
			})
		if res.Error != nil {
			return res.Error
		}
		unitAffected = res.RowsAffected
		res = tx.Model(&orgmodel.MemberMapping{}).
			Where("tenant_uuid = ? AND source_member_uuid IN (SELECT source_member_uuid FROM "+orgmodel.SourceMember{}.TableName()+" WHERE tenant_uuid = ? AND source_account_uuid IN ?)", tenantUUID, tenantUUID, sourceAccounts).
			Updates(map[string]any{
				"mapping_status": orgmodel.MappingStatusDisabled,
				"updated_at":     now,
			})
		if res.Error != nil {
			return res.Error
		}
		memberAffected = res.RowsAffected
		return nil
	}); err != nil {
		return 0, 0, err
	}
	return unitAffected, memberAffected, nil
}

func (s *AccountStatusService) lookupSourceAccounts(ctx context.Context, tenantUUID, channelAccountUUID string) ([]string, error) {
	if s == nil || s.memberMappingRepo == nil || s.memberMappingRepo.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	var rows []struct {
		SourceAccountUUID string
	}
	err := s.memberMappingRepo.DB.WithContext(ctx).
		Model(&orgmodel.SourceAccount{}).
		Select("source_account_uuid").
		Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.SourceAccountUUID) != "" {
			ids = append(ids, strings.TrimSpace(row.SourceAccountUUID))
		}
	}
	return ids, nil
}
