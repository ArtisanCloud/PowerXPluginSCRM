package integration

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WebhookSubscriptionRepository manages persistence for webhook subscriptions.
type WebhookSubscriptionRepository struct {
	*repository.BaseRepository[model.WebhookSubscription]
}

// NewWebhookSubscriptionRepository constructs a repository instance.
func NewWebhookSubscriptionRepository(db *gorm.DB) *WebhookSubscriptionRepository {
	return &WebhookSubscriptionRepository{
		BaseRepository: repository.NewBaseRepository[model.WebhookSubscription](db),
	}
}

// Upsert creates or updates a subscription under the specified tenant.
func (r *WebhookSubscriptionRepository) Upsert(ctx context.Context, sub *model.WebhookSubscription) (*model.WebhookSubscription, error) {
	if sub == nil {
		return nil, errors.New("webhook subscription is nil")
	}
	if strings.TrimSpace(sub.TenantUuid) == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	if strings.TrimSpace(sub.EventType) == "" {
		return nil, errors.New("event_type is required")
	}
	if strings.TrimSpace(sub.TargetURL) == "" {
		return nil, errors.New("target_url is required")
	}

	sub.UpdatedAt = time.Now().UTC()
	if sub.Status == "" {
		sub.Status = model.WebhookStatusActive
	}
	if sub.ID == "" {
		sub.ID = uuid.NewString()
	}

	err := r.WithTenantTx(ctx, sub.TenantUuid, func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "tenant_uuid"},
				{Name: "event_type"},
				{Name: "target_url"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"secret",
				"retry_policy",
				"status",
				"metadata",
				"updated_at",
			}),
		}).Create(sub).Error
	})
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// GetByID retrieves a subscription by tenant and identifier.
func (r *WebhookSubscriptionRepository) GetByID(ctx context.Context, tenantID, subscriptionID string) (*model.WebhookSubscription, error) {
	var sub model.WebhookSubscription
	if err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND id = ?", tenantID, subscriptionID).
		First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

// GetBySubscriptionID retrieves a subscription without tenant scoping (internal usage only).
func (r *WebhookSubscriptionRepository) GetBySubscriptionID(ctx context.Context, subscriptionID string) (*model.WebhookSubscription, error) {
	var sub model.WebhookSubscription
	if err := r.DB.WithContext(ctx).
		Where("id = ?", subscriptionID).
		First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

// ListByTenant returns all subscriptions for a tenant filtered by status.
func (r *WebhookSubscriptionRepository) ListByTenant(ctx context.Context, tenantID string, statuses []string) ([]*model.WebhookSubscription, error) {
	var subs []*model.WebhookSubscription
	query := r.DB.WithContext(ctx).Where("tenant_uuid = ?", tenantID)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	if err := query.Order("created_at DESC").Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}

// Delete removes a subscription.
func (r *WebhookSubscriptionRepository) Delete(ctx context.Context, tenantID, subscriptionID string) error {
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		res := tx.Where("tenant_uuid = ? AND id = ?", tenantID, subscriptionID).Delete(&model.WebhookSubscription{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// UpdateStatus updates subscription status.
func (r *WebhookSubscriptionRepository) UpdateStatus(ctx context.Context, tenantID, subscriptionID, status string) error {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		return errors.New("status is required")
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		res := tx.Model(&model.WebhookSubscription{}).
			Where("tenant_uuid = ? AND id = ?", tenantID, subscriptionID).
			Updates(map[string]any{
				"status":     status,
				"updated_at": time.Now().UTC(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// DebugLogSubscription outputs subscription information for debugging.
func (r *WebhookSubscriptionRepository) DebugLogSubscription(sub *model.WebhookSubscription) {
	if sub == nil {
		return
	}
	logger.WithField("subscription_id", sub.ID).
		WithField("tenant_uuid", sub.TenantUuid).
		WithField("event_type", sub.EventType).
		WithField("status", sub.Status).
		Debug("webhook subscription persisted")
}
