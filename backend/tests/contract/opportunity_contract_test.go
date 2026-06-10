package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	oppmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/opportunity"
	opprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/opportunity"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	oppsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/opportunity"
	httpopp "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/opportunity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	opportunityContractTenant = "00000000-0000-0000-0000-000000000071"
	opportunityContractActor  = "00000000-0000-0000-0000-000000000072"
	opportunityContractOwner  = "00000000-0000-0000-0000-000000000073"
)

type opportunityContractRepo struct {
	*gorm.DB
}

func (r *opportunityContractRepo) Create(ctx context.Context, item *oppmodel.OpportunityRecord) error {
	return r.WithContext(ctx).Create(item).Error
}

func (r *opportunityContractRepo) GetByUUID(ctx context.Context, tenantUUID, opportunityUUID string) (*oppmodel.OpportunityRecord, error) {
	var out oppmodel.OpportunityRecord
	err := r.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *opportunityContractRepo) List(ctx context.Context, tenantUUID string, filter opprepo.OpportunityListFilter) ([]*oppmodel.OpportunityRecord, error) {
	query := r.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID)
	if strings.TrimSpace(filter.Stage) != "" {
		query = query.Where("stage = ?", strings.TrimSpace(filter.Stage))
	}
	if strings.TrimSpace(filter.OwnerUserUUID) != "" {
		query = query.Where("owner_user_uuid = ?", strings.TrimSpace(filter.OwnerUserUUID))
	}
	if strings.TrimSpace(filter.LeadUUID) != "" {
		query = query.Where("lead_uuid = ?", strings.TrimSpace(filter.LeadUUID))
	}
	var out []*oppmodel.OpportunityRecord
	return out, query.Order("created_at DESC").Find(&out).Error
}

func (r *opportunityContractRepo) Dashboard(ctx context.Context, tenantUUID string, _ opprepo.OpportunityListFilter) (*opprepo.OpportunityDashboard, error) {
	var activeCount int64
	if err := r.WithContext(ctx).Model(&oppmodel.OpportunityRecord{}).
		Where("tenant_uuid = ? AND stage IN ?", tenantUUID, []string{oppmodel.StageOpen, oppmodel.StageQualified, oppmodel.StageProposal, oppmodel.StageNegotiation}).
		Count(&activeCount).Error; err != nil {
		return nil, err
	}
	return &opprepo.OpportunityDashboard{ActiveCount: activeCount}, nil
}

func (r *opportunityContractRepo) FindActiveByLead(ctx context.Context, tenantUUID, leadUUID string) (*oppmodel.OpportunityRecord, error) {
	var out oppmodel.OpportunityRecord
	err := r.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ? AND stage IN ?", tenantUUID, leadUUID, []string{oppmodel.StageOpen, oppmodel.StageQualified, oppmodel.StageProposal, oppmodel.StageNegotiation}).
		First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *opportunityContractRepo) Update(ctx context.Context, item *oppmodel.OpportunityRecord) error {
	return r.WithContext(ctx).Save(item).Error
}

func (r *opportunityContractRepo) BeginTenantTx(ctx context.Context, _ string, fn func(tx *gorm.DB) error) error {
	return r.WithContext(ctx).Transaction(fn)
}

func TestOpportunityContract_CreateListStageCloseAndActivities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openContractDB(t, "opportunity_contract")
	require.NoError(t, ensureOpportunityContractTables(db))
	router := setupOpportunityContractRouter(db)
	leadUUID := insertOpportunityContractLead(t, db, "converted")

	createResp := postJSON(t, router, "/api/v1/admin/opportunity/records", map[string]any{
		"lead_uuid":          leadUUID,
		"title":              "合同测试商机",
		"owner_member_uuid":  opportunityContractOwner,
		"amount":             88000,
		"currency":           "CNY",
		"probability":        30,
		"expected_close_at":  "2026-06-30T10:00:00Z",
		"owner_user_uuid":    "",
		"owner_member_label": "ignored",
	})
	require.Equal(t, http.StatusCreated, createResp.Code)
	createBody := decodeBody(t, createResp)
	require.Equal(t, true, createBody["success"])
	createData := createBody["data"].(map[string]any)
	opportunityUUID := createData["opportunity_uuid"].(string)
	require.NotEmpty(t, opportunityUUID)
	require.Equal(t, "open", createData["stage"])
	require.Equal(t, opportunityContractOwner, createData["owner_member_uuid"])
	require.Equal(t, opportunityContractOwner, createData["owner_user_uuid"])
	require.Equal(t, opportunityContractActor, createData["created_by_member_uuid"])

	duplicateResp := postJSON(t, router, "/api/v1/admin/opportunity/records", map[string]any{
		"lead_uuid":         leadUUID,
		"title":             "重复商机",
		"owner_member_uuid": opportunityContractOwner,
	})
	require.Equal(t, http.StatusConflict, duplicateResp.Code)
	duplicateBody := decodeBody(t, duplicateResp)
	require.Equal(t, false, duplicateBody["success"])

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/opportunity/records?owner_member_uuid="+opportunityContractOwner, nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	listBody := decodeBody(t, listRec)
	items := listBody["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)

	stageResp := postJSON(t, router, "/api/v1/admin/opportunity/records/"+opportunityUUID+"/stage", map[string]any{
		"stage": "proposal",
	})
	require.Equal(t, http.StatusOK, stageResp.Code)
	require.Equal(t, "proposal", decodeBody(t, stageResp)["data"].(map[string]any)["stage"])

	closeResp := postJSON(t, router, "/api/v1/admin/opportunity/records/"+opportunityUUID+"/close", map[string]any{
		"result":      "lost",
		"lost_reason": "客户预算取消",
	})
	require.Equal(t, http.StatusOK, closeResp.Code)
	closeData := decodeBody(t, closeResp)["data"].(map[string]any)
	require.Equal(t, "lost", closeData["stage"])
	require.Equal(t, "客户预算取消", closeData["lost_reason"])

	activitiesReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/opportunity/records/"+opportunityUUID+"/activities", nil)
	activitiesRec := httptest.NewRecorder()
	router.ServeHTTP(activitiesRec, activitiesReq)
	require.Equal(t, http.StatusOK, activitiesRec.Code)
	activityItems := decodeBody(t, activitiesRec)["data"].(map[string]any)["items"].([]any)
	require.Len(t, activityItems, 3)
}

