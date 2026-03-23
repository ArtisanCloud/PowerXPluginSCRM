package lead_capture

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/jackc/pgconn"
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

func (r *LeadSourceCatalogRepository) ListByTenant(ctx context.Context, tenantUUID, category string, enabledOnly bool) ([]*model.LeadSourceCatalog, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	category = strings.ToLower(strings.TrimSpace(category))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}

	query := r.DB.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID)
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}

	var out []*model.LeadSourceCatalog
	err := query.Order("sort ASC, created_at ASC").Find(&out).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return []*model.LeadSourceCatalog{}, nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return []*model.LeadSourceCatalog{}, nil
		}
		return nil, err
	}
	return out, nil
}

func (r *LeadSourceCatalogRepository) Create(ctx context.Context, item *model.LeadSourceCatalog) (*model.LeadSourceCatalog, error) {
	if item == nil {
		return nil, errors.New("catalog item is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	item.Category = strings.ToLower(strings.TrimSpace(item.Category))
	item.Code = strings.ToLower(strings.TrimSpace(item.Code))
	item.Label = strings.TrimSpace(item.Label)
	if item.TenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
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

	err := r.WithTenantTx(ctx, item.TenantUUID, func(tx *gorm.DB) error {
		return tx.Create(item).Error
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *LeadSourceCatalogRepository) GetByUUID(ctx context.Context, tenantUUID, catalogUUID string) (*model.LeadSourceCatalog, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	catalogUUID = strings.ToLower(strings.TrimSpace(catalogUUID))
	if tenantUUID == "" || catalogUUID == "" {
		return nil, ErrSourceCatalogNotFound
	}
	var out model.LeadSourceCatalog
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND catalog_uuid = ?", tenantUUID, catalogUUID).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSourceCatalogNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LeadSourceCatalogRepository) UpdateByUUID(ctx context.Context, tenantUUID, catalogUUID string, updates map[string]any) (*model.LeadSourceCatalog, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	catalogUUID = strings.ToLower(strings.TrimSpace(catalogUUID))
	if tenantUUID == "" || catalogUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	updates["updated_at"] = time.Now().UTC()

	var out model.LeadSourceCatalog
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.LeadSourceCatalog{}).
			Where("tenant_uuid = ? AND catalog_uuid = ?", tenantUUID, catalogUUID).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrSourceCatalogNotFound
		}
		return tx.Where("tenant_uuid = ? AND catalog_uuid = ?", tenantUUID, catalogUUID).First(&out).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSourceCatalogNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LeadSourceCatalogRepository) DeleteByUUID(ctx context.Context, tenantUUID, catalogUUID string) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	catalogUUID = strings.ToLower(strings.TrimSpace(catalogUUID))
	if tenantUUID == "" || catalogUUID == "" {
		return repository.ErrTenantUuidRequired
	}

	return r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Where("tenant_uuid = ? AND catalog_uuid = ?", tenantUUID, catalogUUID).Delete(&model.LeadSourceCatalog{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrSourceCatalogNotFound
		}
		return nil
	})
}
