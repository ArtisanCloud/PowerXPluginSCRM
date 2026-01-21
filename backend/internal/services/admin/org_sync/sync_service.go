package org_sync

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	"gorm.io/gorm"
)

// SyncService handles source org sync triggers.
type SyncService struct {
	repo *orgrepo.SourceAccountRepository
}

func NewSyncService(repo *orgrepo.SourceAccountRepository) *SyncService {
	return &SyncService{repo: repo}
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
		return nil, err
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"last_sync_at":      now,
		"last_sync_status":  model.SyncStatusQueued,
		"last_sync_message": "",
		"updated_at":        now,
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
		return nil
	}); err != nil {
		return nil, err
	}
	account.LastSyncAt = &now
	account.LastSyncStatus = model.SyncStatusQueued
	account.LastSyncMessage = ""
	account.UpdatedAt = now
	return account, nil
}
