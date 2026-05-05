package opportunity

import (
	"context"
	"fmt"
	"strings"
	"time"

	oppmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/opportunity"
	"gorm.io/gorm"
)

type opportunityRepository struct{ gormStore }
type opportunityActivityRepository struct{ gormStore }

func NewOpportunityRepository(db *gorm.DB) OpportunityRepository {
	return &opportunityRepository{gormStore{db: db}}
}

func NewOpportunityActivityRepository(db *gorm.DB) OpportunityActivityRepository {
	return &opportunityActivityRepository{gormStore{db: db}}
}

func (r *opportunityRepository) Create(ctx context.Context, item *oppmodel.OpportunityRecord) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	if item == nil {
		return fmt.Errorf("opportunity record is required")
	}
	tenantUUID, err := normalizeTenantUUID(item.TenantUUID)
	if err != nil {
		return err
	}
	item.TenantUUID = tenantUUID
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *opportunityRepository) GetByUUID(ctx context.Context, tenantUUID, opportunityUUID string) (*oppmodel.OpportunityRecord, error) {
	if r == nil || r.db == nil {
		return nil, ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return nil, err
	}
	opportunityUUID = strings.ToLower(strings.TrimSpace(opportunityUUID))
	if opportunityUUID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var out oppmodel.OpportunityRecord
	if err := r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).
		First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *opportunityRepository) Update(ctx context.Context, item *oppmodel.OpportunityRecord) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	if item == nil {
		return fmt.Errorf("opportunity record is required")
	}
	tenantUUID, err := normalizeTenantUUID(item.TenantUUID)
	if err != nil {
		return err
	}
	opportunityUUID := strings.ToLower(strings.TrimSpace(item.OpportunityUUID))
	if opportunityUUID == "" {
		return gorm.ErrRecordNotFound
	}
	item.UpdatedAt = time.Now().UTC()
	q := r.db.WithContext(ctx).
		Model(&oppmodel.OpportunityRecord{}).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).
		Updates(map[string]any{
			"title":             item.Title,
			"stage":             item.Stage,
			"amount":            item.Amount,
			"currency":          item.Currency,
			"owner_user_uuid":   item.OwnerUserUUID,
			"expected_close_at": item.ExpectedCloseAt,
			"won_at":            item.WonAt,
			"lost_at":           item.LostAt,
			"lost_reason":       item.LostReason,
			"risk_flags":        item.RiskFlags,
			"updated_by":        item.UpdatedBy,
			"updated_at":        item.UpdatedAt,
		})
	return q.Error
}

func (r *opportunityRepository) BeginTenantTx(ctx context.Context, tenantUUID string, fn func(tx *gorm.DB) error) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL app.tenant_uuid = ?", tenantUUID).Error; err != nil {
			return err
		}
		return fn(tx)
	})
}

func (r *opportunityActivityRepository) Create(ctx context.Context, item *oppmodel.OpportunityActivity) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	if item == nil {
		return fmt.Errorf("opportunity activity is required")
	}
	tenantUUID, err := normalizeTenantUUID(item.TenantUUID)
	if err != nil {
		return err
	}
	item.TenantUUID = tenantUUID
	item.CreatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *opportunityActivityRepository) ListByOpportunity(ctx context.Context, tenantUUID, opportunityUUID string, limit int) ([]*oppmodel.OpportunityActivity, error) {
	if r == nil || r.db == nil {
		return nil, ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return nil, err
	}
	opportunityUUID = strings.ToLower(strings.TrimSpace(opportunityUUID))
	if opportunityUUID == "" {
		return []*oppmodel.OpportunityActivity{}, nil
	}
	if limit <= 0 {
		limit = 100
	}
	var out []*oppmodel.OpportunityActivity
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error
	return out, err
}
