package lead_capture

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	"github.com/google/uuid"
)

var (
	ErrWelcomeSyncServiceNotReady   = errors.New("welcome sync service not ready")
	ErrWelcomeSyncConfigNotFound    = errors.New("welcome sync config not found")
	ErrWelcomeSyncChannelCodeAbsent = errors.New("welcome sync channel code not found")
)

type WelcomeSyncTriggerRequest struct {
	TenantUUID    string
	CodeUUID      string
	ActorUserUUID string
}

type WelcomeSyncTriggerResult struct {
	CodeUUID   string `json:"code_uuid"`
	SyncStatus string `json:"sync_status"`
	AttemptNo  int    `json:"attempt_no"`
	Message    string `json:"message"`
	ErrorCode  string `json:"error_code,omitempty"`
}

type WelcomeSyncStatusResult struct {
	CodeUUID        string     `json:"code_uuid"`
	SyncStatus      string     `json:"sync_status"`
	LastSyncError   string     `json:"last_sync_error,omitempty"`
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty"`
	LatestAttemptNo int        `json:"latest_attempt_no"`
}

type WelcomeSyncService struct {
	channelCodeRepo leadrepo.ChannelCodeRepository
	configRepo      leadrepo.CodeWelcomeConfigRepository
	attemptRepo     leadrepo.CodeWelcomeSyncAttemptRepository
	adapter         WelcomeSyncAdapter
	metrics         *leadobs.Metrics
	retryBackoffs   []time.Duration
	maxRetries      int
}

func NewWelcomeSyncService(
	channelCodeRepo leadrepo.ChannelCodeRepository,
	configRepo leadrepo.CodeWelcomeConfigRepository,
	attemptRepo leadrepo.CodeWelcomeSyncAttemptRepository,
	adapter WelcomeSyncAdapter,
	metrics *leadobs.Metrics,
) *WelcomeSyncService {
	return (&WelcomeSyncService{
		channelCodeRepo: channelCodeRepo,
		configRepo:      configRepo,
		attemptRepo:     attemptRepo,
		adapter:         adapter,
		metrics:         metrics,
	}).WithRetryPolicy([]time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second}, 3)
}

func (s *WelcomeSyncService) WithRetryPolicy(backoffs []time.Duration, maxRetries int) *WelcomeSyncService {
	if s == nil {
		return s
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if len(backoffs) == 0 {
		backoffs = []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second}
	}
	s.maxRetries = maxRetries
	s.retryBackoffs = backoffs
	return s
}

