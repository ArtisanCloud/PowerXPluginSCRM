package opportunity

import (
	"context"
	"fmt"
	"strings"
	"time"

	oppmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/opportunity"
	"gorm.io/gorm"
)

type opportunityRepository struct{ gormStore }
type opportunityActivityRepository struct{ gormStore }
type opportunityLineItemRepository struct{ gormStore }
type opportunityTaskRepository struct{ gormStore }

func NewOpportunityRepository(db *gorm.DB) OpportunityRepository {
	return &opportunityRepository{gormStore{db: db}}
}

func NewOpportunityActivityRepository(db *gorm.DB) OpportunityActivityRepository {
	return &opportunityActivityRepository{gormStore{db: db}}
}

func NewOpportunityLineItemRepository(db *gorm.DB) OpportunityLineItemRepository {
	return &opportunityLineItemRepository{gormStore{db: db}}
}

func NewOpportunityTaskRepository(db *gorm.DB) OpportunityTaskRepository {
	return &opportunityTaskRepository{gormStore{db: db}}
}

func (r *opportunityRepository) Create(ctx context.Context, item *oppmodel.OpportunityRecord) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	if item == nil {
		return fmt.Errorf("opportunity record is required")
	}
	tenantUUID, err := normalizeTenantUUID(item.TenantUUID)
	if err != nil {
		return err
	}
	item.TenantUUID = tenantUUID
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *opportunityRepository) GetByUUID(ctx context.Context, tenantUUID, opportunityUUID string) (*oppmodel.OpportunityRecord, error) {
	if r == nil || r.db == nil {
		return nil, ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return nil, err
	}
	opportunityUUID = strings.ToLower(strings.TrimSpace(opportunityUUID))
	if opportunityUUID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var out oppmodel.OpportunityRecord
	if err := r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).
		First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *opportunityRepository) List(ctx context.Context, tenantUUID string, filter OpportunityListFilter) ([]*oppmodel.OpportunityRecord, error) {
	if r == nil || r.db == nil {
		return nil, ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return nil, err
	}
	query := r.applyListFilter(r.db.WithContext(ctx).Model(&oppmodel.OpportunityRecord{}), tenantUUID, filter)
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var out []*oppmodel.OpportunityRecord
	err = query.Order("created_at DESC").Limit(limit).Find(&out).Error
	return out, err
}

