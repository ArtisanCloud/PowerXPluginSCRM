package social_channel_governance

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SyncFoundationRepository struct {
	*repository.BaseRepository[model.SyncJob]
}

func NewSyncFoundationRepository(db *gorm.DB) *SyncFoundationRepository {
	return &SyncFoundationRepository{BaseRepository: repository.NewBaseRepository[model.SyncJob](db)}
}

func (r *SyncFoundationRepository) CreateJob(ctx context.Context, job *model.SyncJob) (*model.SyncJob, bool, error) {
	if r == nil || r.DB == nil {
		return nil, false, errors.New("repository database is not initialized")
	}
	if job == nil {
		return nil, false, errors.New("job is required")
	}
	job.TenantUUID = strings.TrimSpace(strings.ToLower(job.TenantUUID))
	job.Domain = strings.TrimSpace(job.Domain)
	job.Direction = strings.TrimSpace(job.Direction)
	job.Mode = strings.TrimSpace(job.Mode)
	job.IdempotencyKey = strings.TrimSpace(job.IdempotencyKey)
	if job.TenantUUID == "" {
		return nil, false, repository.ErrTenantUuidRequired
	}
	if job.Status == "" {
		job.Status = "pending"
	}
	if job.Payload == nil {
		job.Payload = datatypes.JSONMap{}
	}

	created := true
	err := r.WithTenantTx(ctx, job.TenantUUID, func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(job)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			return nil
		}
		created = false
		return tx.Where("tenant_uuid = ? AND idempotency_key = ?", job.TenantUUID, job.IdempotencyKey).First(job).Error
	})
	if err != nil {
		return nil, false, err
	}
	return job, created, nil
}

func (r *SyncFoundationRepository) ListJobs(ctx context.Context, tenantUUID, domain, status string, limit int) ([]model.SyncJob, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.SyncJob, 0)
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		q := tx.Model(&model.SyncJob{}).Where("tenant_uuid = ?", tenantUUID)
		if strings.TrimSpace(domain) != "" {
			q = q.Where("domain = ?", strings.TrimSpace(domain))
		}
		if strings.TrimSpace(status) != "" {
			q = q.Where("status = ?", strings.TrimSpace(status))
		}
		return q.Order("created_at DESC").Limit(limit).Find(&items).Error
	})
	return items, err
}

func (r *SyncFoundationRepository) UpdateJobStatus(ctx context.Context, tenantUUID, jobUUID, status, errorCode, errorMessage string, attemptNo int) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	if tenantUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"status":        strings.TrimSpace(status),
		"attempt_no":    attemptNo,
		"error_code":    strings.TrimSpace(errorCode),
		"error_message": strings.TrimSpace(errorMessage),
		"updated_at":    now,
	}
	if status == "running" {
		updates["started_at"] = now
	}
	if status == "success" || status == "failed" || status == "dead_letter" {
		updates["finished_at"] = now
	}
	return r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.SyncJob{}).
			Where("tenant_uuid = ? AND job_uuid = ?", tenantUUID, strings.TrimSpace(jobUUID)).
			Updates(updates)
		return res.Error
	})
}

func (r *SyncFoundationRepository) SaveConflict(ctx context.Context, conflict *model.SyncConflict) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	if conflict == nil {
		return errors.New("conflict is required")
	}
	conflict.TenantUUID = strings.TrimSpace(strings.ToLower(conflict.TenantUUID))
	if conflict.TenantUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	if conflict.LocalValue == nil {
		conflict.LocalValue = datatypes.JSONMap{}
	}
	if conflict.RemoteValue == nil {
		conflict.RemoteValue = datatypes.JSONMap{}
	}
	return r.WithTenantTx(ctx, conflict.TenantUUID, func(tx *gorm.DB) error {
		return tx.Create(conflict).Error
	})
}

func (r *SyncFoundationRepository) AddDeadLetter(ctx context.Context, item *model.SyncDeadLetterItem) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	if item == nil {
		return errors.New("dead letter item is required")
	}
	item.TenantUUID = strings.TrimSpace(strings.ToLower(item.TenantUUID))
	if item.TenantUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	if item.Payload == nil {
		item.Payload = datatypes.JSONMap{}
	}
	return r.WithTenantTx(ctx, item.TenantUUID, func(tx *gorm.DB) error {
		return tx.Create(item).Error
	})
}
