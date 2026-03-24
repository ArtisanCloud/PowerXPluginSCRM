package lead_capture

import (
	"context"
	"errors"
	"strings"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type channelCodeRepository struct{ gormStore }

type codeWelcomeConfigRepository struct{ gormStore }

type codeWelcomeSyncAttemptRepository struct{ gormStore }

type channelCodeEventRepository struct{ gormStore }

type leadAttributionRepository struct{ gormStore }

type codeConfigChangeLogRepository struct{ gormStore }

func NewChannelCodeRepository(db *gorm.DB) ChannelCodeRepository {
	return &channelCodeRepository{gormStore{db: db}}
}

func NewCodeWelcomeConfigRepository(db *gorm.DB) CodeWelcomeConfigRepository {
	return &codeWelcomeConfigRepository{gormStore{db: db}}
}

func NewCodeWelcomeSyncAttemptRepository(db *gorm.DB) CodeWelcomeSyncAttemptRepository {
	return &codeWelcomeSyncAttemptRepository{gormStore{db: db}}
}

func NewChannelCodeEventRepository(db *gorm.DB) ChannelCodeEventRepository {
	return &channelCodeEventRepository{gormStore{db: db}}
}

func NewLeadAttributionRepository(db *gorm.DB) LeadAttributionRepository {
	return &leadAttributionRepository{gormStore{db: db}}
}

func NewCodeConfigChangeLogRepository(db *gorm.DB) CodeConfigChangeLogRepository {
	return &codeConfigChangeLogRepository{gormStore{db: db}}
}

func (r *channelCodeRepository) Create(ctx context.Context, item *leadmodel.ChannelCode) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("channel code is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.CreatedAt = utcNow()
	item.UpdatedAt = item.CreatedAt
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *channelCodeRepository) GetByCodeUUID(ctx context.Context, tenantUUID, codeUUID string) (*leadmodel.ChannelCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	codeUUID, err = normalizeCodeUUID(codeUUID)
	if err != nil {
		return nil, err
	}
	var out leadmodel.ChannelCode
	q := r.db.WithContext(ctx).Where("tenant_uuid = ? AND code_uuid = ?", tenantUUID, codeUUID).First(&out)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, q.Error
	}
	return &out, nil
}

func (r *channelCodeRepository) List(ctx context.Context, tenantUUID string, filter ChannelCodeListFilter) ([]*leadmodel.ChannelCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	q := r.db.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID)
	if v := strings.TrimSpace(filter.Channel); v != "" {
		q = q.Where("channel = ?", strings.ToLower(v))
	}
	if v := strings.TrimSpace(filter.AppType); v != "" {
		q = q.Where("app_type = ?", strings.ToLower(v))
	}
	if v := strings.TrimSpace(filter.ChannelAccountUUID); v != "" {
		q = q.Where("channel_account_uuid = ?", strings.ToLower(v))
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		q = q.Where("status = ?", strings.ToLower(v))
	}
	limit := ensureLimit(filter.Limit, 20)
	var out []*leadmodel.ChannelCode
	if err := q.Order("created_at desc").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *channelCodeRepository) UpdateStatus(ctx context.Context, tenantUUID, codeUUID, status, updatedBy string) (*leadmodel.ChannelCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	codeUUID, err = normalizeCodeUUID(codeUUID)
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
	q := r.db.WithContext(ctx).Model(&leadmodel.ChannelCode{}).
		Where("tenant_uuid = ? AND code_uuid = ?", tenantUUID, codeUUID).
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
	return r.GetByCodeUUID(ctx, tenantUUID, codeUUID)
}

func (r *codeWelcomeConfigRepository) Save(ctx context.Context, item *leadmodel.CodeWelcomeConfig) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("welcome config is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.CodeUUID = strings.ToLower(strings.TrimSpace(item.CodeUUID))
	if _, err := normalizeCodeUUID(item.CodeUUID); err != nil {
		return err
	}
	now := utcNow()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code_uuid"}},
		DoUpdates: clause.Assignments(map[string]any{
			"welcome_enabled": item.WelcomeEnabled,
			"message_content": item.MessageContent,
			"sync_status":     item.SyncStatus,
			"last_sync_error": item.LastSyncError,
			"last_synced_at":  item.LastSyncedAt,
			"version":         item.Version,
			"updated_by":      item.UpdatedBy,
			"updated_at":      item.UpdatedAt,
		}),
	}).Create(item).Error
}