func TestOpportunityContract_UnqualifiedLeadReturns422(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openContractDB(t, "opportunity_contract_unqualified")
	require.NoError(t, ensureOpportunityContractTables(db))
	router := setupOpportunityContractRouter(db)
	leadUUID := insertOpportunityContractLead(t, db, "assigned")

	resp := postJSON(t, router, "/api/v1/admin/opportunity/records", map[string]any{
		"lead_uuid":         leadUUID,
		"title":             "未合格线索商机",
		"owner_member_uuid": opportunityContractOwner,
	})
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	body := decodeBody(t, resp)
	require.Equal(t, false, body["success"])
	require.Contains(t, body["error"].(map[string]any)["message"], "线索尚未达到可创建商机状态")
}

func setupOpportunityContractRouter(db *gorm.DB) *gin.Engine {
	repo := &opportunityContractRepo{DB: db}
	service := oppsvc.NewServiceWithDB(
		db,
		repo,
		opprepo.NewOpportunityActivityRepository(db),
	)
	handler := httpopp.NewHandler(service)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", opportunityContractTenant)
		c.Set("member_uuid", opportunityContractActor)
		authx.SetTenantContext(c, authx.TenantContext{
			TenantUUID: opportunityContractTenant,
			MemberUUID: opportunityContractActor,
		})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), opportunityContractTenant))
		c.Request = c.Request.WithContext(authx.ContextWithMemberUUID(c.Request.Context(), opportunityContractActor))
		c.Next()
	})
	group := r.Group("/api/v1/admin/opportunity")
	group.GET("/dashboard", handler.Dashboard)
	group.GET("/records", handler.List)
	group.POST("/records", handler.Create)
	group.GET("/records/:opportunity_uuid", handler.Get)
	group.PUT("/records/:opportunity_uuid", handler.Update)
	group.POST("/records/:opportunity_uuid/stage", handler.Stage)
	group.POST("/records/:opportunity_uuid/close", handler.Close)
	group.POST("/records/:opportunity_uuid/reopen", handler.Reopen)
	group.POST("/records/:opportunity_uuid/risk", handler.MarkRisk)
	group.GET("/records/:opportunity_uuid/activities", handler.Activities)
	return r
}

func ensureOpportunityContractTables(db *gorm.DB) error {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS lead_capture_leads (
			lead_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			display_name TEXT,
			phone TEXT,
			email TEXT,
			status TEXT NOT NULL,
			owner_user_uuid TEXT,
			source_channel TEXT,
			source_app_type TEXT,
			source_account_uuid TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS customer_accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_uuid TEXT NOT NULL,
			customer_uuid TEXT NOT NULL,
			email TEXT,
			phone TEXT,
			password_hash TEXT,
			status TEXT NOT NULL,
			metadata TEXT,
			email_verified BOOLEAN,
			phone_verified BOOLEAN,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS opportunity_records (
			opportunity_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			lead_uuid TEXT NOT NULL,
			title TEXT NOT NULL,
			stage TEXT NOT NULL,
			amount NUMERIC,
			currency TEXT NOT NULL,
			probability INTEGER NOT NULL,
			owner_user_uuid TEXT NOT NULL,
			source_channel TEXT,
			source_app_type TEXT,
			source_account_uuid TEXT,
			external_userid TEXT,
			expected_close_at DATETIME,
			won_at DATETIME,
			lost_at DATETIME,
			lost_reason TEXT,
			risk_flags TEXT NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS opportunity_activities (
			activity_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			opportunity_uuid TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			from_stage TEXT,
			to_stage TEXT,
			payload TEXT NOT NULL,
			operator_user_uuid TEXT NOT NULL,
			request_id TEXT,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS opportunity_line_items (
			item_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			opportunity_uuid TEXT NOT NULL,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			quantity NUMERIC NOT NULL,
			unit_price NUMERIC NOT NULL,
			total_amount NUMERIC NOT NULL,
			currency TEXT NOT NULL,
			storage_provider TEXT,
			object_key TEXT,
			file_name TEXT,
			file_size INTEGER NOT NULL,
			content_type TEXT,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS opportunity_tasks (
			task_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			opportunity_uuid TEXT NOT NULL,
			title TEXT NOT NULL,
			due_at DATETIME,
			status TEXT NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func insertOpportunityContractLead(t *testing.T, db *gorm.DB, status string) string {
	t.Helper()
	leadUUID := uuid.NewString()
	sourceAccountUUID := uuid.NewString()
	err := db.Exec(
		`INSERT INTO lead_capture_leads (lead_uuid, tenant_uuid, display_name, phone, email, status, owner_user_uuid, source_channel, source_app_type, source_account_uuid, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		leadUUID,
		opportunityContractTenant,
		fmt.Sprintf("合同线索-%s", status),
		"13800007100",
		status+"@example.test",
		status,
		opportunityContractOwner,
		"wechat",
		"openwork",
		sourceAccountUUID,
	).Error
	require.NoError(t, err)
	return leadUUID
}

func postJSON(t *testing.T, router *gin.Engine, path string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

var _ opprepo.OpportunityRepository = (*opportunityContractRepo)(nil)
