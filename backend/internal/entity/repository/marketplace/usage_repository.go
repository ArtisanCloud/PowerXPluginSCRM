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
	"gorm.io/gorm/clause"
)

// UsageRepository manages persistence for marketplace usage envelopes and aggregates.
type UsageRepository struct {
	*repository.BaseRepository[dbm.UsageEnvelope]
	db *gorm.DB
}

// NewUsageRepository constructs a new usage repository instance.
func NewUsageRepository(db *gorm.DB) *UsageRepository {
	return &UsageRepository{
		BaseRepository: repository.NewBaseRepository[dbm.UsageEnvelope](db),
		db:             db,
	}
}

// InsertEnvelopes stores usage envelopes with idempotent checksum deduplication.
func (r *UsageRepository) InsertEnvelopes(ctx context.Context, tenantID string, envelopes []*dbm.UsageEnvelope) (int, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return 0, errors.New("tenant_uuid is required")
	}
	if len(envelopes) == 0 {
		return 0, nil
	}
	inserted := 0
	err := r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		for _, env := range envelopes {
			if env == nil {
				continue
			}
			env.TenantUuid = tenantID
			if strings.TrimSpace(env.ID) == "" {
				env.ID = uuid.NewString()
			}
			if env.IngestStatus == "" {
				env.IngestStatus = dbm.UsageIngestStatusProcessed
			}
			if env.IngestedAt.IsZero() {
				env.IngestedAt = time.Now().UTC()
			}
		}
		res := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "checksum"},
			},
			DoNothing: true,
		}).Create(&envelopes)
		if res.Error != nil {
			return res.Error
		}
		inserted = int(res.RowsAffected)
		return nil
	})
	return inserted, err
}

// UpsertAggregate upserts aggregates totals for dashboards.
func (r *UsageRepository) UpsertAggregate(ctx context.Context, aggregate *dbm.UsageAggregate) error {
	if aggregate == nil {
		return errors.New("aggregate is required")
	}
	tenantID := strings.TrimSpace(aggregate.TenantUuid)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	if strings.TrimSpace(aggregate.ID) == "" {
		aggregate.ID = uuid.NewString()
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "tenant_uuid"},
				{Name: "license_id"},
				{Name: "metric"},
				{Name: "window"},
				{Name: "time_bucket"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"total":      aggregate.Total,
				"delta":      aggregate.Delta,
				"currency":   aggregate.Currency,
				"revenue":    aggregate.Revenue,
				"updated_at": time.Now().UTC(),
			}),
		}).Create(aggregate).Error
	})
}

// ListAggregates returns aggregates for a license and window ordered by time.
func (r *UsageRepository) ListAggregates(ctx context.Context, tenantID, licenseID string, window dbm.AggregationWindow) ([]*dbm.UsageAggregate, error) {
	tenantID = strings.TrimSpace(tenantID)
	licenseID = strings.TrimSpace(licenseID)
	if tenantID == "" || licenseID == "" {
		return nil, errors.New("tenant_uuid and license_id are required")
	}
	query := r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND license_id = ? AND window = ?", tenantID, licenseID, string(window)).
		Order("time_bucket ASC")
	var aggregates []*dbm.UsageAggregate
	if err := query.Find(&aggregates).Error; err != nil {
		return nil, err
	}
	return aggregates, nil
}

// LatestAggregate returns the most recent aggregate for a metric/window.
func (r *UsageRepository) LatestAggregate(ctx context.Context, tenantID, licenseID, metric string, window dbm.AggregationWindow) (*dbm.UsageAggregate, error) {
	tenantID = strings.TrimSpace(tenantID)
	licenseID = strings.TrimSpace(licenseID)
	metric = strings.TrimSpace(metric)
	if tenantID == "" || licenseID == "" || metric == "" {
		return nil, errors.New("tenant_uuid, license_id and metric are required")
	}
	var aggregate dbm.UsageAggregate
	err := r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND license_id = ? AND metric = ? AND window = ?", tenantID, licenseID, metric, string(window)).
		Order("time_bucket DESC").
		First(&aggregate).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &aggregate, nil
}

// GetAggregate returns aggregate for a specific bucket if present.
func (r *UsageRepository) GetAggregate(ctx context.Context, tenantID, licenseID, metric string, window dbm.AggregationWindow, bucket time.Time) (*dbm.UsageAggregate, error) {
	tenantID = strings.TrimSpace(tenantID)
	licenseID = strings.TrimSpace(licenseID)
	metric = strings.TrimSpace(metric)
	if tenantID == "" || licenseID == "" || metric == "" {
		return nil, errors.New("tenant_uuid, license_id and metric are required")
	}
	var aggregate dbm.UsageAggregate
	err := r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND license_id = ? AND metric = ? AND window = ? AND time_bucket = ?", tenantID, licenseID, metric, string(window), bucket.UTC()).
		First(&aggregate).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &aggregate, nil
}

// DeleteEnvelopesBefore removes envelopes for a license before the cutoff timestamp.
func (r *UsageRepository) DeleteEnvelopesBefore(ctx context.Context, tenantID, licenseID string, before time.Time) (int, error) {
	tenantID = strings.TrimSpace(tenantID)
	licenseID = strings.TrimSpace(licenseID)
	if tenantID == "" || licenseID == "" {
		return 0, errors.New("tenant_uuid and license_id are required")
	}
	before = before.UTC()
	var affected int64
	err := r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		res := tx.Where("tenant_uuid = ? AND license_id = ? AND timestamp_end <= ?", tenantID, licenseID, before).
			Delete(&dbm.UsageEnvelope{})
		if res.Error != nil {
			return res.Error
		}
		affected = res.RowsAffected
		return nil
	})
	return int(affected), err
}

// DeleteAggregatesBefore removes aggregated rows for a license up to cutoff.
func (r *UsageRepository) DeleteAggregatesBefore(ctx context.Context, tenantID, licenseID string, before time.Time) (int, error) {
	tenantID = strings.TrimSpace(tenantID)
	licenseID = strings.TrimSpace(licenseID)
	if tenantID == "" || licenseID == "" {
		return 0, errors.New("tenant_uuid and license_id are required")
	}
	before = before.UTC()
	var affected int64
	err := r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		res := tx.Where("tenant_uuid = ? AND license_id = ? AND time_bucket <= ?", tenantID, licenseID, before).
			Delete(&dbm.UsageAggregate{})
		if res.Error != nil {
			return res.Error
		}
		affected = res.RowsAffected
		return nil
	})
	return int(affected), err
}
