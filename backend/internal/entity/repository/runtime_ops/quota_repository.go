package runtime_ops

import (
	"context"
	"errors"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/runtime_ops"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"gorm.io/gorm"
)

// QuotaRepository persists quota usage and marketplace overage summaries.
type QuotaRepository struct {
	ledger  *repository.BaseRepository[model.QuotaLedger]
	overage *repository.BaseRepository[model.MarketplaceOverage]
}

// NewQuotaRepository constructs a repository for quota ledgers and summaries.
func NewQuotaRepository(db *gorm.DB) *QuotaRepository {
	return &QuotaRepository{
		ledger:  repository.NewBaseRepository[model.QuotaLedger](db),
		overage: repository.NewBaseRepository[model.MarketplaceOverage](db),
	}
}

// RecordUsage inserts a new ledger entry for the given scope window.
func (r *QuotaRepository) RecordUsage(ctx context.Context, entry *model.QuotaLedger) (*model.QuotaLedger, error) {
	if entry == nil {
		return nil, gorm.ErrInvalidData
	}
	if entry.WindowEnd.Before(entry.WindowStart) {
		return nil, errors.New("window end must be after start")
	}
	if entry.ScopeType == "tenant" && entry.ScopeRef == "" {
		tid, err := authx.RequireTenantUUID(ctx)
		if err != nil {
			return nil, err
		}
		entry.ScopeRef = tid
	}
	return r.ledger.Create(ctx, entry)
}

// ListWindow returns ledger entries for the given scope within the window.
func (r *QuotaRepository) ListWindow(ctx context.Context, scopeType, scopeRef string, start, end time.Time) ([]*model.QuotaLedger, error) {
	var ledgers []*model.QuotaLedger
	query := r.ledger.DB.WithContext(ctx).
		Where("scope_type = ?", scopeType).
		Where("scope_ref = ?", scopeRef).
		Where("window_start >= ? AND window_end <= ?", start, end).
		Order("window_start")
	if err := query.Find(&ledgers).Error; err != nil {
		return nil, err
	}
	return ledgers, nil
}

// CreateOverage stores a marketplace overage summary.
func (r *QuotaRepository) CreateOverage(ctx context.Context, summary *model.MarketplaceOverage) (*model.MarketplaceOverage, error) {
	if summary == nil {
		return nil, gorm.ErrInvalidData
	}
	return r.overage.Create(ctx, summary)
}

// MarkOverageReported toggles the reported flag on an overage entry.
func (r *QuotaRepository) MarkOverageReported(ctx context.Context, id string) error {
	_, err := r.overage.Patch(ctx, map[string]interface{}{"id": id}, map[string]interface{}{
		"reported":   true,
		"updated_at": time.Now(),
	})
	return err
}
