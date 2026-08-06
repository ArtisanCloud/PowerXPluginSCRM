package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	httpacq "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/acquisition"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestStaffLiveCodeContract_CreateListUpdateStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000211"
	memberUUID := "11111111-1111-4111-8111-111111111211"

	db := openContractDB(t, "staff_live_code_contract")
	require.NoError(t, ensureAcquisitionContractTables(db))
	require.NoError(t, seedConfirmedMemberMapping(db, tenantUUID, memberUUID))
	r := setupStaffLiveCodeContractRouter(db, tenantUUID)

	createPayload := map[string]any{
		"channel":                     "wechat",
		"app_type":                    "wecom",
		"channel_account_uuid":        "22222222-2222-4222-8222-222222222211",
		"activity_name":               "春季活动",
		"code_key":                    "staff-code-001",
		"member_uuids":                []string{memberUUID},
		"corp_tag_ids":                []string{"tag_a"},
		"new_customer_remark_enabled": true,
	}
	body, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/acquisition/staff-codes", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code)

	var createResp map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	require.Equal(t, true, createResp["success"])
	createData := createResp["data"].(map[string]any)
	require.Equal(t, "draft", createData["status"])
	staffCodeUUID := createData["staff_code_uuid"].(string)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/staff-codes?status=draft", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	var listResp map[string]any
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResp))
	items := listResp["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)

	statusBody := bytes.NewBufferString(`{"status":"active"}`)
	statusReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/leads/acquisition/staff-codes/"+staffCodeUUID+"/status", statusBody)
	statusReq.Header.Set("Content-Type", "application/json")
	statusRec := httptest.NewRecorder()
	r.ServeHTTP(statusRec, statusReq)
	require.Equal(t, http.StatusOK, statusRec.Code)
	var statusResp map[string]any
	require.NoError(t, json.Unmarshal(statusRec.Body.Bytes(), &statusResp))
	require.Equal(t, "active", statusResp["data"].(map[string]any)["status"])
}

func setupStaffLiveCodeContractRouter(db *gorm.DB, tenantUUID string) *gin.Engine {
	repos := acqrepo.NewBundle(db)
	handler := httpacq.NewStaffLiveCodeHandler(acqsvc.NewStaffLiveCodeService(repos.StaffLiveCodes))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	group := r.Group("/api/v1/admin/leads/acquisition")
	group.POST("/staff-codes", handler.Create)
	group.GET("/staff-codes", handler.List)
	group.PATCH("/staff-codes/:staff_code_uuid/status", handler.UpdateStatus)
	return r
}

func ensureAcquisitionContractTables(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS acquisition_staff_live_codes (
			staff_code_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			activity_name TEXT NOT NULL,
			code_key TEXT NOT NULL,
			state TEXT,
			config_id TEXT,
			qr_code TEXT,
			member_uuids TEXT NOT NULL,
			corp_tag_ids TEXT NOT NULL,
			new_customer_remark_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			status TEXT NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_acq_staff_codes_tenant_code_key
			ON acquisition_staff_live_codes (tenant_uuid, code_key);`,
		`CREATE TABLE IF NOT EXISTS acquisition_staff_welcome_configs (
			config_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			staff_code_uuid TEXT NOT NULL,
			welcome_mode TEXT NOT NULL,
			content_blocks TEXT NOT NULL,
			payload_preview TEXT NOT NULL,
			sync_status TEXT NOT NULL,
			last_sync_error TEXT,
			last_synced_at DATETIME,
			version INTEGER NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_acq_staff_welcome_code
			ON acquisition_staff_welcome_configs (staff_code_uuid);`,
		`CREATE TABLE IF NOT EXISTS acquisition_staff_welcome_sync_attempts (
			attempt_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			staff_code_uuid TEXT NOT NULL,
			config_version INTEGER NOT NULL,
			attempt_no INTEGER NOT NULL,
			result TEXT NOT NULL,
			error_code TEXT,
			error_message TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS acquisition_group_live_codes (
			group_code_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			activity_name TEXT NOT NULL,
			corp_tag_ids TEXT,
			new_customer_remark_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			state TEXT,
			config_id TEXT,
			join_scene INTEGER NOT NULL DEFAULT 1,
			skip_verify BOOLEAN NOT NULL DEFAULT FALSE,
			auto_create_room BOOLEAN NOT NULL DEFAULT FALSE,
			target_chat_count INTEGER NOT NULL DEFAULT 0,
			target_chat_ids TEXT,
			shard_count INTEGER NOT NULL DEFAULT 0,
			capacity_total INTEGER NOT NULL DEFAULT 0,
			capacity_used INTEGER NOT NULL DEFAULT 0,
			shard_config_ids TEXT,
			qr_code TEXT,
			status TEXT NOT NULL,
			sync_status TEXT NOT NULL DEFAULT 'pending',
			last_sync_error TEXT,
			last_synced_at DATETIME,
			capability_status TEXT NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS org_sync_member_bindings (
			member_binding_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			main_member_id TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedConfirmedMemberMapping(db *gorm.DB, tenantUUID, sourceMemberUUID string) error {
	now := time.Now().UTC()
	return db.Exec(
		`INSERT INTO org_sync_member_bindings
			(member_binding_uuid, tenant_uuid, main_member_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		"map-"+sourceMemberUUID,
		tenantUUID,
		sourceMemberUUID,
		now,
		now,
	).Error
}

func seedStaffLiveCodeContractData(t *testing.T, db *gorm.DB, tenantUUID string) string {
	t.Helper()
	item := &acqmodel.StaffLiveCode{
		StaffCodeUUID:      "33333333-3333-4333-8333-333333333211",
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "22222222-2222-4222-8222-222222222211",
		ActivityName:       "默认活动",
		CodeKey:            "seed-code",
		MemberUUIDs:        []string{"11111111-1111-4111-8111-111111111211"},
		CorpTagIDs:         []string{},
		RemarkEnabled:      false,
		Status:             acqmodel.LiveCodeStatusActive,
		CreatedBy:          "system",
		UpdatedBy:          "system",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	require.NoError(t, db.Create(item).Error)
	return item.StaffCodeUUID
}
