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

func (r *SyncFoundationRepository) ListConflicts(ctx context.Context, tenantUUID, domain, status string, limit int) ([]model.SyncConflict, error) {
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
	items := make([]model.SyncConflict, 0)
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		q := tx.Model(&model.SyncConflict{}).Where("tenant_uuid = ?", tenantUUID)
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

func (r *SyncFoundationRepository) ReplayConflict(ctx context.Context, tenantUUID, conflictUUID, resolvedBy string) (*model.SyncConflict, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	conflictUUID = strings.TrimSpace(conflictUUID)
	if conflictUUID == "" {
		return nil, errors.New("conflict_uuid is required")
	}
	now := time.Now().UTC()
	resolvedBy = strings.TrimSpace(resolvedBy)
	if resolvedBy == "" {
		resolvedBy = "manual_replay"
	}
	var out model.SyncConflict
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.SyncConflict{}).
			Where("tenant_uuid = ? AND conflict_uuid = ?", tenantUUID, conflictUUID).
			Updates(map[string]any{
				"status":      "replayed",
				"resolved_by": resolvedBy,
				"resolved_at": now,
				"updated_at":  now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("tenant_uuid = ? AND conflict_uuid = ?", tenantUUID, conflictUUID).First(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *SyncFoundationRepository) ListDeadLetters(ctx context.Context, tenantUUID, domain, replayStatus string, limit int) ([]model.SyncDeadLetterItem, error) {
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
	items := make([]model.SyncDeadLetterItem, 0)
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		q := tx.Model(&model.SyncDeadLetterItem{}).Where("tenant_uuid = ?", tenantUUID)
		if strings.TrimSpace(domain) != "" {
			q = q.Where("domain = ?", strings.TrimSpace(domain))
		}
		if strings.TrimSpace(replayStatus) != "" {
			q = q.Where("replay_status = ?", strings.TrimSpace(replayStatus))
		}
		return q.Order("created_at DESC").Limit(limit).Find(&items).Error
	})
	return items, err
}

func (r *SyncFoundationRepository) ReplayDeadLetter(ctx context.Context, tenantUUID, deadLetterUUID, replayedBy string) (*model.SyncDeadLetterItem, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	deadLetterUUID = strings.TrimSpace(deadLetterUUID)
	if deadLetterUUID == "" {
		return nil, errors.New("dead_letter_uuid is required")
	}
	now := time.Now().UTC()
	replayedBy = strings.TrimSpace(replayedBy)
	if replayedBy == "" {
		replayedBy = "manual_replay"
	}
	var out model.SyncDeadLetterItem
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.SyncDeadLetterItem{}).
			Where("tenant_uuid = ? AND dead_letter_uuid = ?", tenantUUID, deadLetterUUID).
			Updates(map[string]any{
				"replay_status": "replayed",
				"replayed_by":   replayedBy,
				"replayed_at":   now,
				"updated_at":    now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("tenant_uuid = ? AND dead_letter_uuid = ?", tenantUUID, deadLetterUUID).First(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *SyncFoundationRepository) UpsertCheckpoint(ctx context.Context, checkpoint *model.SyncCheckpoint) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	if checkpoint == nil {
		return errors.New("checkpoint is required")
	}
	checkpoint.TenantUUID = strings.TrimSpace(strings.ToLower(checkpoint.TenantUUID))
	checkpoint.Domain = strings.TrimSpace(strings.ToLower(checkpoint.Domain))
	checkpoint.Direction = strings.TrimSpace(strings.ToLower(checkpoint.Direction))
	if checkpoint.TenantUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	if checkpoint.Domain == "" || checkpoint.Direction == "" {
		return errors.New("domain and direction are required")
	}
	if checkpoint.LastEventTime.IsZero() {
		checkpoint.LastEventTime = time.Now().UTC()
	}
	return r.WithTenantTx(ctx, checkpoint.TenantUUID, func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "tenant_uuid"},
				{Name: "domain"},
				{Name: "direction"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"cursor":           checkpoint.Cursor,
				"snapshot_version": checkpoint.SnapshotVersion,
				"last_event_time":  checkpoint.LastEventTime,
				"updated_at":       time.Now().UTC(),
			}),
		}).Create(checkpoint).Error
	})
}

func (r *SyncFoundationRepository) GetCheckpoint(ctx context.Context, tenantUUID, domain, direction string) (*model.SyncCheckpoint, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	domain = strings.TrimSpace(strings.ToLower(domain))
	direction = strings.TrimSpace(strings.ToLower(direction))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if domain == "" || direction == "" {
		return nil, errors.New("domain and direction are required")
	}
	out := &model.SyncCheckpoint{}
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		return tx.Where("tenant_uuid = ? AND domain = ? AND direction = ?", tenantUUID, domain, direction).First(out).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
