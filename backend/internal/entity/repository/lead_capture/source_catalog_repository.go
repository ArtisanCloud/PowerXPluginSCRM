package lead_capture

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

var (
	ErrSourceCatalogNotFound = errors.New("lead source catalog not found")
)

type LeadSourceCatalogRepository struct {
	*repository.BaseRepository[model.LeadSourceCatalog]
}

func NewLeadSourceCatalogRepository(db *gorm.DB) *LeadSourceCatalogRepository {
	return &LeadSourceCatalogRepository{BaseRepository: repository.NewBaseRepository[model.LeadSourceCatalog](db)}
}

func (r *LeadSourceCatalogRepository) List(ctx context.Context, category string, enabledOnly bool) ([]*model.LeadSourceCatalog, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	category = strings.ToLower(strings.TrimSpace(category))

	query := r.DB.WithContext(ctx)
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}

	var out []*model.LeadSourceCatalog
	if err := query.Order("sort ASC, created_at ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LeadSourceCatalogRepository) Create(ctx context.Context, item *model.LeadSourceCatalog) (*model.LeadSourceCatalog, error) {
	if item == nil {
		return nil, errors.New("catalog item is required")
	}
	item.Category = strings.ToLower(strings.TrimSpace(item.Category))
	item.Code = strings.ToLower(strings.TrimSpace(item.Code))
	item.Label = strings.TrimSpace(item.Label)
	if item.Category == "" || item.Code == "" || item.Label == "" {
		return nil, errors.New("category, code, label are required")
	}
	if item.Sort == 0 {
		item.Sort = 100
	}
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	if err := r.DB.WithContext(ctx).Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (r *LeadSourceCatalogRepository) GetByUUID(ctx context.Context, catalogUUID string) (*model.LeadSourceCatalog, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	catalogUUID = strings.ToLower(strings.TrimSpace(catalogUUID))
	if catalogUUID == "" {
		return nil, ErrSourceCatalogNotFound
	}
	var out model.LeadSourceCatalog
	err := r.DB.WithContext(ctx).
		Where("catalog_uuid = ?", catalogUUID).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSourceCatalogNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LeadSourceCatalogRepository) UpdateByUUID(ctx context.Context, catalogUUID string, updates map[string]any) (*model.LeadSourceCatalog, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	catalogUUID = strings.ToLower(strings.TrimSpace(catalogUUID))
	if catalogUUID == "" {
		return nil, ErrSourceCatalogNotFound
	}
	updates["updated_at"] = time.Now().UTC()

	var out model.LeadSourceCatalog
	res := r.DB.WithContext(ctx).Model(&model.LeadSourceCatalog{}).
		Where("catalog_uuid = ?", catalogUUID).
		Updates(updates)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrSourceCatalogNotFound
	}
	if err := r.DB.WithContext(ctx).Where("catalog_uuid = ?", catalogUUID).First(&out).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSourceCatalogNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LeadSourceCatalogRepository) DeleteByUUID(ctx context.Context, catalogUUID string) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	catalogUUID = strings.ToLower(strings.TrimSpace(catalogUUID))
	if catalogUUID == "" {
		return ErrSourceCatalogNotFound
	}

	res := r.DB.WithContext(ctx).Where("catalog_uuid = ?", catalogUUID).Delete(&model.LeadSourceCatalog{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSourceCatalogNotFound
	}
	return nil
}
