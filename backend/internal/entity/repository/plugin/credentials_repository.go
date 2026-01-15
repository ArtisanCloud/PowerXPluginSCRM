package plugin

import (
	"context"
	"errors"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CredentialsRepository persists plugin credentials per tenant.
type CredentialsRepository struct {
	*repository.BaseRepository[models.PluginCredential]
}

// NewCredentialsRepository constructs a repository using the shared BaseRepository pattern.
func NewCredentialsRepository(db *gorm.DB) *CredentialsRepository {
	return &CredentialsRepository{BaseRepository: repository.NewBaseRepository[models.PluginCredential](db)}
}

// Upsert saves or updates the credential record for a tenant-plugin pair.
func (r *CredentialsRepository) Upsert(ctx context.Context, pc *models.PluginCredential) error {
	if pc == nil {
		return errors.New("nil credential model")
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_uuid"},
			{Name: "plugin_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"client_id", "secret_ciphertext", "iv_nonce", "key_version", "updated_at"}),
	}).Create(pc).Error
}

// GetByTenantPlugin fetches credentials for a tenant-plugin combination.
func (r *CredentialsRepository) GetByTenantPlugin(ctx context.Context, tenantUUID string, pluginID string) (*models.PluginCredential, error) {
	var pc models.PluginCredential
	if err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND plugin_id = ?", tenantUUID, pluginID).
		First(&pc).Error; err != nil {
		return nil, err
	}
	return &pc, nil
}
