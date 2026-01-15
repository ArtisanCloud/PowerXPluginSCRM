package security

import (
	"context"
	"errors"
	"strings"
	"time"

	secmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/security"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AdvisoryListFilter captures query parameters for listing advisories.
type AdvisoryListFilter struct {
	Severities []string
	Statuses   []string
	Limit      int
}

// AdvisoryRepository provides persistence helpers for vulnerability advisories.
type AdvisoryRepository struct {
	db         *gorm.DB
	advisories *repository.BaseRepository[secmodel.Advisory]
}

// NewAdvisoryRepository constructs the repository with the provided database handle.
func NewAdvisoryRepository(db *gorm.DB) *AdvisoryRepository {
	return &AdvisoryRepository{
		db:         db,
		advisories: repository.NewBaseRepository[secmodel.Advisory](db),
	}
}

// WithTx clones the repository with the supplied transactional DB handle.
func (r *AdvisoryRepository) WithTx(tx *gorm.DB) *AdvisoryRepository {
	if tx == nil {
		return r
	}
	return NewAdvisoryRepository(tx)
}

// Create inserts a new advisory row.
func (r *AdvisoryRepository) Create(ctx context.Context, advisory *secmodel.Advisory) (*secmodel.Advisory, error) {
	if advisory == nil {
		return nil, gorm.ErrInvalidData
	}
	if advisory.Reference == "" {
		return nil, errors.New("reference is required")
	}
	if advisory.Severity == "" {
		return nil, errors.New("severity is required")
	}
	if advisory.Status == "" {
		advisory.Status = secmodel.AdvisoryStatusOpen
	}
	if len(advisory.AffectedVersions) == 0 {
		advisory.SetAffectedVersions([]string{})
	}
	if advisory.ID == "" {
		advisory.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if advisory.CreatedAt.IsZero() {
		advisory.CreatedAt = now
	}
	if advisory.UpdatedAt.IsZero() {
		advisory.UpdatedAt = now
	}
	return r.advisories.Create(ctx, advisory)
}

// GetByID fetches an advisory by primary key.
func (r *AdvisoryRepository) GetByID(ctx context.Context, id string) (*secmodel.Advisory, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}
	var record secmodel.Advisory
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Take(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// FindByReference returns an advisory by human readable reference.
func (r *AdvisoryRepository) FindByReference(ctx context.Context, reference string) (*secmodel.Advisory, error) {
	if reference == "" {
		return nil, errors.New("reference is required")
	}
	var record secmodel.Advisory
	err := r.db.WithContext(ctx).
		Where("reference = ?", reference).
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateFields performs a partial update on the advisory.
func (r *AdvisoryRepository) UpdateFields(ctx context.Context, id string, updates map[string]interface{}) error {
	if id == "" {
		return errors.New("id is required")
	}
	if len(updates) == 0 {
		return nil
	}
	if _, exists := updates["updated_at"]; !exists {
		updates["updated_at"] = time.Now().UTC()
	}
	return r.db.WithContext(ctx).
		Model(&secmodel.Advisory{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// ReplaceAffectedVersions rewrites the affected versions array for the advisory.
func (r *AdvisoryRepository) ReplaceAffectedVersions(ctx context.Context, id string, versions []string) error {
	if id == "" {
		return errors.New("id is required")
	}
	payload := datatypes.NewJSONSlice(versions)
	return r.UpdateFields(ctx, id, map[string]interface{}{
		"affected_versions": payload,
	})
}

// List returns advisories filtered by severity/status with optional limit.
func (r *AdvisoryRepository) List(ctx context.Context, filter AdvisoryListFilter) ([]*secmodel.Advisory, error) {
	query := r.db.WithContext(ctx).
		Model(&secmodel.Advisory{}).
		Order("created_at DESC")
	if len(filter.Severities) > 0 {
		query = query.Where("severity IN ?", normalizeStrings(filter.Severities))
	}
	if len(filter.Statuses) > 0 {
		query = query.Where("status IN ?", normalizeStrings(filter.Statuses))
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	var advisories []*secmodel.Advisory
	if err := query.Find(&advisories).Error; err != nil {
		return nil, err
	}
	return advisories, nil
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			out = append(out, strings.ToUpper(trimmed))
		}
	}
	return out
}
