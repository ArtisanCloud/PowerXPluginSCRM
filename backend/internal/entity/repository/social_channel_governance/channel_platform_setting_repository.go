package social_channel_governance

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrChannelPlatformSettingNotFound = errors.New("channel platform setting not found")

type ChannelPlatformSettingRepository struct {
	*repository.BaseRepository[model.ChannelPlatformSetting]
}

func NewChannelPlatformSettingRepository(db *gorm.DB) *ChannelPlatformSettingRepository {
	return &ChannelPlatformSettingRepository{BaseRepository: repository.NewBaseRepository[model.ChannelPlatformSetting](db)}
}

func (r *ChannelPlatformSettingRepository) GetByChannelProvider(ctx context.Context, channelCode, providerCode string) (*model.ChannelPlatformSetting, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	channelCode = strings.ToLower(strings.TrimSpace(channelCode))
	providerCode = strings.ToLower(strings.TrimSpace(providerCode))
	if channelCode == "" || providerCode == "" {
		return nil, ErrChannelPlatformSettingNotFound
	}
	var out model.ChannelPlatformSetting
	if err := r.DB.WithContext(ctx).
		Where("channel_code = ? AND provider_code = ?", channelCode, providerCode).
		First(&out).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChannelPlatformSettingNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *ChannelPlatformSettingRepository) UpsertByChannelProvider(
	ctx context.Context,
	channelCode, providerCode string,
	enabled bool,
	config datatypes.JSONMap,
) (*model.ChannelPlatformSetting, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	channelCode = strings.ToLower(strings.TrimSpace(channelCode))
	providerCode = strings.ToLower(strings.TrimSpace(providerCode))
	if channelCode == "" || providerCode == "" {
		return nil, errors.New("channel_code and provider_code are required")
	}
	if config == nil {
		config = datatypes.JSONMap{}
	}

	now := time.Now().UTC()
	record := &model.ChannelPlatformSetting{
		ChannelCode:  channelCode,
		ProviderCode: providerCode,
		Enabled:      enabled,
		Config:       config,
		UpdatedAt:    now,
	}
	if err := r.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "channel_code"},
				{Name: "provider_code"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"enabled":    enabled,
				"config":     config,
				"updated_at": now,
				"deleted_at": nil,
			}),
		}).
		Create(record).Error; err != nil {
		return nil, err
	}
	return r.GetByChannelProvider(ctx, channelCode, providerCode)
}
