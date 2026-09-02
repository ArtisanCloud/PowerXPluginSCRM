package lead_capture

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

type LeadTimelineSnapshot struct {
	Activities    []*model.LeadActivity
	Assignments   []*model.LeadAssignment
	StatusHistory []*model.LeadStatusHistory
	Sources       []*model.LeadSource
}

func (r *LeadRepository) ListTimelineSnapshot(ctx context.Context, tenantUUID, leadUUID string) (*LeadTimelineSnapshot, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if leadUUID == "" {
		return nil, ErrLeadNotFound
	}
	out := &LeadTimelineSnapshot{}
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if err := ensureLeadExists(ctx, tx, tenantUUID, leadUUID); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).Find(&out.Activities).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).Find(&out.Assignments).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).Find(&out.StatusHistory).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).Find(&out.Sources).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
