package org_sync

import (
	"context"
	"errors"
	"strings"
	"time"

	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

// AccountStatusService updates bindings when a source account is disabled.
type AccountStatusService struct {
	db *gorm.DB
}

func NewAccountStatusService(db *gorm.DB) *AccountStatusService {
	return &AccountStatusService{db: db}
}

func (s *AccountStatusService) MarkBindingsDisabled(ctx context.Context, tenantUUID, channelAccountUUID string) (int64, int64, error) {
	if s == nil || s.db == nil {
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
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&orgmodel.UnitBinding{}).
			Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
			Updates(map[string]any{
				"sync_status": "disabled",
				"updated_at":  now,
			})
		if res.Error != nil {
			return res.Error
		}
		unitAffected = res.RowsAffected
		res = tx.Model(&orgmodel.MemberBinding{}).
			Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
			Updates(map[string]any{
				"sync_status": "disabled",
				"updated_at":  now,
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
	if s == nil || s.db == nil {
		return nil, errors.New("repository database is not initialized")
	}
	var rows []struct {
		SourceAccountUUID string
	}
	err := s.db.WithContext(ctx).
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
