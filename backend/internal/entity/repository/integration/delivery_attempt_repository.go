package integration

import (
	"context"
	"errors"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeliveryAttemptRepository manages webhook delivery attempt records.
type DeliveryAttemptRepository struct {
	*repository.BaseRepository[model.DeliveryAttempt]
}

// NewDeliveryAttemptRepository constructs a new repository.
func NewDeliveryAttemptRepository(db *gorm.DB) *DeliveryAttemptRepository {
	return &DeliveryAttemptRepository{
		BaseRepository: repository.NewBaseRepository[model.DeliveryAttempt](db),
	}
}

// Create stores a new delivery attempt.
func (r *DeliveryAttemptRepository) Create(ctx context.Context, attempt *model.DeliveryAttempt) (*model.DeliveryAttempt, error) {
	if attempt == nil {
		return nil, errors.New("delivery attempt is nil")
	}
	if attempt.Status == "" {
		attempt.Status = model.AttemptStatusPending
	}
	attempt.CreatedAt = time.Now().UTC()
	attempt.UpdatedAt = attempt.CreatedAt
	if attempt.ID == "" {
		attempt.ID = uuid.NewString()
	}

	if err := r.DB.WithContext(ctx).Create(attempt).Error; err != nil {
		return nil, err
	}
	return attempt, nil
}

// UpdateStatus updates the status, error, and next delivery time for an attempt.
func (r *DeliveryAttemptRepository) UpdateStatus(ctx context.Context, attemptID string, status string, next time.Time, lastError string, retryCount int) error {
	updates := map[string]any{
		"status":      status,
		"updated_at":  time.Now().UTC(),
		"last_error":  lastError,
		"retry_count": retryCount,
	}
	if !next.IsZero() {
		updates["next_delivery_at"] = next.UTC()
	} else {
		updates["next_delivery_at"] = nil
	}

	res := r.DB.WithContext(ctx).
		Model(&model.DeliveryAttempt{}).
		Where("id = ?", attemptID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListDueForRetry returns attempts that should be retried at or before the given time.
func (r *DeliveryAttemptRepository) ListDueForRetry(ctx context.Context, cutoff time.Time, limit int) ([]*model.DeliveryAttempt, error) {
	var attempts []*model.DeliveryAttempt
	query := r.DB.WithContext(ctx).
		Where("(status = ? OR status = ?) AND next_delivery_at IS NOT NULL AND next_delivery_at <= ?", model.AttemptStatusPending, model.AttemptStatusRetrying, cutoff.UTC()).
		Order("next_delivery_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&attempts).Error; err != nil {
		return nil, err
	}
	return attempts, nil
}

// ListBySubscription returns attempts for the given subscription.
func (r *DeliveryAttemptRepository) ListBySubscription(ctx context.Context, subscriptionID string, limit int) ([]*model.DeliveryAttempt, error) {
	var attempts []*model.DeliveryAttempt
	query := r.DB.WithContext(ctx).
		Where("subscription_id = ?", subscriptionID).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&attempts).Error; err != nil {
		return nil, err
	}
	return attempts, nil
}

// GetByID fetches an attempt by identifier.
func (r *DeliveryAttemptRepository) GetByID(ctx context.Context, attemptID string) (*model.DeliveryAttempt, error) {
	var attempt model.DeliveryAttempt
	if err := r.DB.WithContext(ctx).Where("id = ?", attemptID).First(&attempt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &attempt, nil
}
