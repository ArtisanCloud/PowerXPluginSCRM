package opportunity

import (
	"context"
	"fmt"
	"strings"
	"testing"

	oppmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/opportunity"
	opprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/opportunity"
	entitymodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	testTenantUUID = "11111111-1111-1111-1111-111111111111"
	testActorUUID  = "22222222-2222-2222-2222-222222222222"
	testOwnerUUID  = "33333333-3333-3333-3333-333333333333"
)

type testOpportunityRepo struct {
	*gorm.DB
}

func (r *testOpportunityRepo) Create(ctx context.Context, item *oppmodel.OpportunityRecord) error {
	return r.WithContext(ctx).Create(item).Error
}

func (r *testOpportunityRepo) GetByUUID(ctx context.Context, tenantUUID, opportunityUUID string) (*oppmodel.OpportunityRecord, error) {
	var out oppmodel.OpportunityRecord
	err := r.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *testOpportunityRepo) List(ctx context.Context, tenantUUID string, filter opprepo.OpportunityListFilter) ([]*oppmodel.OpportunityRecord, error) {
	query := r.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID)
	if filter.Stage != "" {
		query = query.Where("stage = ?", filter.Stage)
	}
	var out []*oppmodel.OpportunityRecord
	return out, query.Order("created_at DESC").Find(&out).Error
}

func (r *testOpportunityRepo) Dashboard(ctx context.Context, tenantUUID string, _ opprepo.OpportunityListFilter) (*opprepo.OpportunityDashboard, error) {
	var activeCount int64
	if err := r.WithContext(ctx).Model(&oppmodel.OpportunityRecord{}).
		Where("tenant_uuid = ? AND stage IN ?", tenantUUID, []string{oppmodel.StageOpen, oppmodel.StageQualified, oppmodel.StageProposal, oppmodel.StageNegotiation}).
		Count(&activeCount).Error; err != nil {
		return nil, err
	}
	return &opprepo.OpportunityDashboard{ActiveCount: activeCount}, nil
}

func (r *testOpportunityRepo) FindActiveByLead(ctx context.Context, tenantUUID, leadUUID string) (*oppmodel.OpportunityRecord, error) {
	var out oppmodel.OpportunityRecord
	err := r.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ? AND stage IN ?", tenantUUID, leadUUID, []string{oppmodel.StageOpen, oppmodel.StageQualified, oppmodel.StageProposal, oppmodel.StageNegotiation}).
		First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *testOpportunityRepo) Update(ctx context.Context, item *oppmodel.OpportunityRecord) error {
	return r.WithContext(ctx).Save(item).Error
}

func (r *testOpportunityRepo) BeginTenantTx(ctx context.Context, _ string, fn func(tx *gorm.DB) error) error {
	return r.WithContext(ctx).Transaction(fn)
}

