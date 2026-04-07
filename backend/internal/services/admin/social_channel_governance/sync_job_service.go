package social_channel_governance

import (
	"context"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"gorm.io/datatypes"
)

type SyncJobService struct {
	repo              *socialrepo.SyncFoundationRepository
	idempotency       *IdempotencyService
	scheduler         *SyncScheduler
	capabilityService *CapabilityService
}

type CreateSyncJobInput struct {
	TenantUUID string
	Channel    string
	AppType    string
	Domain     string
	Direction  string
	Mode       string
	Payload    map[string]any
}

func NewSyncJobService(
	repo *socialrepo.SyncFoundationRepository,
	idempotency *IdempotencyService,
	scheduler *SyncScheduler,
	capabilityService *CapabilityService,
) *SyncJobService {
	return &SyncJobService{
		repo:              repo,
		idempotency:       idempotency,
		scheduler:         scheduler,
		capabilityService: capabilityService,
	}
}

func (s *SyncJobService) Create(ctx context.Context, in CreateSyncJobInput) (*model.SyncJob, bool, error) {
	if s == nil || s.repo == nil {
		return nil, false, nil
	}
	in.TenantUUID = strings.TrimSpace(strings.ToLower(in.TenantUUID))
	in.Domain = strings.TrimSpace(in.Domain)
	in.Direction = strings.TrimSpace(in.Direction)
	in.Mode = strings.TrimSpace(in.Mode)
	if in.Direction == "" {
		in.Direction = "pull"
	}
	if in.Mode == "" {
		in.Mode = "incremental"
	}
	payload := datatypes.JSONMap(in.Payload)
	if payload == nil {
		payload = datatypes.JSONMap{}
	}
	matrix := map[string]string{}
	if s.capabilityService != nil {
		matrix = s.capabilityService.Matrix(ctx, in.TenantUUID, in.Channel, in.AppType)
	}
	capabilityStatus := "supported"
	if v := strings.TrimSpace(matrix[in.Domain]); v != "" {
		capabilityStatus = v
	}
	idempotencyKey := ""
	if s.idempotency != nil {
		idempotencyKey = s.idempotency.BuildKey(in.TenantUUID, in.Domain, in.Direction, in.Mode, in.Channel, in.AppType)
	}
	job := &model.SyncJob{
		TenantUUID:      in.TenantUUID,
		Domain:          in.Domain,
		Direction:       in.Direction,
		Mode:            in.Mode,
		Status:          "pending",
		IdempotencyKey:  idempotencyKey,
		MaxAttempts:     3,
		Payload:         payload,
		CapabilityState: capabilityStatus,
	}
	createdJob, created, err := s.repo.CreateJob(ctx, job)
	if err != nil {
		return nil, false, err
	}
	return createdJob, created, nil
}

func (s *SyncJobService) List(ctx context.Context, tenantUUID, domain, status string, limit int) ([]model.SyncJob, error) {
	if s == nil || s.repo == nil {
		return []model.SyncJob{}, nil
	}
	return s.repo.ListJobs(ctx, tenantUUID, domain, status, limit)
}
