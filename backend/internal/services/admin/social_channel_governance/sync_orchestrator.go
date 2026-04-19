package social_channel_governance

import (
	"context"
	"errors"
	"strings"
)

type SyncOrchestrator struct {
	factory    *ChannelFactory
	scheduler  *SyncScheduler
	jobService *SyncJobService
}

func NewSyncOrchestrator(factory *ChannelFactory, scheduler *SyncScheduler, jobService *SyncJobService) *SyncOrchestrator {
	return &SyncOrchestrator{
		factory:    factory,
		scheduler:  scheduler,
		jobService: jobService,
	}
}

type OrchestrateInput struct {
	TenantUUID string
	Channel    string
	AppType    string
	Domain     string
	Direction  string
	Mode       string
	Payload    map[string]any
}

func (s *SyncOrchestrator) Submit(ctx context.Context, in OrchestrateInput) (string, error) {
	if s == nil || s.jobService == nil {
		return "", errors.New("sync orchestrator unavailable")
	}
	if strings.TrimSpace(in.TenantUUID) == "" {
		return "", errors.New("tenant_uuid is required")
	}
	if strings.TrimSpace(in.Domain) == "" {
		return "", errors.New("domain is required")
	}

	// FR-018: same tenant + domain must be serialized.
	unlock := func() {}
	if s.scheduler != nil {
		unlock = s.scheduler.Lock(in.TenantUUID, in.Domain)
	}
	defer unlock()

	// Adapter resolve is explicit; orchestration remains channel-agnostic.
	if s.factory != nil {
		if _, ok := s.factory.Resolve(in.Channel, in.AppType); !ok {
			// No adapter is still accepted; job will be created with not_supported capability status.
		}
	}
	job, _, err := s.jobService.Create(ctx, CreateSyncJobInput{
		TenantUUID: in.TenantUUID,
		Channel:    in.Channel,
		AppType:    in.AppType,
		Domain:     in.Domain,
		Direction:  in.Direction,
		Mode:       in.Mode,
		Payload:    in.Payload,
	})
	if err != nil {
		return "", err
	}
	if job == nil {
		return "", errors.New("failed to create sync job")
	}
	if strings.TrimSpace(strings.ToLower(job.Status)) == "pending" {
		_ = s.jobService.Execute(ctx, job)
	}
	return job.JobUUID, nil
}
