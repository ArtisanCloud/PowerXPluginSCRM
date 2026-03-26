package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

var (
	ErrStaffWelcomeServiceNotReady = errors.New("staff welcome service not ready")
	ErrInvalidStaffWelcomePayload  = errors.New("invalid staff welcome payload")
	ErrStaffWelcomeCodeNotFound    = errors.New("staff live code not found")
	ErrStaffWelcomeConfigNotFound  = errors.New("staff welcome config not found")
)

type StaffWelcomeSaveRequest struct {
	TenantUUID    string
	StaffCodeUUID string
	WelcomeMode   string
	ContentBlocks datatypes.JSON
	ActorUserUUID string
}

type StaffWelcomeSyncResult struct {
	StaffCodeUUID string `json:"staff_code_uuid"`
	SyncStatus    string `json:"sync_status"`
	AttemptNo     int    `json:"attempt_no"`
	Message       string `json:"message"`
}

type StaffWelcomeSyncStatus struct {
	StaffCodeUUID   string     `json:"staff_code_uuid"`
	SyncStatus      string     `json:"sync_status"`
	LastSyncError   string     `json:"last_sync_error,omitempty"`
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty"`
	LatestAttemptNo int        `json:"latest_attempt_no"`
}

type StaffWelcomeService struct {
	codeRepo    acqrepo.StaffLiveCodeRepository
	configRepo  acqrepo.StaffWelcomeConfigRepository
	attemptRepo acqrepo.StaffWelcomeSyncAttemptRepository
	maxRetries  int
}

func NewStaffWelcomeService(
	codeRepo acqrepo.StaffLiveCodeRepository,
	configRepo acqrepo.StaffWelcomeConfigRepository,
	attemptRepo acqrepo.StaffWelcomeSyncAttemptRepository,
) *StaffWelcomeService {
	return &StaffWelcomeService{
		codeRepo:    codeRepo,
		configRepo:  configRepo,
		attemptRepo: attemptRepo,
		maxRetries:  3,
	}
}

