package marketplace

import (
	"context"
	"errors"
	"strings"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChecklistRepository handles persistence for checklist runs and items.
type ChecklistRepository struct {
	*repository.BaseRepository[dbm.ChecklistRun]
}

func NewChecklistRepository(db *gorm.DB) *ChecklistRepository {
	return &ChecklistRepository{
		BaseRepository: repository.NewBaseRepository[dbm.ChecklistRun](db),
	}
}

// CreateRun persists a checklist run with its items in a transaction.
func (r *ChecklistRepository) CreateRun(ctx context.Context, run *dbm.ChecklistRun, items []dbm.ChecklistItem) error {
	if run == nil {
		return errors.New("checklist run is required")
	}
	tenantID := strings.TrimSpace(run.TenantUuid)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	run.TenantUuid = tenantID
	if strings.TrimSpace(run.ID) == "" {
		run.ID = uuid.NewString()
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		if err := tx.Omit("Items").Create(run).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for i := range items {
			items[i].ChecklistRunID = run.ID
			items[i].TenantUuid = run.TenantUuid
			if strings.TrimSpace(items[i].ID) == "" {
				items[i].ID = uuid.NewString()
			}
		}
		for i := range items {
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// LatestRun fetches the latest checklist run for a listing.
func (r *ChecklistRepository) LatestRun(ctx context.Context, tenantID, listingID string) (*dbm.ChecklistRun, error) {
	var run dbm.ChecklistRun
	err := r.DB.WithContext(ctx).
		Preload("Items").
		Where("tenant_uuid = ? AND listing_id = ?", tenantID, listingID).
		Order("run_number DESC, created_at DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// UpdateRunResult updates the completion status of a run.
func (r *ChecklistRepository) UpdateRunResult(ctx context.Context, runID, tenantID, status, summary string, completedAt *time.Time) error {
	updates := map[string]any{
		"status": status,
	}
	if summary != "" {
		updates["summary"] = summary
	}
	if completedAt != nil {
		updates["completed_at"] = *completedAt
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return errors.New("run_id is required")
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		res := tx.Model(&dbm.ChecklistRun{}).
			Where("id = ? AND tenant_uuid = ?", runID, tenantID).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// ListRuns returns recent checklist runs for auditing.
func (r *ChecklistRepository) ListRuns(ctx context.Context, tenantID, listingID string, limit int) ([]*dbm.ChecklistRun, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var runs []*dbm.ChecklistRun
	err := r.DB.WithContext(ctx).
		Preload("Items").
		Where("tenant_uuid = ? AND listing_id = ?", tenantID, listingID).
		Order("run_number DESC").
		Limit(limit).
		Find(&runs).Error
	return runs, err
}
