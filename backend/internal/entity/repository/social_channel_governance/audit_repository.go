package social_channel_governance

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

type AuditRepository struct {
	*repository.BaseRepository[model.AuditEvent]
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{BaseRepository: repository.NewBaseRepository[model.AuditEvent](db)}
}

func (r *AuditRepository) Create(ctx context.Context, event *model.AuditEvent) (*model.AuditEvent, error) {
	if event == nil {
		return nil, errors.New("audit event is required")
	}
	event.TenantUuid = strings.ToLower(strings.TrimSpace(event.TenantUuid))
	event.AccountUUID = strings.ToLower(strings.TrimSpace(event.AccountUUID))
	event.ActorUserUUID = strings.ToLower(strings.TrimSpace(event.ActorUserUUID))
	event.EventType = strings.TrimSpace(event.EventType)
	if event.TenantUuid == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if event.AccountUUID == "" || event.EventType == "" || event.ActorUserUUID == "" {
		return nil, errors.New("account_uuid, event_type, and actor_user_uuid are required")
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	now := time.Now().UTC()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.UpdatedAt = now

	err := r.WithTenantTx(ctx, event.TenantUuid, func(tx *gorm.DB) error {
		return tx.Create(event).Error
	})
	if err != nil {
		return nil, err
	}
	return event, nil
}