func TestServiceCreateRequiresQualifiedLeadAndDetectsConflict(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	ctx := context.Background()
	lead := insertLead(t, db, leadmodel.LeadStatusConverted)

	item, err := svc.Create(ctx, testTenantUUID, CreateRequest{
		LeadUUID:      lead.LeadUUID,
		Title:         "年度采购商机",
		OwnerUserUUID: testOwnerUUID,
		Currency:      "",
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.Equal(t, oppmodel.StageOpen, item.Stage)
	require.Equal(t, "CNY", item.Currency)
	require.Equal(t, lead.SourceChannel, item.SourceChannel)
	require.Equal(t, lead.SourceAppType, item.SourceAppType)

	var activities []oppmodel.OpportunityActivity
	require.NoError(t, db.Where("opportunity_uuid = ?", item.OpportunityUUID).Find(&activities).Error)
	require.Len(t, activities, 1)
	require.Equal(t, oppmodel.ActivityCreate, activities[0].ActivityType)

	_, err = svc.Create(ctx, testTenantUUID, CreateRequest{
		LeadUUID:      lead.LeadUUID,
		Title:         "重复商机",
		OwnerUserUUID: testOwnerUUID,
		ActorUserUUID: testActorUUID,
	})
	var conflict *ConflictError
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, item.OpportunityUUID, conflict.OpportunityUUID)

	unqualified := insertLead(t, db, leadmodel.LeadStatusAssigned)
	_, err = svc.Create(ctx, testTenantUUID, CreateRequest{
		LeadUUID:      unqualified.LeadUUID,
		Title:         "未合格商机",
		OwnerUserUUID: testOwnerUUID,
		ActorUserUUID: testActorUUID,
	})
	require.ErrorIs(t, err, ErrLeadMustBeQualified)
}

func TestServiceStageCloseReopenAndRiskAreAudited(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	ctx := context.Background()
	lead := insertLead(t, db, leadmodel.LeadStatusConverted)
	item := createOpportunityForTest(t, svc, lead.LeadUUID)

	advanced, err := svc.AdvanceStage(ctx, testTenantUUID, item.OpportunityUUID, StageRequest{
		Stage:         oppmodel.StageProposal,
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.Equal(t, oppmodel.StageProposal, advanced.Stage)

	_, err = svc.AdvanceStage(ctx, testTenantUUID, item.OpportunityUUID, StageRequest{
		Stage:         oppmodel.StageProposal,
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	requireActivityCount(t, db, item.OpportunityUUID, oppmodel.ActivityStageChange, 1)

	closed, err := svc.Close(ctx, testTenantUUID, item.OpportunityUUID, CloseRequest{
		Result:        oppmodel.StageLost,
		LostReason:    "预算取消",
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.Equal(t, oppmodel.StageLost, closed.Stage)
	require.NotNil(t, closed.LostAt)
	requireActivityCount(t, db, item.OpportunityUUID, oppmodel.ActivityClose, 1)

	var updatedLead leadmodel.Lead
	require.NoError(t, db.Where("lead_uuid = ?", lead.LeadUUID).First(&updatedLead).Error)
	require.Equal(t, leadmodel.LeadStatusClosed, updatedLead.Status)

	reopened, err := svc.Reopen(ctx, testTenantUUID, item.OpportunityUUID, ReopenRequest{ActorUserUUID: testActorUUID})
	require.NoError(t, err)
	require.Equal(t, oppmodel.StageOpen, reopened.Stage)
	require.Nil(t, reopened.LostAt)
	require.Empty(t, reopened.LostReason)
	requireActivityCount(t, db, item.OpportunityUUID, oppmodel.ActivityReopen, 1)

	risked, err := svc.MarkRisk(ctx, testTenantUUID, item.OpportunityUUID, RiskRequest{ActorUserUUID: testActorUUID})
	require.NoError(t, err)
	require.Contains(t, string(risked.RiskFlags), "disconnected")
	requireActivityCount(t, db, item.OpportunityUUID, oppmodel.ActivityRiskFlag, 1)
}

func TestServiceCloseAndReopenAreIdempotent(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	ctx := context.Background()
	lead := insertLead(t, db, leadmodel.LeadStatusConverted)
	item := createOpportunityForTest(t, svc, lead.LeadUUID)

	_, err := svc.Close(ctx, testTenantUUID, item.OpportunityUUID, CloseRequest{
		Result:        oppmodel.StageWon,
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	_, err = svc.Close(ctx, testTenantUUID, item.OpportunityUUID, CloseRequest{
		Result:        oppmodel.StageWon,
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	requireActivityCount(t, db, item.OpportunityUUID, oppmodel.ActivityClose, 1)

	_, err = svc.Reopen(ctx, testTenantUUID, item.OpportunityUUID, ReopenRequest{ActorUserUUID: testActorUUID})
	require.NoError(t, err)
	_, err = svc.Reopen(ctx, testTenantUUID, item.OpportunityUUID, ReopenRequest{ActorUserUUID: testActorUUID})
	require.NoError(t, err)
	requireActivityCount(t, db, item.OpportunityUUID, oppmodel.ActivityReopen, 1)
}

func TestServiceRejectsInvalidActorUUID(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	lead := insertLead(t, db, leadmodel.LeadStatusConverted)

	_, err := svc.Create(context.Background(), testTenantUUID, CreateRequest{
		LeadUUID:      lead.LeadUUID,
		Title:         "无操作人商机",
		OwnerUserUUID: testOwnerUUID,
		ActorUserUUID: "system",
	})
	require.ErrorIs(t, err, ErrActorUserUUIDRequired)
}

func TestServiceTenantIsolation(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	ctx := context.Background()
	lead := insertLead(t, db, leadmodel.LeadStatusConverted)
	item := createOpportunityForTest(t, svc, lead.LeadUUID)
	otherTenantUUID := "44444444-4444-4444-4444-444444444444"

	_, err := svc.Get(ctx, otherTenantUUID, item.OpportunityUUID)
	require.ErrorIs(t, err, ErrOpportunityNotFound)

	items, err := svc.List(ctx, otherTenantUUID, ListFilter{})
	require.NoError(t, err)
	require.Empty(t, items)

	_, err = svc.AdvanceStage(ctx, otherTenantUUID, item.OpportunityUUID, StageRequest{
		Stage:         oppmodel.StageProposal,
		ActorUserUUID: testActorUUID,
	})
	require.ErrorIs(t, err, ErrOpportunityNotFound)

	var stored oppmodel.OpportunityRecord
	require.NoError(t, db.Where("tenant_uuid = ? AND opportunity_uuid = ?", testTenantUUID, item.OpportunityUUID).First(&stored).Error)
	require.Equal(t, oppmodel.StageOpen, stored.Stage)
}

func newOpportunityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	entitymodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	for _, stmt := range []string{
		`CREATE TABLE lead_capture_leads (
			lead_uuid text PRIMARY KEY,
			tenant_uuid text NOT NULL,
			display_name text,
			phone text,
			email text,
			status text NOT NULL,
			owner_user_uuid text,
			source_channel text,
			source_app_type text,
			source_account_uuid text,
			created_at datetime,
			updated_at datetime
		)`,
		`CREATE TABLE customer_accounts (
			id integer PRIMARY KEY AUTOINCREMENT,
			tenant_uuid text NOT NULL,
			customer_uuid text NOT NULL,
			email text,
			phone text,
			password_hash text,
			status text NOT NULL,
			metadata text,
			email_verified boolean,
			phone_verified boolean,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`CREATE TABLE opportunity_records (
			opportunity_uuid text PRIMARY KEY,
			tenant_uuid text NOT NULL,
			lead_uuid text NOT NULL,
			title text NOT NULL,
			stage text NOT NULL,
			amount numeric,
			currency text NOT NULL,
			probability integer NOT NULL,
			owner_user_uuid text NOT NULL,
			source_channel text,
			source_app_type text,
			source_account_uuid text,
			external_userid text,
			expected_close_at datetime,
			won_at datetime,
			lost_at datetime,
			lost_reason text,
			risk_flags text NOT NULL,
			created_by text NOT NULL,
			updated_by text NOT NULL,
			created_at datetime,
			updated_at datetime
		)`,
		`CREATE TABLE opportunity_activities (
			activity_uuid text PRIMARY KEY,
			tenant_uuid text NOT NULL,
			opportunity_uuid text NOT NULL,
			activity_type text NOT NULL,
			from_stage text,
			to_stage text,
			payload text NOT NULL,
			operator_user_uuid text NOT NULL,
			request_id text,
			created_at datetime
		)`,
		`CREATE TABLE opportunity_line_items (
			item_uuid text PRIMARY KEY,
			tenant_uuid text NOT NULL,
			opportunity_uuid text NOT NULL,
			kind text NOT NULL,
			name text NOT NULL,
			quantity numeric NOT NULL,
			unit_price numeric NOT NULL,
			total_amount numeric NOT NULL,
			currency text NOT NULL,
			storage_provider text,
			object_key text,
			file_name text,
			file_size integer NOT NULL,
			content_type text,
			created_by text NOT NULL,
			updated_by text NOT NULL,
			created_at datetime,
			updated_at datetime
		)`,
		`CREATE TABLE opportunity_tasks (
			task_uuid text PRIMARY KEY,
			tenant_uuid text NOT NULL,
			opportunity_uuid text NOT NULL,
			title text NOT NULL,
			due_at datetime,
			status text NOT NULL,
			created_by text NOT NULL,
			updated_by text NOT NULL,
			created_at datetime,
			updated_at datetime
		)`,
	} {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

func newOpportunityTestService(db *gorm.DB) *Service {
	repo := &testOpportunityRepo{DB: db}
	return &Service{
		repo:      repo,
		activity:  opprepo.NewOpportunityActivityRepository(db),
		lineItems: opprepo.NewOpportunityLineItemRepository(db),
		tasks:     opprepo.NewOpportunityTaskRepository(db),
		db:        db,
	}
}

func insertLead(t *testing.T, db *gorm.DB, status string) *leadmodel.Lead {
	t.Helper()
	sourceAccount := uuid.NewString()
	lead := &leadmodel.Lead{
		LeadUUID:          uuid.NewString(),
		TenantUUID:        testTenantUUID,
		DisplayName:       fmt.Sprintf("测试线索 %s", status),
		Phone:             "13800000000",
		Email:             strings.ReplaceAll(status, "_", "-") + "@example.test",
		Status:            status,
		OwnerUserUUID:     testOwnerUUID,
		SourceChannel:     "wechat",
		SourceAppType:     "openwork",
		SourceAccountUUID: &sourceAccount,
	}
	require.NoError(t, db.Create(lead).Error)
	return lead
}

func createOpportunityForTest(t *testing.T, svc *Service, leadUUID string) *oppmodel.OpportunityRecord {
	t.Helper()
	item, err := svc.Create(context.Background(), testTenantUUID, CreateRequest{
		LeadUUID:      leadUUID,
		Title:         "测试商机",
		OwnerUserUUID: testOwnerUUID,
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	return item
}

func requireActivityCount(t *testing.T, db *gorm.DB, opportunityUUID, activityType string, expected int64) {
	t.Helper()
	var count int64
	err := db.Model(&oppmodel.OpportunityActivity{}).
		Where("opportunity_uuid = ? AND activity_type = ?", opportunityUUID, activityType).
		Count(&count).Error
	require.NoError(t, err)
	require.Equal(t, expected, count)
}

var _ opprepo.OpportunityRepository = (*testOpportunityRepo)(nil)
