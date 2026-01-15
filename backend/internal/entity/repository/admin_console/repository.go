package admin_console

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

// Store provides tenant-scoped helpers backed by BaseRepository.
type Store struct {
	*repository.BaseRepository[struct{}]
}

// NewStore constructs a BaseRepository-backed admin console store.
func NewStore(db *gorm.DB) *Store {
	return &Store{
		BaseRepository: repository.NewBaseRepository[struct{}](db),
	}
}

// WithTenant executes a scoped transaction that sets app.tenant_uuid.
func (s *Store) WithTenant(ctx context.Context, tenantID string, fn func(tx *gorm.DB) error) error {
	return s.BaseRepository.WithTenantTx(ctx, tenantID, fn)
}
