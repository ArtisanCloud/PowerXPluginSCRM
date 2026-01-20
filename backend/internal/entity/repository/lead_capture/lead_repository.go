package lead_capture

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

var ErrLeadNotFound = errors.New("lead not found")

type LeadRepository struct {
	*repository.BaseRepository[model.Lead]
}

func NewLeadRepository(db *gorm.DB) *LeadRepository {
	return &LeadRepository{BaseRepository: repository.NewBaseRepository[model.Lead](db)}
}

func (r *LeadRepository) Create(ctx context.Context, lead *model.Lead) (*model.Lead, error) {
	if lead == nil {
		return nil, errors.New("lead is required")
	}
	lead.TenantUUID = strings.ToLower(strings.TrimSpace(lead.TenantUUID))
	if lead.TenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if err := r.WithTenantTx(ctx, lead.TenantUUID, func(tx *gorm.DB) error {
		return tx.Create(lead).Error
	}); err != nil {
		return nil, err
	}
	return lead, nil
}

func (r *LeadRepository) GetByUUID(ctx context.Context, tenantUUID, leadUUID string) (*model.Lead, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, ErrLeadNotFound
	}
	var out model.Lead
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLeadNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LeadRepository) List(ctx context.Context, tenantUUID string) ([]*model.Lead, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, ErrLeadNotFound
	}
	var out []*model.Lead
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ?", tenantUUID).
		Order("created_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LeadRepository) HasMergeActivity(ctx context.Context, tenantUUID, leadUUID string) (bool, error) {
	if r == nil || r.DB == nil {
		return false, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return false, ErrLeadNotFound
	}
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&model.LeadActivity{}).
		Where("tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?", tenantUUID, leadUUID, model.LeadActivityTypeMerge).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *LeadRepository) MergeActivityFlags(ctx context.Context, tenantUUID string, leadUUIDs []string) (map[string]bool, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" || len(leadUUIDs) == 0 {
		return map[string]bool{}, nil
	}
	var rows []struct {
		LeadUUID string
	}
	err := r.DB.WithContext(ctx).
		Model(&model.LeadActivity{}).
		Select("lead_uuid").
		Where("tenant_uuid = ? AND activity_type = ? AND lead_uuid IN ?", tenantUUID, model.LeadActivityTypeMerge, leadUUIDs).
		Group("lead_uuid").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	flags := make(map[string]bool, len(rows))
	for _, row := range rows {
		flags[row.LeadUUID] = true
	}
	return flags, nil
}

func (r *LeadRepository) FindFirstByPhone(ctx context.Context, tenantUUID, phone string) (*model.Lead, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	phone = strings.TrimSpace(phone)
	if tenantUUID == "" || phone == "" {
		return nil, ErrLeadNotFound
	}
	var out model.Lead
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND phone = ?", tenantUUID, phone).
		Order("created_at ASC").
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLeadNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LeadRepository) FindFirstByEmail(ctx context.Context, tenantUUID, email string) (*model.Lead, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	email = strings.ToLower(strings.TrimSpace(email))
	if tenantUUID == "" || email == "" {
		return nil, ErrLeadNotFound
	}
	var out model.Lead
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND email = ?", tenantUUID, email).
		Order("created_at ASC").
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLeadNotFound
		}
		return nil, err
	}
	return &out, nil
}
