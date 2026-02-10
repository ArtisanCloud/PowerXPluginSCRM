package org_sync

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
)

// SyncLogService provides read access to sync logs.
type SyncLogService struct {
	repo       *orgrepo.SyncLogRepository
	sourceRepo *orgrepo.SourceAccountRepository
}

func NewSyncLogService(repo *orgrepo.SyncLogRepository, sourceRepo *orgrepo.SourceAccountRepository) *SyncLogService {
	return &SyncLogService{repo: repo, sourceRepo: sourceRepo}
}

func (s *SyncLogService) ListByAccount(ctx context.Context, tenantUUID, sourceAccountUUID, channelAccountUUID string, limit int) ([]*model.SyncLog, string, string, error) {
	if s == nil || s.repo == nil {
		return nil, "", "", errors.New("sync log repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || (sourceAccountUUID == "" && channelAccountUUID == "") {
		return nil, "", "", repository.ErrTenantUuidRequired
	}
	if channelAccountUUID != "" {
		logs, err := s.repo.ListByChannelAccount(ctx, tenantUUID, channelAccountUUID, limit)
		mode, hint := s.resolveSyncMode(ctx, tenantUUID, "", channelAccountUUID)
		return logs, mode, hint, err
	}
	logs, err := s.repo.ListByAccount(ctx, tenantUUID, sourceAccountUUID, limit)
	mode, hint := s.resolveSyncMode(ctx, tenantUUID, sourceAccountUUID, "")
	return logs, mode, hint, err
}

func (s *SyncLogService) resolveSyncMode(ctx context.Context, tenantUUID, sourceAccountUUID, channelAccountUUID string) (string, string) {
	if s == nil || s.sourceRepo == nil {
		return "", ""
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" {
		return "", ""
	}
	var account *model.SourceAccount
	var err error
	switch {
	case sourceAccountUUID != "":
		account, err = s.sourceRepo.GetByUUID(ctx, tenantUUID, sourceAccountUUID)
	case channelAccountUUID != "":
		account, err = s.sourceRepo.FindByChannelAccount(ctx, tenantUUID, channelAccountUUID)
	default:
		return "", ""
	}
	if err != nil || account == nil {
		return "", ""
	}
	if strings.EqualFold(strings.TrimSpace(account.Provider), "wechat") && strings.EqualFold(strings.TrimSpace(account.AppType), "wecom") {
		return "app_detail", "应用 Secret 模式：通过 user/list + department/list 同步部门与成员详情。"
	}
	return "", ""
}
