package lead_capture

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrChannelRuleNotFound = errors.New("channel rule not found")
)

type ChannelRuleRepository struct {
	*repository.BaseRepository[model.ChannelRule]
}

func NewChannelRuleRepository(db *gorm.DB) *ChannelRuleRepository {
	return &ChannelRuleRepository{BaseRepository: repository.NewBaseRepository[model.ChannelRule](db)}
}

func (r *ChannelRuleRepository) GetByChannelApp(ctx context.Context, tenantUUID, channel, appType string) (*model.ChannelRule, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channel = strings.ToLower(strings.TrimSpace(channel))
	appType = strings.ToLower(strings.TrimSpace(appType))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var out model.ChannelRule
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel = ? AND app_type = ?", tenantUUID, channel, appType).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChannelRuleNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *ChannelRuleRepository) UpsertAutoCreateLeadRule(ctx context.Context, tenantUUID, channel, appType string, enabled bool) (*model.ChannelRule, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channel = strings.ToLower(strings.TrimSpace(channel))
	appType = strings.ToLower(strings.TrimSpace(appType))
	if tenantUUID == "" || channel == "" || appType == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	now := time.Now().UTC()
	entity := &model.ChannelRule{
		RuleUUID:                     uuid.NewString(),
		TenantUUID:                   tenantUUID,
		Channel:                      channel,
		AppType:                      appType,
		AutoCreateLeadFromCustomerDM: enabled,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}
	if err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "tenant_uuid"},
				{Name: "channel"},
				{Name: "app_type"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"auto_create_lead_from_customer_dm": enabled,
				"updated_at":                        now,
			}),
		}).Create(entity).Error
	}); err != nil {
		return nil, err
	}
	return r.GetByChannelApp(ctx, tenantUUID, channel, appType)
}
