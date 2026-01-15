package tool_grant

import (
	"context"
	"errors"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/tool_grant"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository exposes persistence helpers for tool grant lifecycle data.
type Repository struct {
	db          *gorm.DB
	revocations *repository.BaseRepository[model.Revocation]
	events      *repository.BaseRepository[model.UsageEvent]
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db:          db,
		revocations: repository.NewBaseRepository[model.Revocation](db),
		events:      repository.NewBaseRepository[model.UsageEvent](db),
	}
}

func (r *Repository) WithTx(tx *gorm.DB) *Repository {
	if tx == nil {
		return r
	}
	return NewRepository(tx)
}

func (r *Repository) RecordRevocation(ctx context.Context, rec *model.Revocation) (*model.Revocation, error) {
	if rec == nil {
		return nil, gorm.ErrInvalidData
	}
	if rec.TenantUuid == "" || rec.ToolGrantID == "" {
		return nil, errors.New("tenant_uuid and toolgrant_id required")
	}
	if rec.RevokedAt.IsZero() {
		rec.RevokedAt = time.Now().UTC()
	}
	if rec.TtlExpiry.IsZero() {
		return nil, errors.New("ttl_expiry required")
	}
	return r.revocations.Upsert(ctx, rec, []clause.Column{
		{Name: "tenant_uuid"},
		{Name: "toolgrant_id"},
	})
}

func (r *Repository) ListRevocations(ctx context.Context, tenantID string, limit int) ([]*model.Revocation, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Revocation{}).
		Where("tenant_uuid = ?", tenantID).
		Order("revoked_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var revs []*model.Revocation
	if err := query.Find(&revs).Error; err != nil {
		return nil, err
	}
	return revs, nil
}

func (r *Repository) RecordUsageEvent(ctx context.Context, evt *model.UsageEvent, metadata datatypes.JSONMap) (*model.UsageEvent, error) {
	if evt == nil {
		return nil, gorm.ErrInvalidData
	}
	if evt.TenantUuid == "" || evt.ToolGrantID == "" {
		return nil, errors.New("tenant_uuid and toolgrant_id required")
	}
	if evt.EventType == "" {
		return nil, errors.New("event_type required")
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now().UTC()
	}
	if metadata != nil {
		evt.Metadata = metadata
	}
	return r.events.Create(ctx, evt)
}

func (r *Repository) ListUsageEvents(ctx context.Context, tenantID, toolGrantID string, limit int) ([]*model.UsageEvent, error) {
	query := r.db.WithContext(ctx).
		Model(&model.UsageEvent{}).
		Where("tenant_uuid = ?", tenantID)
	if toolGrantID != "" {
		query = query.Where("toolgrant_id = ?", toolGrantID)
	}
	query = query.Order("occurred_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var events []*model.UsageEvent
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
