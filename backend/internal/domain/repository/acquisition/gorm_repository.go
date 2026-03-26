package acquisition

import (
	"context"
	"errors"
	"strings"
	"time"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	entitymodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	orgsyncmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type staffLiveCodeRepository struct{ gormStore }
type staffWelcomeConfigRepository struct{ gormStore }
type staffWelcomeSyncAttemptRepository struct{ gormStore }
type groupLiveCodeRepository struct{ gormStore }

func NewStaffLiveCodeRepository(db *gorm.DB) StaffLiveCodeRepository {
	return &staffLiveCodeRepository{gormStore{db: db}}
}

func NewStaffWelcomeConfigRepository(db *gorm.DB) StaffWelcomeConfigRepository {
	return &staffWelcomeConfigRepository{gormStore{db: db}}
}

func NewStaffWelcomeSyncAttemptRepository(db *gorm.DB) StaffWelcomeSyncAttemptRepository {
	return &staffWelcomeSyncAttemptRepository{gormStore{db: db}}
}

func NewGroupLiveCodeRepository(db *gorm.DB) GroupLiveCodeRepository {
	return &groupLiveCodeRepository{gormStore{db: db}}
}

func (r *staffLiveCodeRepository) Create(ctx context.Context, item *acqmodel.StaffLiveCode) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("staff live code is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.CreatedAt = utcNow()
	item.UpdatedAt = item.CreatedAt
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *staffLiveCodeRepository) GetByUUID(ctx context.Context, tenantUUID, staffCodeUUID string) (*acqmodel.StaffLiveCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	staffCodeUUID, err = normalizeStaffCodeUUID(staffCodeUUID)
	if err != nil {
		return nil, err
	}
	var out acqmodel.StaffLiveCode
	q := r.db.WithContext(ctx).Where("tenant_uuid = ? AND staff_code_uuid = ?", tenantUUID, staffCodeUUID).First(&out)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, q.Error
	}
	return &out, nil
}

func (r *staffLiveCodeRepository) List(ctx context.Context, tenantUUID string, filter StaffLiveCodeListFilter) ([]*acqmodel.StaffLiveCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	q := r.db.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID)
	if v := strings.TrimSpace(filter.ActivityName); v != "" {
		q = q.Where("activity_name ILIKE ?", "%"+v+"%")
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		q = q.Where("status = ?", strings.ToLower(v))
	}
	limit := ensureLimit(filter.Limit, 20)
	var out []*acqmodel.StaffLiveCode
	if err := q.Order("created_at desc").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *staffLiveCodeRepository) UpdateStatus(ctx context.Context, tenantUUID, staffCodeUUID, status, updatedBy string) (*acqmodel.StaffLiveCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	staffCodeUUID, err = normalizeStaffCodeUUID(staffCodeUUID)
	if err != nil {
		return nil, err
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return nil, errors.New("status is required")
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		updatedBy = "system"
	}
	now := time.Now().UTC()
	q := r.db.WithContext(ctx).Model(&acqmodel.StaffLiveCode{}).
		Where("tenant_uuid = ? AND staff_code_uuid = ?", tenantUUID, staffCodeUUID).
		Updates(map[string]any{
			"status":     status,
			"updated_by": updatedBy,
			"updated_at": now,
		})
	if q.Error != nil {
		return nil, q.Error
	}
	if q.RowsAffected == 0 {
		return nil, ErrRecordNotFound
	}
	return r.GetByUUID(ctx, tenantUUID, staffCodeUUID)
}

func (r *staffLiveCodeRepository) CountConfirmedMappings(ctx context.Context, tenantUUID string, memberUUIDs []string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return 0, err
	}
	if len(memberUUIDs) == 0 {
		return 0, nil
	}
	cleanIDs := make([]string, 0, len(memberUUIDs))
	seen := map[string]struct{}{}
	for _, id := range memberUUIDs {
		id = strings.ToLower(strings.TrimSpace(id))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		cleanIDs = append(cleanIDs, id)
	}
	if len(cleanIDs) == 0 {
		return 0, nil
	}
	var count int64
	err = r.db.WithContext(ctx).
		Model(&orgsyncmodel.MemberMapping{}).
		Where("tenant_uuid = ? AND mapping_status = ? AND source_member_uuid IN ?", tenantUUID, orgsyncmodel.MappingStatusConfirmed, cleanIDs).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *staffLiveCodeRepository) ExistsByCodeKey(ctx context.Context, tenantUUID, codeKey string) (bool, error) {
	if r == nil || r.db == nil {
		return false, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return false, err
	}
	codeKey = strings.ToLower(strings.TrimSpace(codeKey))
	if codeKey == "" {
		return false, errors.New("code_key is required")
	}
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&acqmodel.StaffLiveCode{}).
		Where("tenant_uuid = ? AND lower(code_key) = ?", tenantUUID, codeKey).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *staffWelcomeConfigRepository) Save(ctx context.Context, item *acqmodel.StaffWelcomeConfig) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("staff welcome config is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.StaffCodeUUID = strings.ToLower(strings.TrimSpace(item.StaffCodeUUID))
	if _, err := normalizeStaffCodeUUID(item.StaffCodeUUID); err != nil {
		return err
	}
	now := utcNow()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "staff_code_uuid"}},
		DoUpdates: clause.Assignments(map[string]any{
			"welcome_mode":    item.WelcomeMode,
			"content_blocks":  item.ContentBlocks,
			"payload_preview": item.PayloadPreview,
			"sync_status":     item.SyncStatus,
			"last_sync_error": item.LastSyncError,
			"last_synced_at":  item.LastSyncedAt,
			"version":         item.Version,
			"updated_by":      item.UpdatedBy,
			"updated_at":      item.UpdatedAt,
		}),
	}).Create(item).Error
}

func (r *staffWelcomeConfigRepository) GetByStaffCodeUUID(ctx context.Context, tenantUUID, staffCodeUUID string) (*acqmodel.StaffWelcomeConfig, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	staffCodeUUID, err = normalizeStaffCodeUUID(staffCodeUUID)
	if err != nil {
		return nil, err
	}
	var out acqmodel.StaffWelcomeConfig
	q := r.db.WithContext(ctx).Where("tenant_uuid = ? AND staff_code_uuid = ?", tenantUUID, staffCodeUUID).First(&out)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, q.Error
	}
	return &out, nil
}

func (r *staffWelcomeSyncAttemptRepository) Create(ctx context.Context, item *acqmodel.StaffWelcomeSyncAttempt) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("staff welcome sync attempt is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.StaffCodeUUID = strings.ToLower(strings.TrimSpace(item.StaffCodeUUID))
	if _, err := normalizeStaffCodeUUID(item.StaffCodeUUID); err != nil {
		return err
	}
	item.CreatedAt = utcNow()
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *staffWelcomeSyncAttemptRepository) ListByStaffCodeUUID(ctx context.Context, tenantUUID, staffCodeUUID string, limit int) ([]*acqmodel.StaffWelcomeSyncAttempt, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	staffCodeUUID, err = normalizeStaffCodeUUID(staffCodeUUID)
	if err != nil {
		return nil, err
	}
	limit = ensureLimit(limit, 20)
	var out []*acqmodel.StaffWelcomeSyncAttempt
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND staff_code_uuid = ?", tenantUUID, staffCodeUUID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *groupLiveCodeRepository) List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupLiveCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	limit = ensureLimit(limit, 20)
	var out []*acqmodel.GroupLiveCode
	err = r.db.WithContext(ctx).
		Table(entitymodels.S(entitymodels.TableAcquisitionGroupLiveCodes)).
		Where("tenant_uuid = ?", tenantUUID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}
