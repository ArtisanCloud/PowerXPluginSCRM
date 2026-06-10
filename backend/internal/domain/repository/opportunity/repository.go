package opportunity

import (
	"context"
	"errors"
	"strings"
	"time"

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
	List(ctx context.Context, tenantUUID string, filter OpportunityListFilter) ([]*oppmodel.OpportunityRecord, error)
	Dashboard(ctx context.Context, tenantUUID string, filter OpportunityListFilter) (*OpportunityDashboard, error)
	FindActiveByLead(ctx context.Context, tenantUUID, leadUUID string) (*oppmodel.OpportunityRecord, error)
	Update(ctx context.Context, item *oppmodel.OpportunityRecord) error
	BeginTenantTx(ctx context.Context, tenantUUID string, fn func(tx *gorm.DB) error) error
}

type OpportunityLineItemRepository interface {
	ListByOpportunity(ctx context.Context, tenantUUID, opportunityUUID string) ([]*oppmodel.OpportunityLineItem, error)
	Create(ctx context.Context, item *oppmodel.OpportunityLineItem) error
	Delete(ctx context.Context, tenantUUID, opportunityUUID, itemUUID string) error
}

type OpportunityTaskRepository interface {
	ListByOpportunity(ctx context.Context, tenantUUID, opportunityUUID string) ([]*oppmodel.OpportunityTask, error)
	Create(ctx context.Context, item *oppmodel.OpportunityTask) error
	Update(ctx context.Context, item *oppmodel.OpportunityTask) error
	GetByUUID(ctx context.Context, tenantUUID, opportunityUUID, taskUUID string) (*oppmodel.OpportunityTask, error)
}

type OpportunityActivityRepository interface {
	Create(ctx context.Context, item *oppmodel.OpportunityActivity) error
	ListByOpportunity(ctx context.Context, tenantUUID, opportunityUUID string, limit int) ([]*oppmodel.OpportunityActivity, error)
}

type gormStore struct {
	db *gorm.DB
}

type OpportunityListFilter struct {
	Stage             string
	OwnerUserUUID     string
	LeadUUID          string
	Keyword           string
	SourceChannel     string
	RiskOnly          bool
	ExpectedCloseFrom *time.Time
	ExpectedCloseTo   *time.Time
	Limit             int
}

type OpportunityDashboard struct {
	ActiveCount    int64                     `json:"active_count"`
	PipelineAmount float64                   `json:"pipeline_amount"`
	WonAmount      float64                   `json:"won_amount"`
	RiskCount      int64                     `json:"risk_count"`
	StageSummaries []OpportunityStageSummary `json:"stage_summaries"`
}

type OpportunityStageSummary struct {
	Stage  string  `json:"stage"`
	Count  int64   `json:"count"`
	Amount float64 `json:"amount"`
}

func normalizeTenantUUID(tenantUUID string) (string, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return "", ErrTenantUUIDMissing
	}
	return tenantUUID, nil
}