func (s *StaffWelcomeService) Save(ctx context.Context, req StaffWelcomeSaveRequest) (*acqmodel.StaffWelcomeConfig, error) {
	if s == nil || s.codeRepo == nil || s.configRepo == nil {
		return nil, ErrStaffWelcomeServiceNotReady
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	req.StaffCodeUUID = strings.ToLower(strings.TrimSpace(req.StaffCodeUUID))
	req.WelcomeMode = strings.ToLower(strings.TrimSpace(req.WelcomeMode))
	req.ActorUserUUID = strings.TrimSpace(req.ActorUserUUID)
	if req.TenantUUID == "" || req.StaffCodeUUID == "" || len(req.ContentBlocks) == 0 {
		return nil, ErrInvalidStaffWelcomePayload
	}
	if req.WelcomeMode != acqmodel.WelcomeModeSend && req.WelcomeMode != acqmodel.WelcomeModeSilent {
		return nil, ErrInvalidStaffWelcomePayload
	}
	if req.ActorUserUUID == "" {
		req.ActorUserUUID = "system"
	}

	if _, err := s.codeRepo.GetByUUID(ctx, req.TenantUUID, req.StaffCodeUUID); err != nil {
		if errors.Is(err, acqrepo.ErrRecordNotFound) {
			return nil, ErrStaffWelcomeCodeNotFound
		}
		return nil, err
	}

	preview, err := buildPayloadPreview(req.WelcomeMode, req.ContentBlocks)
	if err != nil {
		return nil, ErrInvalidStaffWelcomePayload
	}

	prev, err := s.configRepo.GetByStaffCodeUUID(ctx, req.TenantUUID, req.StaffCodeUUID)
	if err != nil && !errors.Is(err, acqrepo.ErrRecordNotFound) {
		return nil, err
	}
	version := 1
	if prev != nil {
		version = prev.Version + 1
	}

	cfg := &acqmodel.StaffWelcomeConfig{
		ConfigUUID:     uuid.NewString(),
		TenantUUID:     req.TenantUUID,
		StaffCodeUUID:  req.StaffCodeUUID,
		WelcomeMode:    req.WelcomeMode,
		ContentBlocks:  req.ContentBlocks,
		PayloadPreview: preview,
		SyncStatus:     acqmodel.WelcomeSyncStatusPending,
		Version:        version,
		CreatedBy:      req.ActorUserUUID,
		UpdatedBy:      req.ActorUserUUID,
	}
	if err := s.configRepo.Save(ctx, cfg); err != nil {
		return nil, err
	}
	return s.configRepo.GetByStaffCodeUUID(ctx, req.TenantUUID, req.StaffCodeUUID)
}

func (s *StaffWelcomeService) TriggerSync(ctx context.Context, tenantUUID, staffCodeUUID, actorUserUUID string) (*StaffWelcomeSyncResult, error) {
	if s == nil || s.configRepo == nil || s.attemptRepo == nil {
		return nil, ErrStaffWelcomeServiceNotReady
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	staffCodeUUID = strings.ToLower(strings.TrimSpace(staffCodeUUID))
	actorUserUUID = strings.TrimSpace(actorUserUUID)
	if actorUserUUID == "" {
		actorUserUUID = "system"
	}
	cfg, err := s.configRepo.GetByStaffCodeUUID(ctx, tenantUUID, staffCodeUUID)
	if err != nil {
		if errors.Is(err, acqrepo.ErrRecordNotFound) {
			return nil, ErrStaffWelcomeConfigNotFound
		}
		return nil, err
	}

	latestAttemptNo := 0
	latest, _ := s.attemptRepo.ListByStaffCodeUUID(ctx, tenantUUID, staffCodeUUID, 1)
	if len(latest) > 0 && latest[0] != nil {
		latestAttemptNo = latest[0].AttemptNo
	}
	attemptNo := latestAttemptNo
	retries := s.maxRetries
	if retries <= 0 {
		retries = 3
	}
	for i := 0; i < retries; i++ {
		attemptNo++
		now := time.Now().UTC()
		if err := s.attemptRepo.Create(ctx, &acqmodel.StaffWelcomeSyncAttempt{
			AttemptUUID:   uuid.NewString(),
			TenantUUID:    tenantUUID,
			StaffCodeUUID: staffCodeUUID,
			ConfigVersion: cfg.Version,
			AttemptNo:     attemptNo,
			Result:        "failed",
			ErrorCode:     acqmodel.WelcomeSyncStatusNotImplemented,
			ErrorMessage:  "channel sync adapter is not implemented in phase 7",
			StartedAt:     &now,
			FinishedAt:    &now,
		}); err != nil {
			return nil, err
		}
	}

	cfg.SyncStatus = acqmodel.WelcomeSyncStatusManualRequired
	cfg.LastSyncError = "channel sync adapter is not implemented in phase 7"
	cfg.UpdatedBy = actorUserUUID
	cfg.LastSyncedAt = nil
	if err := s.configRepo.Save(ctx, cfg); err != nil {
		return nil, err
	}

	return &StaffWelcomeSyncResult{
		StaffCodeUUID: staffCodeUUID,
		SyncStatus:    cfg.SyncStatus,
		AttemptNo:     attemptNo,
		Message:       "sync failed, manual retry required",
	}, nil
}

func (s *StaffWelcomeService) GetSyncStatus(ctx context.Context, tenantUUID, staffCodeUUID string) (*StaffWelcomeSyncStatus, error) {
	if s == nil || s.configRepo == nil || s.attemptRepo == nil {
		return nil, ErrStaffWelcomeServiceNotReady
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	staffCodeUUID = strings.ToLower(strings.TrimSpace(staffCodeUUID))
	cfg, err := s.configRepo.GetByStaffCodeUUID(ctx, tenantUUID, staffCodeUUID)
	if err != nil {
		if errors.Is(err, acqrepo.ErrRecordNotFound) {
			return nil, ErrStaffWelcomeConfigNotFound
		}
		return nil, err
	}
	latestAttemptNo := 0
	latest, _ := s.attemptRepo.ListByStaffCodeUUID(ctx, tenantUUID, staffCodeUUID, 1)
	if len(latest) > 0 && latest[0] != nil {
		latestAttemptNo = latest[0].AttemptNo
	}
	return &StaffWelcomeSyncStatus{
		StaffCodeUUID:   staffCodeUUID,
		SyncStatus:      cfg.SyncStatus,
		LastSyncError:   cfg.LastSyncError,
		LastSyncedAt:    cfg.LastSyncedAt,
		LatestAttemptNo: latestAttemptNo,
	}, nil
}

func buildPayloadPreview(welcomeMode string, contentBlocks datatypes.JSON) (datatypes.JSON, error) {
	var blocks []map[string]any
	if err := json.Unmarshal(contentBlocks, &blocks); err != nil {
		return nil, err
	}
	preview := map[string]any{
		"welcome_mode":   welcomeMode,
		"content_blocks": blocks,
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
	}
	buf, err := json.Marshal(preview)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(buf), nil
}
