package marketplace

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

// ListingQuery defines filters for listing lookups.
type ListingQuery struct {
	Status []string
	Locale string
	Search string
	Limit  int
	Offset int
}

// ListingRepository provides persistence operations for marketplace listings.
type ListingRepository struct {
	*repository.BaseRepository[dbm.Listing]
}

func NewListingRepository(db *gorm.DB) *ListingRepository {
	return &ListingRepository{
		BaseRepository: repository.NewBaseRepository[dbm.Listing](db),
	}
}

// Create persists a new listing draft.
func (r *ListingRepository) Create(ctx context.Context, listing *dbm.Listing) error {
	if listing == nil {
		return errors.New("listing is required")
	}
	tenantID := strings.TrimSpace(listing.TenantUuid)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	listing.TenantUuid = tenantID
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Create(listing).Error
	})
}

// Update saves listing field changes.
func (r *ListingRepository) Update(ctx context.Context, listing *dbm.Listing) error {
	if listing == nil {
		return errors.New("listing is required")
	}
	tenantID := strings.TrimSpace(listing.TenantUuid)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	listing.TenantUuid = tenantID
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Save(listing).Error
	})
}

// FindByID fetches a listing by identifier and tenant.
func (r *ListingRepository) FindByID(ctx context.Context, tenantID, listingID string) (*dbm.Listing, error) {
	var listing dbm.Listing
	err := r.baseQuery(ctx, tenantID).
		Preload("Assets").
		Preload("PricingPlans").
		Preload("PricingPlans.Tiers").
		First(&listing, "id = ?", listingID).Error
	if err != nil {
		return nil, err
	}
	return &listing, nil
}

// List returns paginated listings for a tenant with optional filters.
func (r *ListingRepository) List(ctx context.Context, tenantID string, query ListingQuery) ([]*dbm.Listing, int64, error) {
	db := r.baseQuery(ctx, tenantID)
	if len(query.Status) > 0 {
		db = db.Where("status IN ?", query.Status)
	}
	if query.Locale != "" {
		db = db.Where("locale = ?", query.Locale)
	}
	if trimmed := strings.TrimSpace(query.Search); trimmed != "" {
		pattern := "%" + strings.ToLower(trimmed) + "%"
		db = db.Where("(LOWER(title) LIKE ? OR LOWER(summary) LIKE ?)", pattern, pattern)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	var listings []*dbm.Listing
	err := db.Order("created_at DESC").
		Limit(query.Limit).
		Offset(query.Offset).
		Preload("Assets").
		Preload("PricingPlans").
		Preload("PricingPlans.Tiers").
		Find(&listings).Error
	if err != nil {
		return nil, 0, err
	}
	return listings, total, nil
}

// ReplaceAssets replaces all assets for a listing in a transaction.
func (r *ListingRepository) ReplaceAssets(ctx context.Context, tenantID, listingID string, assets []dbm.ListingAsset) error {
	tenantID = strings.TrimSpace(tenantID)
	listingID = strings.TrimSpace(listingID)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	if listingID == "" {
		return errors.New("listing_id is required")
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		if err := tx.Where("listing_id = ? AND tenant_uuid = ?", listingID, tenantID).
			Delete(&dbm.ListingAsset{}).Error; err != nil {
			return err
		}
		if len(assets) == 0 {
			return nil
		}
		for i := range assets {
			assets[i].ListingID = listingID
			assets[i].TenantUuid = tenantID
		}
		return tx.Create(&assets).Error
	})
}

// ReplacePricingPlans replaces pricing plans and tiers for a listing.
func (r *ListingRepository) ReplacePricingPlans(ctx context.Context, tenantID, listingID string, plans []dbm.PricingPlan) error {
	tenantID = strings.TrimSpace(tenantID)
	listingID = strings.TrimSpace(listingID)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	if listingID == "" {
		return errors.New("listing_id is required")
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		tiersTable := models.S(models.TableMarketplacePlanTiers)
		plansTable := models.S(models.TableMarketplacePricingPlans)

		if err := tx.Exec("DELETE FROM "+tiersTable+" WHERE plan_id IN (SELECT id FROM "+plansTable+" WHERE listing_id = ? AND tenant_uuid = ?)", listingID, tenantID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM "+plansTable+" WHERE listing_id = ? AND tenant_uuid = ?", listingID, tenantID).Error; err != nil {
			return err
		}
		if len(plans) == 0 {
			return nil
		}
		for i := range plans {
			plan := &plans[i]
			plan.ListingID = listingID
			plan.TenantUuid = tenantID
			tiers := plan.Tiers
			plan.Tiers = nil
			if err := tx.Create(plan).Error; err != nil {
				return err
			}
			if len(tiers) > 0 {
				for j := range tiers {
					tiers[j].PlanID = plan.ID
					tiers[j].TenantUuid = tenantID
				}
				if err := tx.Create(&tiers).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// CreateVersion records a listing version snapshot.
func (r *ListingRepository) CreateVersion(ctx context.Context, version *dbm.ListingVersion) error {
	if version == nil {
		return errors.New("version is required")
	}
	tenantID := strings.TrimSpace(version.TenantUuid)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	version.TenantUuid = tenantID
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Create(version).Error
	})
}

// UpdateRecommendedWeight updates the recommendation weight for a listing.
func (r *ListingRepository) UpdateRecommendedWeight(ctx context.Context, tenantID, listingID string, weight float64) error {
	tenantID = strings.TrimSpace(tenantID)
	listingID = strings.TrimSpace(listingID)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	if listingID == "" {
		return errors.New("listing_id is required")
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		res := tx.Model(&dbm.Listing{}).
			Where("id = ? AND tenant_uuid = ?", listingID, tenantID).
			Update("recommended_weight", weight)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// ListTenantUuids returns distinct tenant identifiers that have listings.
func (r *ListingRepository) ListTenantUuids(ctx context.Context) ([]string, error) {
	var ids []string
	if err := r.DB.WithContext(ctx).Model(&dbm.Listing{}).Distinct().Pluck("tenant_uuid", &ids).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (r *ListingRepository) baseQuery(ctx context.Context, tenantID string) *gorm.DB {
	db := r.DB.WithContext(ctx).Model(&dbm.Listing{})
	if tenantID != "" {
		db = db.Where("tenant_uuid = ?", tenantID)
	}
	return db
}

// TopRecommended returns published listings ordered by recommendation weight.
func (r *ListingRepository) TopRecommended(ctx context.Context, tenantID string, limit int) ([]*dbm.Listing, error) {
	db := r.baseQuery(ctx, tenantID).
		Where("status = ?", dbm.ListingStatusPublished).
		Order("recommended_weight DESC")
	if limit > 0 {
		db = db.Limit(limit)
	}
	var listings []*dbm.Listing
	if err := db.Preload("Assets").Preload("PricingPlans").Find(&listings).Error; err != nil {
		return nil, err
	}
	return listings, nil
}
