package opportunity

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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

func (r *testOpportunityRepo) Forecast(ctx context.Context, tenantUUID string, _ opprepo.OpportunityListFilter) (*opprepo.OpportunityForecast, error) {
	var items []oppmodel.OpportunityRecord
	if err := r.WithContext(ctx).
		Where("tenant_uuid = ? AND stage IN ?", tenantUUID, []string{oppmodel.StageOpen, oppmodel.StageQualified, oppmodel.StageProposal, oppmodel.StageNegotiation}).
		Find(&items).Error; err != nil {
		return nil, err
	}
	out := &opprepo.OpportunityForecast{}
	stageBuckets := map[string]*opprepo.OpportunityForecastBucket{}
	ownerBuckets := map[string]*opprepo.OpportunityForecastBucket{}
	sourceBuckets := map[string]*opprepo.OpportunityForecastBucket{}
	for _, item := range items {
		amount := 0.0
		if item.Amount != nil {
			amount = *item.Amount
		}
		rate := item.Probability
		if rate <= 0 {
			rate = 20
		}
		weighted := amount * float64(rate) / 100
		out.TotalAmount += amount
		out.WeightedAmount += weighted
		addForecastBucket(stageBuckets, item.Stage, amount, weighted, rate)
		addForecastBucket(ownerBuckets, item.OwnerUserUUID, amount, weighted, rate)
		addForecastBucket(sourceBuckets, firstNonEmpty(item.SourceChannel, "unknown"), amount, weighted, rate)
	}
	for _, bucket := range stageBuckets {
		out.StageForecasts = append(out.StageForecasts, *bucket)
	}
	for _, bucket := range ownerBuckets {
		out.OwnerForecasts = append(out.OwnerForecasts, *bucket)
	}
	for _, bucket := range sourceBuckets {
		out.SourceForecasts = append(out.SourceForecasts, *bucket)
	}
	return out, nil
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

func TestServiceQuoteApprovalMakesEffectiveAndSyncsAmount(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	ctx := context.Background()
	lead := insertLead(t, db, leadmodel.LeadStatusConverted)
	item := createOpportunityForTest(t, svc, lead.LeadUUID)

	quote, err := svc.AddLineItem(ctx, testTenantUUID, item.OpportunityUUID, LineItemRequest{
		Kind:          oppmodel.LineItemKindQuoteFile,
		Name:          "v1.pdf",
		Quantity:      1,
		UnitPrice:     12800,
		TotalAmount:   12800,
		Currency:      "CNY",
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.Equal(t, 1, quote.VersionNo)
	require.Equal(t, oppmodel.QuoteApprovalDraft, quote.ApprovalStatus)

	quote, err = svc.UpdateQuoteApproval(ctx, testTenantUUID, item.OpportunityUUID, quote.ItemUUID, QuoteApprovalRequest{
		Action:        "submit",
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.Equal(t, oppmodel.QuoteApprovalSubmitted, quote.ApprovalStatus)
	require.NotNil(t, quote.SubmittedAt)

	quote, err = svc.UpdateQuoteApproval(ctx, testTenantUUID, item.OpportunityUUID, quote.ItemUUID, QuoteApprovalRequest{
		Action:        "approve",
		Comment:       "金额确认",
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.Equal(t, oppmodel.QuoteApprovalApproved, quote.ApprovalStatus)
	require.Equal(t, testActorUUID, quote.ApprovedBy)

	quote, err = svc.UpdateQuoteApproval(ctx, testTenantUUID, item.OpportunityUUID, quote.ItemUUID, QuoteApprovalRequest{
		Action:        "effective",
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.True(t, quote.IsEffective)
	require.Equal(t, oppmodel.QuoteApprovalEffective, quote.ApprovalStatus)

	updated, err := svc.Get(ctx, testTenantUUID, item.OpportunityUUID)
	require.NoError(t, err)
	require.NotNil(t, updated.Amount)
	require.Equal(t, 12800.0, *updated.Amount)
	require.Equal(t, "CNY", updated.Currency)
	requireActivityCount(t, db, item.OpportunityUUID, oppmodel.ActivityQuote, 3)
}

func TestServiceContractAndPaymentLifecycle(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	ctx := context.Background()
	lead := insertLead(t, db, leadmodel.LeadStatusConverted)
	item := createOpportunityForTest(t, svc, lead.LeadUUID)
	amount := 86000.0

	item, err := svc.Close(ctx, testTenantUUID, item.OpportunityUUID, CloseRequest{
		Result:        oppmodel.StageWon,
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.Equal(t, oppmodel.StageWon, item.Stage)

	contract, err := svc.AddContract(ctx, testTenantUUID, item.OpportunityUUID, ContractRequest{
		Title:         "年度服务合同",
		Amount:        &amount,
		Currency:      "CNY",
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, contract.ContractUUID)
	require.Equal(t, oppmodel.ContractStatusDraft, contract.Status)
	require.Equal(t, amount, contract.Amount)

	contract, err = svc.UpdateContractStatus(ctx, testTenantUUID, item.OpportunityUUID, contract.ContractUUID, ContractStatusRequest{
		Status:        oppmodel.ContractStatusSigned,
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.Equal(t, oppmodel.ContractStatusSigned, contract.Status)
	require.NotNil(t, contract.SignedAt)

	dueAt := time.Now().UTC().Add(24 * time.Hour)
	payment, err := svc.AddPayment(ctx, testTenantUUID, item.OpportunityUUID, PaymentRequest{
		ContractUUID:  contract.ContractUUID,
		Title:         "首付款",
		PlannedAmount: 43000,
		DueAt:         &dueAt,
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.Equal(t, oppmodel.PaymentStatusPlanned, payment.Status)
	require.Equal(t, 43000.0, payment.PlannedAmount)

	paidAmount := 43000.0
	payment, err = svc.UpdatePaymentStatus(ctx, testTenantUUID, item.OpportunityUUID, payment.PaymentUUID, PaymentStatusRequest{
		Status:        oppmodel.PaymentStatusPaid,
		PaidAmount:    &paidAmount,
		Method:        "bank_transfer",
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)
	require.Equal(t, oppmodel.PaymentStatusPaid, payment.Status)
	require.Equal(t, paidAmount, payment.PaidAmount)
	require.NotNil(t, payment.PaidAt)

	payments, summary, err := svc.ListPayments(ctx, testTenantUUID, item.OpportunityUUID)
	require.NoError(t, err)
	require.Len(t, payments, 1)
	require.Equal(t, amount, summary.ContractAmount)
	require.Equal(t, paidAmount, summary.PlannedAmount)
	require.Equal(t, paidAmount, summary.PaidAmount)
	require.Equal(t, 100.0, summary.CompletionRate)

	updated, err := svc.Get(ctx, testTenantUUID, item.OpportunityUUID)
	require.NoError(t, err)
	require.Equal(t, oppmodel.StageWon, updated.Stage)
	requireActivityCount(t, db, item.OpportunityUUID, oppmodel.ActivityContract, 2)
	requireActivityCount(t, db, item.OpportunityUUID, oppmodel.ActivityPayment, 2)
}

func TestServiceForecastAggregatesPipeline(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	ctx := context.Background()
	lead := insertLead(t, db, leadmodel.LeadStatusConverted)
	item := createOpportunityForTest(t, svc, lead.LeadUUID)
	amount := 100000.0
	probability := 60

	_, err := svc.Update(ctx, testTenantUUID, item.OpportunityUUID, UpdateRequest{
		Amount:        &amount,
		Probability:   &probability,
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)

	forecast, err := svc.Forecast(ctx, testTenantUUID, ListFilter{})
	require.NoError(t, err)
	require.Equal(t, amount, forecast.TotalAmount)
	require.Equal(t, 60000.0, forecast.WeightedAmount)
	require.NotEmpty(t, forecast.StageForecasts)
	require.NotEmpty(t, forecast.OwnerForecasts)
	require.NotEmpty(t, forecast.SourceForecasts)
}

func TestServiceStageConfigLifecycle(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	ctx := context.Background()

	items, err := svc.ListStageConfigs(ctx, testTenantUUID)
	require.NoError(t, err)
	require.Len(t, items, 6)
	require.Equal(t, oppmodel.StageOpen, items[0].StageKey)
	require.Empty(t, items[0].ConfigUUID)

	active := true
	item, err := svc.SaveStageConfig(ctx, testTenantUUID, StageConfigRequest{
		StageKey:        "demo",
		Label:           "演示阶段",
		SortOrder:       15,
		DefaultWinRate:  35,
		SLADays:         4,
		FixedStage:      oppmodel.StageQualified,
		IsActive:        &active,
		MigrationPolicy: "map_to_fixed",
		ActorUserUUID:   testActorUUID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, item.ConfigUUID)
	require.NotEmpty(t, item.PipelineGroupUUID)
	require.Equal(t, "demo", item.StageKey)
	require.Equal(t, 35, item.DefaultWinRate)

	_, err = svc.SaveStageConfig(ctx, testTenantUUID, StageConfigRequest{
		StageKey:       "bad",
		Label:          "错误阶段",
		DefaultWinRate: 101,
		FixedStage:     oppmodel.StageQualified,
		ActorUserUUID:  testActorUUID,
	})
	require.ErrorIs(t, err, ErrInvalidPayload)
}

func TestServiceDetectsAndMergesDuplicateOpportunity(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	ctx := context.Background()
	lead := insertLead(t, db, leadmodel.LeadStatusConverted)
	target := createOpportunityForTest(t, svc, lead.LeadUUID)

	duplicate := &oppmodel.OpportunityRecord{
		OpportunityUUID: uuid.NewString(),
		TenantUUID:      testTenantUUID,
		LeadUUID:        lead.LeadUUID,
		Title:           "测试商机重复",
		Stage:           oppmodel.StageOpen,
		Currency:        "CNY",
		Probability:     20,
		OwnerUserUUID:   testOwnerUUID,
		RiskFlags:       []byte("[]"),
		CreatedBy:       testActorUUID,
		UpdatedBy:       testActorUUID,
	}
	require.NoError(t, db.Create(duplicate).Error)
	quote, err := svc.AddLineItem(ctx, testTenantUUID, duplicate.OpportunityUUID, LineItemRequest{
		Name:          "重复报价.pdf",
		Quantity:      1,
		UnitPrice:     1000,
		TotalAmount:   1000,
		Currency:      "CNY",
		ActorUserUUID: testActorUUID,
	})
	require.NoError(t, err)

	candidates, err := svc.DetectDuplicates(ctx, testTenantUUID, target.OpportunityUUID)
	require.NoError(t, err)
	require.NotEmpty(t, candidates)
	require.Equal(t, duplicate.OpportunityUUID, candidates[0].OpportunityUUID)

	_, err = svc.MergeOpportunity(ctx, testTenantUUID, target.OpportunityUUID, MergeRequest{
		SourceOpportunityUUID: duplicate.OpportunityUUID,
		Reason:                "测试重复合并",
		ActorUserUUID:         testActorUUID,
	})
	require.NoError(t, err)

	var moved oppmodel.OpportunityLineItem
	require.NoError(t, db.Where("item_uuid = ?", quote.ItemUUID).First(&moved).Error)
	require.Equal(t, target.OpportunityUUID, moved.OpportunityUUID)

	var source oppmodel.OpportunityRecord
	require.NoError(t, db.Where("opportunity_uuid = ?", duplicate.OpportunityUUID).First(&source).Error)
	require.Equal(t, oppmodel.StageLost, source.Stage)
	require.Contains(t, source.LostReason, "测试重复合并")
	requireActivityCount(t, db, target.OpportunityUUID, oppmodel.ActivityMerge, 1)
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

func TestServiceListPipelineGroupsDedupeByGroupKey(t *testing.T) {
	db := newOpportunityTestDB(t)
	svc := newOpportunityTestService(db)
	now := time.Now().UTC()
	olderUUID := uuid.NewString()
	newerUUID := uuid.NewString()

	require.NoError(t, db.Create(&oppmodel.OpportunityPipelineGroup{
		GroupUUID:   olderUUID,
		TenantUUID:  testTenantUUID,
		GroupKey:    oppmodel.DefaultPipelineGroupKey,
		Name:        "默认销售流程",
		Description: "历史重复数据",
		IsDefault:   true,
		IsActive:    true,
		SortOrder:   10,
		CreatedBy:   testActorUUID,
		UpdatedBy:   testActorUUID,
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now.Add(-time.Hour),
	}).Error)
	require.NoError(t, db.Create(&oppmodel.OpportunityPipelineGroup{
		GroupUUID:   newerUUID,
		TenantUUID:  testTenantUUID,
		GroupKey:    oppmodel.DefaultPipelineGroupKey,
		Name:        "默认销售流程",
		Description: "阶段更完整的数据",
		IsDefault:   true,
		IsActive:    true,
		SortOrder:   20,
		CreatedBy:   testActorUUID,
		UpdatedBy:   testActorUUID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}).Error)
	require.NoError(t, db.Create(&oppmodel.OpportunityStageConfig{
		ConfigUUID:        uuid.NewString(),
		TenantUUID:        testTenantUUID,
		PipelineGroupUUID: newerUUID,
		StageKey:          oppmodel.StageOpen,
		Label:             "打开",
		SortOrder:         10,
		DefaultWinRate:    20,
		SLADays:           3,
		StageType:         oppmodel.StageTypeActive,
		FixedStage:        oppmodel.StageOpen,
		IsActive:          true,
		MigrationPolicy:   "map_to_fixed",
		CreatedBy:         testActorUUID,
		UpdatedBy:         testActorUUID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error)

	list, err := svc.ListPipelineGroups(context.Background(), testTenantUUID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, newerUUID, list[0].GroupUUID)
	require.Equal(t, oppmodel.DefaultPipelineGroupKey, list[0].GroupKey)
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
			pipeline_group_uuid text,
			current_stage_uuid text,
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
			version_no integer NOT NULL DEFAULT 1,
			approval_status text NOT NULL DEFAULT 'draft',
			is_effective boolean NOT NULL DEFAULT false,
			submitted_at datetime,
			approved_at datetime,
			rejected_at datetime,
			effective_at datetime,
			approval_comment text,
			approved_by text,
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
		`CREATE TABLE opportunity_contracts (
			contract_uuid text PRIMARY KEY,
			tenant_uuid text NOT NULL,
			opportunity_uuid text NOT NULL,
			customer_uuid text,
			quote_item_uuid text,
			contract_no text NOT NULL,
			title text NOT NULL,
			amount numeric NOT NULL,
			currency text NOT NULL,
			status text NOT NULL,
			signed_at datetime,
			storage_provider text,
			object_key text,
			file_name text,
			file_size integer NOT NULL DEFAULT 0,
			content_type text,
			created_by text NOT NULL,
			updated_by text NOT NULL,
			created_at datetime,
			updated_at datetime
		)`,
		`CREATE TABLE opportunity_payments (
			payment_uuid text PRIMARY KEY,
			tenant_uuid text NOT NULL,
			opportunity_uuid text NOT NULL,
			contract_uuid text NOT NULL,
			title text NOT NULL,
			planned_amount numeric NOT NULL,
			paid_amount numeric NOT NULL DEFAULT 0,
			currency text NOT NULL,
			status text NOT NULL,
			due_at datetime,
			paid_at datetime,
			method text,
			transaction_no text,
			note text,
			created_by text NOT NULL,
			updated_by text NOT NULL,
			created_at datetime,
			updated_at datetime
		)`,
		`CREATE TABLE opportunity_pipeline_groups (
			group_uuid text PRIMARY KEY,
			tenant_uuid text NOT NULL,
			group_key text NOT NULL,
			name text NOT NULL,
			description text,
			is_default boolean NOT NULL DEFAULT false,
			is_active boolean NOT NULL DEFAULT true,
			sort_order integer NOT NULL DEFAULT 0,
			created_by text NOT NULL,
			updated_by text NOT NULL,
			created_at datetime,
			updated_at datetime
		)`,
		`CREATE TABLE opportunity_stage_configs (
			config_uuid text PRIMARY KEY,
			tenant_uuid text NOT NULL,
			pipeline_group_uuid text,
			stage_key text NOT NULL,
			label text NOT NULL,
			sort_order integer NOT NULL DEFAULT 0,
			default_win_rate integer NOT NULL DEFAULT 0,
			sla_days integer NOT NULL DEFAULT 0,
			stage_type text NOT NULL DEFAULT 'active',
			fixed_stage text NOT NULL,
			is_active boolean NOT NULL DEFAULT true,
			migration_policy text NOT NULL DEFAULT 'map_to_fixed',
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

func addForecastBucket(buckets map[string]*opprepo.OpportunityForecastBucket, key string, amount float64, weighted float64, rate int) {
	key = firstNonEmpty(key, "unknown")
	bucket := buckets[key]
	if bucket == nil {
		bucket = &opprepo.OpportunityForecastBucket{Key: key, Label: key}
		buckets[key] = bucket
	}
	bucket.Count++
	bucket.Amount += amount
	bucket.WeightedAmount += weighted
	bucket.AverageRate = ((bucket.AverageRate * float64(bucket.Count-1)) + float64(rate)) / float64(bucket.Count)
}
