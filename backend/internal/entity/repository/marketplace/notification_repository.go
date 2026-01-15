package marketplace

import (
	"context"
	"errors"
	"strings"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationRepository manages marketplace notification entries.
type NotificationRepository struct {
	*repository.BaseRepository[dbm.Notification]
}

// NewNotificationRepository constructs repository instance.
func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{
		BaseRepository: repository.NewBaseRepository[dbm.Notification](db),
	}
}

// QueueNotification stores notification for asynchronous dispatch.
func (r *NotificationRepository) QueueNotification(ctx context.Context, notification *dbm.Notification) error {
	if notification == nil {
		return errors.New("notification is required")
	}
	tenantID := strings.TrimSpace(notification.TenantUuid)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	if strings.TrimSpace(notification.ID) == "" {
		notification.ID = uuid.NewString()
	}
	if notification.Status == "" {
		notification.Status = dbm.NotificationStatusPending
	}
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		if notification.CreatedAt.IsZero() {
			notification.CreatedAt = time.Now().UTC()
		}
		return tx.Create(notification).Error
	})
}

// ListByTenant returns queued notifications for a tenant ordered by creation time.
func (r *NotificationRepository) ListByTenant(ctx context.Context, tenantID string) ([]*dbm.Notification, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	var notifications []*dbm.Notification
	if err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ?", tenantID).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		return nil, err
	}
	return notifications, nil
}
