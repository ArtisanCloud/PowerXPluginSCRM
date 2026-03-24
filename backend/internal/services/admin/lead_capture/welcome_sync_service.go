package lead_capture

import (
	"context"
	"errors"
	"strings"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
)

var (
	ErrWelcomeSyncServiceNotReady = errors.New("welcome sync service not ready")
)

type WelcomeSyncService struct {
	configRepo  leadrepo.CodeWelcomeConfigRepository
	attemptRepo leadrepo.CodeWelcomeSyncAttemptRepository
	metrics     *leadobs.Metrics
}

func NewWelcomeSyncService(
	configRepo leadrepo.CodeWelcomeConfigRepository,
	attemptRepo leadrepo.CodeWelcomeSyncAttemptRepository,
	metrics *leadobs.Metrics,
) *WelcomeSyncService {
	return &WelcomeSyncService{
		configRepo:  configRepo,
		attemptRepo: attemptRepo,
		metrics:     metrics,
	}
}

func (s *WelcomeSyncService) SaveConfig(ctx context.Context, input *leadmodel.CodeWelcomeConfig) error {
	if s == nil || s.configRepo == nil {
		return ErrWelcomeSyncServiceNotReady
	}
	if input == nil {
		return errors.New("welcome config is required")
	}
	if strings.TrimSpace(input.SyncStatus) == "" {
		input.SyncStatus = leadmodel.WelcomeSyncStatusPending
	}
	if err := s.configRepo.Save(ctx, input); err != nil {
		return err
	}
	if s.metrics != nil {
		s.metrics.RecordChannelCodeConfigChange("", "")
	}
	leadobs.EmitWelcomeConfigSaved(ctx, input.TenantUUID, input.CodeUUID, input.UpdatedBy, map[string]any{
		"sync_status": input.SyncStatus,
		"version":     input.Version,
	})
	return nil
}

func (s *WelcomeSyncService) RecordAttempt(ctx context.Context, input *leadmodel.CodeWelcomeSyncAttempt) error {
	if s == nil || s.attemptRepo == nil {
		return ErrWelcomeSyncServiceNotReady
	}
	if input == nil {
		return errors.New("welcome sync attempt is required")
	}
	if strings.TrimSpace(input.TriggerSource) == "" {
		input.TriggerSource = leadmodel.WelcomeSyncTriggerManual
	}
	if strings.TrimSpace(input.Result) == "" {
		input.Result = leadmodel.WelcomeSyncResultFailed
	}
	if err := s.attemptRepo.Create(ctx, input); err != nil {
		return err
	}
	if s.metrics != nil {
		s.metrics.RecordWelcomeSyncAttempt("", "", input.TriggerSource, input.Result, input.ErrorCode)
	}
	leadobs.EmitWelcomeSyncPublished(ctx, input.TenantUUID, input.CodeUUID, "", map[string]any{
		"trigger_source": input.TriggerSource,
		"attempt_no":     input.AttemptNo,
		"result":         input.Result,
		"error_code":     input.ErrorCode,
	})
	return nil
}
