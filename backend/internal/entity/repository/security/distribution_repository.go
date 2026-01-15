package security

import (
	"context"
	"errors"
	"time"

	secmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/security"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DistributionRepository manages advisory distribution persistence helpers.
type DistributionRepository struct {
	db   *gorm.DB
	base *repository.BaseRepository[secmodel.AdvisoryDistribution]
}

// NewDistributionRepository constructs the repository.
func NewDistributionRepository(db *gorm.DB) *DistributionRepository {
	return &DistributionRepository{
		db:   db,
		base: repository.NewBaseRepository[secmodel.AdvisoryDistribution](db),
	}
}

// WithTx clones the repository using the supplied transactional handle.
func (r *DistributionRepository) WithTx(tx *gorm.DB) *DistributionRepository {
	if tx == nil {
		return r
	}
	return NewDistributionRepository(tx)
}

// Upsert creates or updates a distribution row keyed by advisory, tenant, and channel.
func (r *DistributionRepository) Upsert(ctx context.Context, record *secmodel.AdvisoryDistribution) (*secmodel.AdvisoryDistribution, error) {
	if record == nil {
		return nil, gorm.ErrInvalidData
	}
	if record.AdvisoryID == "" || record.TenantUuid == "" || record.Channel == "" {
		return nil, errors.New("advisory_id, tenant_uuid, and channel are required")
	}
	if record.Status == "" {
		record.Status = secmodel.DistributionStatusPending
	}
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now

	result, err := r.base.Upsert(ctx, record, []clause.Column{
		{Name: "advisory_id"},
		{Name: "tenant_uuid"},
		{Name: "channel"},
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateStatus updates the status (and optionally delivery metadata) for a distribution row.
func (r *DistributionRepository) UpdateStatus(ctx context.Context, id string, status string, deliveredAt *time.Time, metadata datatypes.JSONMap) error {
	if id == "" {
		return errors.New("id is required")
	}
	if status == "" {
		return errors.New("status is required")
	}
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}
	if deliveredAt != nil {
		updates["delivered_at"] = *deliveredAt
	}
	if metadata != nil {
		updates["metadata"] = metadata
	}
	return r.db.WithContext(ctx).
		Model(&secmodel.AdvisoryDistribution{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// ListByAdvisory returns distributions associated with a given advisory.
func (r *DistributionRepository) ListByAdvisory(ctx context.Context, advisoryID string) ([]*secmodel.AdvisoryDistribution, error) {
	if advisoryID == "" {
		return nil, errors.New("advisory_id is required")
	}
	var records []*secmodel.AdvisoryDistribution
	if err := r.db.WithContext(ctx).
		Model(&secmodel.AdvisoryDistribution{}).
		Where("advisory_id = ?", advisoryID).
		Order("created_at ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// FindForTenant returns the distribution row for a specific tenant/channel pair.
func (r *DistributionRepository) FindForTenant(ctx context.Context, advisoryID, tenantID, channel string) (*secmodel.AdvisoryDistribution, error) {
	if advisoryID == "" || tenantID == "" || channel == "" {
		return nil, errors.New("advisory_id, tenant_uuid, and channel are required")
	}
	var record secmodel.AdvisoryDistribution
	err := r.db.WithContext(ctx).
		Where("advisory_id = ? AND tenant_uuid = ? AND channel = ?", advisoryID, tenantID, channel).
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}
