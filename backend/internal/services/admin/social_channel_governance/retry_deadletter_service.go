package social_channel_governance

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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

func (s *RetryDeadletterService) List(ctx context.Context, tenantUUID, domain, replayStatus string, limit int) ([]model.SyncDeadLetterItem, error) {
	if s == nil || s.repo == nil {
		return []model.SyncDeadLetterItem{}, nil
	}
	return s.repo.ListDeadLetters(ctx, tenantUUID, domain, replayStatus, limit)
}

func (s *RetryDeadletterService) Replay(ctx context.Context, tenantUUID, deadLetterUUID, replayedBy string) (*model.SyncDeadLetterItem, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("dead letter service unavailable")
	}
	out, err := s.repo.ReplayDeadLetter(ctx, tenantUUID, deadLetterUUID, replayedBy)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, socialrepo.ErrSyncJobNotFound
		}
		return nil, err
	}
	return out, nil
}
