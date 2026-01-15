package integration

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SecretRepository 管理外部凭证生命周期数据。
type SecretRepository struct {
	*repository.BaseRepository[model.SecretCredential]
}

// NewSecretRepository 构造仓储实例。
func NewSecretRepository(db *gorm.DB) *SecretRepository {
	return &SecretRepository{
		BaseRepository: repository.NewBaseRepository[model.SecretCredential](db),
	}
}

// Create 插入新的凭证记录。
func (r *SecretRepository) Create(ctx context.Context, secret *model.SecretCredential) (*model.SecretCredential, error) {
	if secret == nil {
		return nil, errors.New("secret credential is nil")
	}
	if strings.TrimSpace(secret.TenantUuid) == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	if strings.TrimSpace(secret.IntegrationType) == "" {
		return nil, errors.New("integration_type is required")
	}
	if secret.Metadata == nil {
		secret.Metadata = datatypes.JSONMap{}
	}
	secret.Status = strings.ToUpper(strings.TrimSpace(secret.Status))
	if secret.Status == "" {
		secret.Status = model.SecretStatusActive
	}
	now := time.Now().UTC()
	secret.CreatedAt = now
	secret.UpdatedAt = now

	err := r.WithTenantTx(ctx, secret.TenantUuid, func(tx *gorm.DB) error {
		return tx.Create(secret).Error
	})
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// Update 更新凭证记录。
func (r *SecretRepository) Update(ctx context.Context, secret *model.SecretCredential) error {
	if secret == nil {
		return errors.New("secret credential is nil")
	}
	secret.UpdatedAt = time.Now().UTC()
	return r.WithTenantTx(ctx, secret.TenantUuid, func(tx *gorm.DB) error {
		return tx.Model(&model.SecretCredential{}).
			Where("id = ?", secret.ID).
			Updates(map[string]any{
				"current_secret_ref":     secret.CurrentSecretRef,
				"pending_secret_ref":     secret.PendingSecretRef,
				"rotation_interval_days": secret.RotationInterval,
				"last_rotated_at":        secret.LastRotatedAt,
				"next_rotation_due_at":   secret.NextRotationDueAt,
				"status":                 secret.Status,
				"audit_log":              secret.AuditLog,
				"metadata":               secret.Metadata,
				"updated_at":             secret.UpdatedAt,
			}).Error
	})
}

// GetByID 根据租户和 ID 获取凭证。
func (r *SecretRepository) GetByID(ctx context.Context, tenantID, secretID string) (*model.SecretCredential, error) {
	var secret model.SecretCredential
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND id = ?", tenantID, secretID).
		First(&secret).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &secret, nil
}

// GetByIntegrationType 获取某集成类型的凭证。
func (r *SecretRepository) GetByIntegrationType(ctx context.Context, tenantID, integrationType string) (*model.SecretCredential, error) {
	var secret model.SecretCredential
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND integration_type = ?", tenantID, integrationType).
		First(&secret).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &secret, nil
}

// ListByTenant 列出租户的全部凭证。
func (r *SecretRepository) ListByTenant(ctx context.Context, tenantID string) ([]*model.SecretCredential, error) {
	var secrets []*model.SecretCredential
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ?", tenantID).
		Order("created_at DESC").
		Find(&secrets).Error
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

// ListDueForRotation 返回即将到期需要提醒的凭证。
func (r *SecretRepository) ListDueForRotation(ctx context.Context, cutoff time.Time, limit int) ([]*model.SecretCredential, error) {
	var secrets []*model.SecretCredential
	query := r.DB.WithContext(ctx).
		Where("status = ? AND next_rotation_due_at IS NOT NULL AND next_rotation_due_at <= ?", model.SecretStatusActive, cutoff.UTC()).
		Order("next_rotation_due_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&secrets).Error; err != nil {
		return nil, err
	}
	return secrets, nil
}
