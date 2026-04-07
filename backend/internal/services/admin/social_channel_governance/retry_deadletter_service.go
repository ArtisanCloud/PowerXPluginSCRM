package social_channel_governance

import (
	"context"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"gorm.io/datatypes"
)

type RetryDeadletterService struct {
	repo *socialrepo.SyncFoundationRepository
}

func NewRetryDeadletterService(repo *socialrepo.SyncFoundationRepository) *RetryDeadletterService {
	return &RetryDeadletterService{repo: repo}
}

func (s *RetryDeadletterService) HandleFailure(ctx context.Context, job *model.SyncJob, errCode, errMessage string) error {
	if s == nil || s.repo == nil || job == nil {
		return nil
	}
	job.AttemptNo++
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = 3
	}
	if job.AttemptNo < job.MaxAttempts {
		return s.repo.UpdateJobStatus(ctx, job.TenantUUID, job.JobUUID, "failed", errCode, errMessage, job.AttemptNo)
	}
	if err := s.repo.UpdateJobStatus(ctx, job.TenantUUID, job.JobUUID, "dead_letter", errCode, errMessage, job.AttemptNo); err != nil {
		return err
	}
	return s.repo.AddDeadLetter(ctx, &model.SyncDeadLetterItem{
		TenantUUID:       strings.TrimSpace(strings.ToLower(job.TenantUUID)),
		JobUUID:          job.JobUUID,
		Domain:           job.Domain,
		Direction:        job.Direction,
		LastErrorCode:    strings.TrimSpace(errCode),
		LastErrorMessage: strings.TrimSpace(errMessage),
		RetryExhaustedAt: time.Now().UTC(),
		ReplayStatus:     "pending",
		Payload:          datatypes.JSONMap(job.Payload),
	})
}