func (r *opportunityRepository) Dashboard(ctx context.Context, tenantUUID string, filter OpportunityListFilter) (*OpportunityDashboard, error) {
	if r == nil || r.db == nil {
		return nil, ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return nil, err
	}
	base := r.applyListFilter(r.db.WithContext(ctx).Model(&oppmodel.OpportunityRecord{}), tenantUUID, filter)
	active := []string{oppmodel.StageOpen, oppmodel.StageQualified, oppmodel.StageProposal, oppmodel.StageNegotiation}
	out := &OpportunityDashboard{StageSummaries: make([]OpportunityStageSummary, 0, len(allStages()))}
	if err := base.Session(&gorm.Session{}).Where("stage IN ?", active).Count(&out.ActiveCount).Error; err != nil {
		return nil, err
	}
	if err := base.Session(&gorm.Session{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("stage IN ?", active).
		Scan(&out.PipelineAmount).Error; err != nil {
		return nil, err
	}
	if err := base.Session(&gorm.Session{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("stage = ?", oppmodel.StageWon).
		Scan(&out.WonAmount).Error; err != nil {
		return nil, err
	}
	if err := base.Session(&gorm.Session{}).
		Where("jsonb_array_length(COALESCE(risk_flags, '[]'::jsonb)) > 0").
		Count(&out.RiskCount).Error; err != nil {
		return nil, err
	}
	type stageRow struct {
		Stage  string
		Count  int64
		Amount float64
	}
	var rows []stageRow
	if err := base.Session(&gorm.Session{}).
		Select("stage, COUNT(*) AS count, COALESCE(SUM(amount), 0) AS amount").
		Group("stage").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	byStage := map[string]OpportunityStageSummary{}
	for _, row := range rows {
		stage := strings.ToLower(strings.TrimSpace(row.Stage))
		byStage[stage] = OpportunityStageSummary{Stage: stage, Count: row.Count, Amount: row.Amount}
	}
	for _, stage := range allStages() {
		summary := byStage[stage]
		if summary.Stage == "" {
			summary.Stage = stage
		}
		out.StageSummaries = append(out.StageSummaries, summary)
	}
	return out, nil
}

func (r *opportunityRepository) applyListFilter(query *gorm.DB, tenantUUID string, filter OpportunityListFilter) *gorm.DB {
	query = query.Where("tenant_uuid = ?", tenantUUID)
	if stage := strings.ToLower(strings.TrimSpace(filter.Stage)); stage != "" {
		query = query.Where("stage = ?", stage)
	}
	if owner := strings.TrimSpace(filter.OwnerUserUUID); owner != "" {
		query = query.Where("owner_user_uuid = ?", owner)
	}
	if leadUUID := strings.ToLower(strings.TrimSpace(filter.LeadUUID)); leadUUID != "" {
		query = query.Where("lead_uuid = ?", leadUUID)
	}
	if source := strings.ToLower(strings.TrimSpace(filter.SourceChannel)); source != "" {
		query = query.Where("LOWER(source_channel) = ?", source)
	}
	if filter.RiskOnly {
		query = query.Where("jsonb_array_length(COALESCE(risk_flags, '[]'::jsonb)) > 0")
	}
	if filter.ExpectedCloseFrom != nil {
		query = query.Where("expected_close_at >= ?", *filter.ExpectedCloseFrom)
	}
	if filter.ExpectedCloseTo != nil {
		query = query.Where("expected_close_at <= ?", *filter.ExpectedCloseTo)
	}
	if keyword := strings.ToLower(strings.TrimSpace(filter.Keyword)); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Joins("LEFT JOIN lead_capture_leads ON lead_capture_leads.tenant_uuid = opportunity_records.tenant_uuid AND lead_capture_leads.lead_uuid = opportunity_records.lead_uuid").
			Where(
				`LOWER(opportunity_records.title) LIKE ? OR LOWER(opportunity_records.lead_uuid::text) LIKE ? OR LOWER(opportunity_records.owner_user_uuid::text) LIKE ? OR LOWER(COALESCE(opportunity_records.source_channel, '')) LIKE ? OR LOWER(COALESCE(lead_capture_leads.display_name, '')) LIKE ? OR LOWER(COALESCE(lead_capture_leads.phone, '')) LIKE ? OR LOWER(COALESCE(lead_capture_leads.email, '')) LIKE ?`,
				like,
				like,
				like,
				like,
				like,
				like,
				like,
			)
	}
	return query
}

func (r *opportunityRepository) FindActiveByLead(ctx context.Context, tenantUUID, leadUUID string) (*oppmodel.OpportunityRecord, error) {
	if r == nil || r.db == nil {
		return nil, ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return nil, err
	}
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if leadUUID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var out oppmodel.OpportunityRecord
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ? AND stage IN ?", tenantUUID, leadUUID, activeStages()).
		Order("created_at DESC").
		First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *opportunityRepository) Update(ctx context.Context, item *oppmodel.OpportunityRecord) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	if item == nil {
		return fmt.Errorf("opportunity record is required")
	}
	tenantUUID, err := normalizeTenantUUID(item.TenantUUID)
	if err != nil {
		return err
	}
	opportunityUUID := strings.ToLower(strings.TrimSpace(item.OpportunityUUID))
	if opportunityUUID == "" {
		return gorm.ErrRecordNotFound
	}
	item.UpdatedAt = time.Now().UTC()
	q := r.db.WithContext(ctx).
		Model(&oppmodel.OpportunityRecord{}).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).
		Updates(map[string]any{
			"title":             item.Title,
			"stage":             item.Stage,
			"amount":            item.Amount,
			"currency":          item.Currency,
			"probability":       item.Probability,
			"owner_user_uuid":   item.OwnerUserUUID,
			"expected_close_at": item.ExpectedCloseAt,
			"won_at":            item.WonAt,
			"lost_at":           item.LostAt,
			"lost_reason":       item.LostReason,
			"risk_flags":        item.RiskFlags,
			"updated_by":        item.UpdatedBy,
			"updated_at":        item.UpdatedAt,
		})
	return q.Error
}

func (r *opportunityRepository) BeginTenantTx(ctx context.Context, tenantUUID string, fn func(tx *gorm.DB) error) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL app.tenant_uuid = ?", tenantUUID).Error; err != nil {
			return err
		}
		return fn(tx)
	})
}

