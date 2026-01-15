package admin_console

import (
	"context"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

// ConfigChangeRepository manages configuration change history.
type ConfigChangeRepository struct {
	*repository.BaseRepository[model.ConfigChange]
}

// NewConfigChangeRepository constructs repository instance.
func NewConfigChangeRepository(db *gorm.DB) *ConfigChangeRepository {
	return &ConfigChangeRepository{BaseRepository: repository.NewBaseRepository[model.ConfigChange](db)}
}

// Create writes a new configuration change record.
func (r *ConfigChangeRepository) Create(ctx context.Context, change *model.ConfigChange) error {
	return r.DB.WithContext(ctx).Create(change).Error
}

// LatestBySection fetches the newest change for a section and tenant scope.
func (r *ConfigChangeRepository) LatestBySection(ctx context.Context, pluginID string, tenantID *string, sectionKey string) (*model.ConfigChange, error) {
	query := r.DB.WithContext(ctx).
		Where("plugin_id = ? AND section_key = ?", pluginID, sectionKey).
		Preload("AuditEvent").
		Order("applied_at DESC")
	if tenantID == nil {
		query = query.Where("tenant_uuid IS NULL")
	} else {
		query = query.Where("tenant_uuid = ?", *tenantID)
	}
	var change model.ConfigChange
	if err := query.First(&change).Error; err != nil {
		return nil, err
	}
	return &change, nil
}
