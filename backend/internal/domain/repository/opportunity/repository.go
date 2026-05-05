package opportunity

import (
	"context"
	"errors"
	"strings"

	oppmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/opportunity"
	"gorm.io/gorm"
)

var (
	ErrDBNotReady        = errors.New("opportunity repository database is not initialized")
	ErrTenantUUIDMissing = errors.New("tenant_uuid is required")
)

type OpportunityRepository interface {
	Create(ctx context.Context, item *oppmodel.OpportunityRecord) error
	GetByUUID(ctx context.Context, tenantUUID, opportunityUUID string) (*oppmodel.OpportunityRecord, error)
	Update(ctx context.Context, item *oppmodel.OpportunityRecord) error
	BeginTenantTx(ctx context.Context, tenantUUID string, fn func(tx *gorm.DB) error) error
}

type OpportunityActivityRepository interface {
	Create(ctx context.Context, item *oppmodel.OpportunityActivity) error
	ListByOpportunity(ctx context.Context, tenantUUID, opportunityUUID string, limit int) ([]*oppmodel.OpportunityActivity, error)
}

type gormStore struct {
	db *gorm.DB
}

func normalizeTenantUUID(tenantUUID string) (string, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return "", ErrTenantUUIDMissing
	}
	return tenantUUID, nil
}