func (r *opportunityActivityRepository) Create(ctx context.Context, item *oppmodel.OpportunityActivity) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	if item == nil {
		return fmt.Errorf("opportunity activity is required")
	}
	tenantUUID, err := normalizeTenantUUID(item.TenantUUID)
	if err != nil {
		return err
	}
	item.TenantUUID = tenantUUID
	item.CreatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *opportunityActivityRepository) ListByOpportunity(ctx context.Context, tenantUUID, opportunityUUID string, limit int) ([]*oppmodel.OpportunityActivity, error) {
	if r == nil || r.db == nil {
		return nil, ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return nil, err
	}
	opportunityUUID = strings.ToLower(strings.TrimSpace(opportunityUUID))
	if opportunityUUID == "" {
		return []*oppmodel.OpportunityActivity{}, nil
	}
	if limit <= 0 {
		limit = 100
	}
	var out []*oppmodel.OpportunityActivity
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error
	return out, err
}

func (r *opportunityLineItemRepository) ListByOpportunity(ctx context.Context, tenantUUID, opportunityUUID string) ([]*oppmodel.OpportunityLineItem, error) {
	if r == nil || r.db == nil {
		return nil, ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return nil, err
	}
	var out []*oppmodel.OpportunityLineItem
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, strings.ToLower(strings.TrimSpace(opportunityUUID))).
		Order("created_at DESC").
		Find(&out).Error
	return out, err
}

func (r *opportunityLineItemRepository) Create(ctx context.Context, item *oppmodel.OpportunityLineItem) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	if item == nil {
		return fmt.Errorf("opportunity line item is required")
	}
	tenantUUID, err := normalizeTenantUUID(item.TenantUUID)
	if err != nil {
		return err
	}
	item.TenantUUID = tenantUUID
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *opportunityLineItemRepository) Delete(ctx context.Context, tenantUUID, opportunityUUID, itemUUID string) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ? AND item_uuid = ?", tenantUUID, strings.ToLower(strings.TrimSpace(opportunityUUID)), strings.ToLower(strings.TrimSpace(itemUUID))).
		Delete(&oppmodel.OpportunityLineItem{}).Error
}

func (r *opportunityTaskRepository) ListByOpportunity(ctx context.Context, tenantUUID, opportunityUUID string) ([]*oppmodel.OpportunityTask, error) {
	if r == nil || r.db == nil {
		return nil, ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return nil, err
	}
	var out []*oppmodel.OpportunityTask
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, strings.ToLower(strings.TrimSpace(opportunityUUID))).
		Order("status ASC, due_at ASC NULLS LAST, created_at DESC").
		Find(&out).Error
	return out, err
}

func (r *opportunityTaskRepository) Create(ctx context.Context, item *oppmodel.OpportunityTask) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	if item == nil {
		return fmt.Errorf("opportunity task is required")
	}
	tenantUUID, err := normalizeTenantUUID(item.TenantUUID)
	if err != nil {
		return err
	}
	item.TenantUUID = tenantUUID
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *opportunityTaskRepository) Update(ctx context.Context, item *oppmodel.OpportunityTask) error {
	if r == nil || r.db == nil {
		return ErrDBNotReady
	}
	if item == nil {
		return fmt.Errorf("opportunity task is required")
	}
	item.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&oppmodel.OpportunityTask{}).
		Where("tenant_uuid = ? AND opportunity_uuid = ? AND task_uuid = ?", item.TenantUUID, item.OpportunityUUID, item.TaskUUID).
		Updates(map[string]any{
			"title":      item.Title,
			"due_at":     item.DueAt,
			"status":     item.Status,
			"updated_by": item.UpdatedBy,
			"updated_at": item.UpdatedAt,
		}).Error
}

func (r *opportunityTaskRepository) GetByUUID(ctx context.Context, tenantUUID, opportunityUUID, taskUUID string) (*oppmodel.OpportunityTask, error) {
	if r == nil || r.db == nil {
		return nil, ErrDBNotReady
	}
	tenantUUID, err := normalizeTenantUUID(tenantUUID)
	if err != nil {
		return nil, err
	}
	var out oppmodel.OpportunityTask
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ? AND task_uuid = ?", tenantUUID, strings.ToLower(strings.TrimSpace(opportunityUUID)), strings.ToLower(strings.TrimSpace(taskUUID))).
		First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func activeStages() []string {
	return []string{
		oppmodel.StageOpen,
		oppmodel.StageQualified,
		oppmodel.StageProposal,
		oppmodel.StageNegotiation,
	}
}

func allStages() []string {
	return []string{
		oppmodel.StageOpen,
		oppmodel.StageQualified,
		oppmodel.StageProposal,
		oppmodel.StageNegotiation,
		oppmodel.StageWon,
		oppmodel.StageLost,
	}
}