func (r *codeWelcomeConfigRepository) GetByCodeUUID(ctx context.Context, tenantUUID, codeUUID string) (*leadmodel.CodeWelcomeConfig, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	codeUUID, err = normalizeCodeUUID(codeUUID)
	if err != nil {
		return nil, err
	}
	var out leadmodel.CodeWelcomeConfig
	q := r.db.WithContext(ctx).Where("tenant_uuid = ? AND code_uuid = ?", tenantUUID, codeUUID).First(&out)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, q.Error
	}
	return &out, nil
}

func (r *codeWelcomeSyncAttemptRepository) Create(ctx context.Context, item *leadmodel.CodeWelcomeSyncAttempt) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("welcome sync attempt is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.CodeUUID = strings.ToLower(strings.TrimSpace(item.CodeUUID))
	if _, err := normalizeCodeUUID(item.CodeUUID); err != nil {
		return err
	}
	item.CreatedAt = utcNow()
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *codeWelcomeSyncAttemptRepository) ListByCodeUUID(ctx context.Context, tenantUUID, codeUUID string, limit int) ([]*leadmodel.CodeWelcomeSyncAttempt, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	codeUUID, err = normalizeCodeUUID(codeUUID)
	if err != nil {
		return nil, err
	}
	limit = ensureLimit(limit, 20)
	var out []*leadmodel.CodeWelcomeSyncAttempt
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND code_uuid = ?", tenantUUID, codeUUID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *channelCodeEventRepository) Create(ctx context.Context, item *leadmodel.ChannelCodeEvent) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("channel code event is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	if strings.TrimSpace(item.IdempotencyKey) == "" {
		return errors.New("idempotency_key is required")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = utcNow()
	}
	if item.OccurredAt.IsZero() {
		item.OccurredAt = item.CreatedAt
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *channelCodeEventRepository) GetByIdempotencyKey(ctx context.Context, tenantUUID, idempotencyKey string) (*leadmodel.ChannelCodeEvent, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return nil, errors.New("idempotency_key is required")
	}
	var out leadmodel.ChannelCodeEvent
	q := r.db.WithContext(ctx).Where("tenant_uuid = ? AND idempotency_key = ?", tenantUUID, idempotencyKey).First(&out)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, q.Error
	}
	return &out, nil
}

func (r *leadAttributionRepository) Create(ctx context.Context, item *leadmodel.LeadAttributionRecord) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("attribution record is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	if strings.TrimSpace(item.LeadUUID) == "" {
		return errors.New("lead_uuid is required")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = utcNow()
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *leadAttributionRepository) ListByLeadUUID(ctx context.Context, tenantUUID, leadUUID string) ([]*leadmodel.LeadAttributionRecord, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if leadUUID == "" {
		return nil, errors.New("lead_uuid is required")
	}
	var out []*leadmodel.LeadAttributionRecord
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Order("created_at asc").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *codeConfigChangeLogRepository) Create(ctx context.Context, item *leadmodel.CodeConfigChangeLog) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("config change log is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.CodeUUID = strings.ToLower(strings.TrimSpace(item.CodeUUID))
	if _, err := normalizeCodeUUID(item.CodeUUID); err != nil {
		return err
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = utcNow()
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *codeConfigChangeLogRepository) ListByCodeUUID(ctx context.Context, tenantUUID, codeUUID string, limit int) ([]*leadmodel.CodeConfigChangeLog, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	codeUUID, err = normalizeCodeUUID(codeUUID)
	if err != nil {
		return nil, err
	}
	limit = ensureLimit(limit, 20)
	var out []*leadmodel.CodeConfigChangeLog
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND code_uuid = ?", tenantUUID, codeUUID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}