func (s *WelcomeSyncService) TriggerSync(ctx context.Context, req WelcomeSyncTriggerRequest) (*WelcomeSyncTriggerResult, error) {
	if s == nil || s.channelCodeRepo == nil || s.configRepo == nil || s.attemptRepo == nil || s.adapter == nil {
		return nil, ErrWelcomeSyncServiceNotReady
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	req.CodeUUID = strings.ToLower(strings.TrimSpace(req.CodeUUID))
	req.ActorUserUUID = strings.TrimSpace(req.ActorUserUUID)
	if req.TenantUUID == "" || req.CodeUUID == "" {
		return nil, ErrWelcomeSyncConfigNotFound
	}
	if req.ActorUserUUID == "" {
		req.ActorUserUUID = leadobs.ResolveActorUserUUID(ctx, "system")
	}

	code, err := s.channelCodeRepo.GetByCodeUUID(ctx, req.TenantUUID, req.CodeUUID)
	if err != nil {
		if errors.Is(err, leadrepo.ErrRecordNotFound) {
			return nil, ErrWelcomeSyncChannelCodeAbsent
		}
		return nil, err
	}
	cfg, err := s.configRepo.GetByCodeUUID(ctx, req.TenantUUID, req.CodeUUID)
	if err != nil {
		if errors.Is(err, leadrepo.ErrRecordNotFound) {
			return nil, ErrWelcomeSyncConfigNotFound
		}
		return nil, err
	}

	cfg.SyncStatus = leadmodel.WelcomeSyncStatusSyncing
	cfg.LastSyncError = ""
	cfg.UpdatedBy = req.ActorUserUUID
	if err := s.configRepo.Save(ctx, cfg); err != nil {
		return nil, err
	}

	latestAttemptNo := 0
	latestAttempts, err := s.attemptRepo.ListByCodeUUID(ctx, req.TenantUUID, req.CodeUUID, 1)
	if err == nil && len(latestAttempts) > 0 && latestAttempts[0] != nil {
		latestAttemptNo = latestAttempts[0].AttemptNo
	}

	attemptNo := latestAttemptNo
	for i := 0; i < s.maxRetries; i++ {
		attemptNo++
		startedAt := time.Now().UTC()
		publishErr := s.adapter.PublishWelcome(ctx, WelcomeSyncPublishInput{
			TenantUUID:         req.TenantUUID,
			CodeUUID:           req.CodeUUID,
			Channel:            code.Channel,
			AppType:            code.AppType,
			ChannelAccountUUID: code.ChannelAccountUUID,
			MessageContent:     cfg.MessageContent,
		})

		errorCode := ""
		errorMsg := ""
		result := leadmodel.WelcomeSyncResultSuccess
		if publishErr != nil {
			result = leadmodel.WelcomeSyncResultFailed
			errorCode, errorMsg = resolveWelcomeSyncError(publishErr)
		}
		now := time.Now().UTC()
		if err := s.attemptRepo.Create(ctx, &leadmodel.CodeWelcomeSyncAttempt{
			AttemptUUID:   uuid.NewString(),
			TenantUUID:    req.TenantUUID,
			CodeUUID:      req.CodeUUID,
			ConfigVersion: cfg.Version,
			TriggerSource: leadmodel.WelcomeSyncTriggerManual,
			AttemptNo:     attemptNo,
			Result:        result,
			ErrorCode:     errorCode,
			ErrorMessage:  errorMsg,
			StartedAt:     &startedAt,
			FinishedAt:    &now,
		}); err != nil {
			return nil, err
		}
		s.recordAttemptMetrics(ctx, code, req, attemptNo, result, errorCode)

		if publishErr == nil {
			cfg.SyncStatus = leadmodel.WelcomeSyncStatusSuccess
			cfg.LastSyncError = ""
			cfg.LastSyncedAt = &now
			cfg.UpdatedBy = req.ActorUserUUID
			if err := s.configRepo.Save(ctx, cfg); err != nil {
				return nil, err
			}
			return &WelcomeSyncTriggerResult{
				CodeUUID:   req.CodeUUID,
				SyncStatus: cfg.SyncStatus,
				AttemptNo:  attemptNo,
				Message:    "sync succeeded",
			}, nil
		}

		if i < s.maxRetries-1 {
			if backoff := s.backoffAt(i); backoff > 0 {
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
		}
	}

	lastAttempt, _ := s.attemptRepo.ListByCodeUUID(ctx, req.TenantUUID, req.CodeUUID, 1)
	lastErrCode := ""
	lastErrMsg := ""
	if len(lastAttempt) > 0 && lastAttempt[0] != nil {
		lastErrCode = strings.TrimSpace(lastAttempt[0].ErrorCode)
		lastErrMsg = strings.TrimSpace(lastAttempt[0].ErrorMessage)
	}
	cfg.SyncStatus = leadmodel.WelcomeSyncStatusManualRequired
	cfg.LastSyncError = formatSyncError(lastErrCode, lastErrMsg)
	cfg.UpdatedBy = req.ActorUserUUID
	if err := s.configRepo.Save(ctx, cfg); err != nil {
		return nil, err
	}

	return &WelcomeSyncTriggerResult{
		CodeUUID:   req.CodeUUID,
		SyncStatus: cfg.SyncStatus,
		AttemptNo:  attemptNo,
		Message:    "sync failed, manual retry required",
		ErrorCode:  lastErrCode,
	}, nil
}

func (s *WelcomeSyncService) GetStatus(ctx context.Context, tenantUUID, codeUUID string) (*WelcomeSyncStatusResult, error) {
	if s == nil || s.configRepo == nil || s.attemptRepo == nil {
		return nil, ErrWelcomeSyncServiceNotReady
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	codeUUID = strings.ToLower(strings.TrimSpace(codeUUID))
	if tenantUUID == "" || codeUUID == "" {
		return nil, ErrWelcomeSyncConfigNotFound
	}
	cfg, err := s.configRepo.GetByCodeUUID(ctx, tenantUUID, codeUUID)
	if err != nil {
		if errors.Is(err, leadrepo.ErrRecordNotFound) {
			return nil, ErrWelcomeSyncConfigNotFound
		}
		return nil, err
	}
	latestAttemptNo := 0
	attempts, err := s.attemptRepo.ListByCodeUUID(ctx, tenantUUID, codeUUID, 1)
	if err == nil && len(attempts) > 0 && attempts[0] != nil {
		latestAttemptNo = attempts[0].AttemptNo
	}
	return &WelcomeSyncStatusResult{
		CodeUUID:        codeUUID,
		SyncStatus:      cfg.SyncStatus,
		LastSyncError:   cfg.LastSyncError,
		LastSyncedAt:    cfg.LastSyncedAt,
		LatestAttemptNo: latestAttemptNo,
	}, nil
}

func (s *WelcomeSyncService) backoffAt(i int) time.Duration {
	if s == nil || len(s.retryBackoffs) == 0 {
		return 0
	}
	if i < 0 {
		i = 0
	}
	if i >= len(s.retryBackoffs) {
		return s.retryBackoffs[len(s.retryBackoffs)-1]
	}
	return s.retryBackoffs[i]
}

func (s *WelcomeSyncService) recordAttemptMetrics(
	ctx context.Context,
	code *leadmodel.ChannelCode,
	req WelcomeSyncTriggerRequest,
	attemptNo int,
	result string,
	errorCode string,
) {
	if code == nil {
		return
	}
	if s.metrics != nil {
		s.metrics.RecordWelcomeSyncAttempt(code.Channel, code.AppType, leadmodel.WelcomeSyncTriggerManual, result, errorCode)
	}
	leadobs.EmitWelcomeSyncPublished(ctx, req.TenantUUID, req.CodeUUID, req.ActorUserUUID, map[string]any{
		"trigger_source": leadmodel.WelcomeSyncTriggerManual,
		"attempt_no":     attemptNo,
		"result":         result,
		"error_code":     errorCode,
	})
}

func resolveWelcomeSyncError(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	if typed := (*WelcomeSyncAdapterError)(nil); errors.As(err, &typed) && typed != nil {
		code := normalizeWelcomeSyncErrorCode(typed.Code)
		if code == "" {
			code = WelcomeSyncErrorChannelUnavailable
		}
		return code, strings.TrimSpace(typed.Message)
	}
	return WelcomeSyncErrorChannelUnavailable, strings.TrimSpace(err.Error())
}

func formatSyncError(errorCode, errorMsg string) string {
	errorCode = strings.TrimSpace(errorCode)
	errorMsg = strings.TrimSpace(errorMsg)
	if errorCode == "" {
		return errorMsg
	}
	if errorMsg == "" {
		return errorCode
	}
	return fmt.Sprintf("%s: %s", errorCode, errorMsg)
}
