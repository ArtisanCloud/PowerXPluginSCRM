package marketplace

import (
	"context"
	"errors"
	"strings"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PricingRepository manages pricing plans and tiers.
type PricingRepository struct {
	*repository.BaseRepository[dbm.PricingPlan]
}

// NewPricingRepository constructs a new repository instance.
func NewPricingRepository(db *gorm.DB) *PricingRepository {
	return &PricingRepository{
		BaseRepository: repository.NewBaseRepository[dbm.PricingPlan](db),
	}
}

// ListPlans returns plans for a listing with tiers preloaded.
func (r *PricingRepository) ListPlans(ctx context.Context, tenantID, listingID string) ([]*dbm.PricingPlan, error) {
	tenantID = strings.TrimSpace(tenantID)
	listingID = strings.TrimSpace(listingID)
	if tenantID == "" || listingID == "" {
		return nil, errors.New("tenant_uuid and listing_id are required")
	}
	var plans []*dbm.PricingPlan
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND listing_id = ?", tenantID, listingID).
		Order("created_at ASC").
		Preload("Tiers", func(tx *gorm.DB) *gorm.DB { return tx.Order("range_from ASC") }).
		Find(&plans).Error
	return plans, err
}

// GetPlan retrieves a single plan by identifier.
func (r *PricingRepository) GetPlan(ctx context.Context, tenantID, planID string) (*dbm.PricingPlan, error) {
	tenantID = strings.TrimSpace(tenantID)
	planID = strings.TrimSpace(planID)
	if tenantID == "" || planID == "" {
		return nil, errors.New("tenant_uuid and plan_id are required")
	}
	var plan dbm.PricingPlan
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND id = ?", tenantID, planID).
		Preload("Tiers", func(tx *gorm.DB) *gorm.DB { return tx.Order("range_from ASC") }).
		First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// CreatePlan persists a plan with optional tiers inside tenant transaction.
func (r *PricingRepository) CreatePlan(ctx context.Context, plan *dbm.PricingPlan, tiers []dbm.PlanTier) error {
	if plan == nil {
		return errors.New("plan is required")
	}
	tenantID := strings.TrimSpace(plan.TenantUuid)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	if strings.TrimSpace(plan.ID) == "" {
		plan.ID = uuid.NewString()
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		if err := tx.Create(plan).Error; err != nil {
			return err
		}
		if len(tiers) == 0 {
			return nil
		}
		for i := range tiers {
			if strings.TrimSpace(tiers[i].ID) == "" {
				tiers[i].ID = uuid.NewString()
			}
			tiers[i].PlanID = plan.ID
			tiers[i].TenantUuid = tenantID
		}
		return tx.Create(&tiers).Error
	})
}

// UpdatePlan updates plan fields and replaces tiers.
func (r *PricingRepository) UpdatePlan(ctx context.Context, plan *dbm.PricingPlan, tiers []dbm.PlanTier) error {
	if plan == nil {
		return errors.New("plan is required")
	}
	tenantID := strings.TrimSpace(plan.TenantUuid)
	if tenantID == "" || strings.TrimSpace(plan.ID) == "" {
		return errors.New("tenant_uuid and plan id are required")
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		if err := tx.Model(&dbm.PricingPlan{}).
			Where("id = ? AND tenant_uuid = ?", plan.ID, tenantID).
			Omit("id", "tenant_uuid", "listing_id").
			Updates(plan).Error; err != nil {
			return err
		}
		if err := tx.Where("plan_id = ?", plan.ID).Delete(&dbm.PlanTier{}).Error; err != nil {
			return err
		}
		if len(tiers) == 0 {
			return nil
		}
		for i := range tiers {
			if strings.TrimSpace(tiers[i].ID) == "" {
				tiers[i].ID = uuid.NewString()
			}
			tiers[i].PlanID = plan.ID
			tiers[i].TenantUuid = tenantID
		}
		return tx.Create(&tiers).Error
	})
}

// DeletePlan removes a plan and its tiers.
func (r *PricingRepository) DeletePlan(ctx context.Context, tenantID, planID string) error {
	tenantID = strings.TrimSpace(tenantID)
	planID = strings.TrimSpace(planID)
	if tenantID == "" || planID == "" {
		return errors.New("tenant_uuid and plan_id are required")
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		if err := tx.Where("plan_id = ?", planID).Delete(&dbm.PlanTier{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ? AND tenant_uuid = ?", planID, tenantID).Delete(&dbm.PricingPlan{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// SetDefaultPlan marks a plan as default and unsets others for the listing.
func (r *PricingRepository) SetDefaultPlan(ctx context.Context, tenantID, listingID, planID string) error {
	tenantID = strings.TrimSpace(tenantID)
	listingID = strings.TrimSpace(listingID)
	planID = strings.TrimSpace(planID)
	if tenantID == "" || listingID == "" || planID == "" {
		return errors.New("tenant_uuid, listing_id and plan_id are required")
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		if err := tx.Model(&dbm.PricingPlan{}).
			Where("tenant_uuid = ? AND listing_id = ?", tenantID, listingID).
			Update("is_default", false).Error; err != nil {
			return err
		}
		res := tx.Model(&dbm.PricingPlan{}).
			Where("tenant_uuid = ? AND id = ?", tenantID, planID).
			Update("is_default", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
