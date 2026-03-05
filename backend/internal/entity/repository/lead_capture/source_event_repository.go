package lead_capture

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

// LeadSourceEventRepository persists lead source trace events.
type LeadSourceEventRepository struct {
	*repository.BaseRepository[model.LeadSource]
}

func NewLeadSourceEventRepository(db *gorm.DB) *LeadSourceEventRepository {
	return &LeadSourceEventRepository{BaseRepository: repository.NewBaseRepository[model.LeadSource](db)}
}

func (r *LeadSourceEventRepository) CreateEventTx(ctx context.Context, tx *gorm.DB, event *model.LeadSource) (*model.LeadSource, error) {
	if tx == nil {
		return nil, errors.New("database transaction is nil")
	}
	if event == nil {
		return nil, errors.New("source event is required")
	}
	event.TenantUUID = strings.ToLower(strings.TrimSpace(event.TenantUUID))
	event.LeadUUID = strings.ToLower(strings.TrimSpace(event.LeadUUID))
	event.ChannelCode = strings.ToLower(strings.TrimSpace(event.ChannelCode))
	event.AppType = strings.ToLower(strings.TrimSpace(event.AppType))
	event.AccountUUID = strings.ToLower(strings.TrimSpace(event.AccountUUID))
	event.CampaignCode = strings.TrimSpace(event.CampaignCode)
	event.UTMSource = strings.TrimSpace(event.UTMSource)
	event.UTMMedium = strings.TrimSpace(event.UTMMedium)
	event.UTMCampaign = strings.TrimSpace(event.UTMCampaign)
	if event.TenantUUID == "" || event.LeadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	now := time.Now().UTC()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.UpdatedAt = now
	if err := tx.WithContext(ctx).Create(event).Error; err != nil {
		return nil, err
	}
	return event, nil
}

func (r *LeadSourceEventRepository) ListByLead(ctx context.Context, tenantUUID, leadUUID string) ([]*model.LeadSource, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return []*model.LeadSource{}, nil
	}
	var out []*model.LeadSource
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Order("created_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}
