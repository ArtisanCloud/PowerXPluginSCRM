package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httplead "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLeadCaptureWeComSyncContract_ResolveDefaultAccountAndProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	defaultAccountUUID := "11111111-1111-4111-8111-111111111111"

	db := openContractDB(t, "lead_capture_wecom_sync_contract")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     defaultAccountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-default-account",
		DisplayName:     "企微默认账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	svc := leadsvc.NewWeComSyncService(taskRepo, nil, nil)
	handler := httplead.NewWeComSyncHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.POST("/api/v1/admin/leads/wecom/sync", handler.TriggerSync)

	reqBody := bytes.NewBufferString(`{"trace_id":"trace-us1-contract"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/wecom/sync", reqBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])

	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, data["task_uuid"])
	require.Equal(t, defaultAccountUUID, data["channel_account_uuid"])
	require.Equal(t, leadmodel.LeadSyncTaskResolveDefault, data["account_resolve_source"])
	require.Equal(t, leadmodel.LeadSyncTaskProviderLocalFallback, data["task_provider"])
	require.NotEmpty(t, data["status"])
}

func openContractDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, createContractTestSchema(db))
	return db
}

func createContractTestSchema(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS social_channel_accounts (
			account_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel_code TEXT NOT NULL,
			app_type TEXT NOT NULL,
			account_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			status TEXT NOT NULL,
			org_sync_default BOOLEAN NOT NULL DEFAULT FALSE,
			owner_member_uuid TEXT NOT NULL,
			member_user_uuids TEXT,
			capabilities TEXT,
			credentials TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_sync_tasks (
			task_uuid TEXT PRIMARY KEY,
			external_task_id TEXT,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			account_resolve_source TEXT NOT NULL,
			task_provider TEXT NOT NULL,
			trigger_type TEXT NOT NULL,
			status TEXT NOT NULL,
			stats_total INTEGER NOT NULL DEFAULT 0,
			stats_created INTEGER NOT NULL DEFAULT 0,
			stats_updated INTEGER NOT NULL DEFAULT 0,
			stats_merged INTEGER NOT NULL DEFAULT 0,
			error_code TEXT,
			error_message TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);`,
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
		`CREATE TABLE IF NOT EXISTS lead_capture_sources (
			source_uuid TEXT PRIMARY KEY,
			lead_uuid TEXT NOT NULL,
			tenant_uuid TEXT NOT NULL,
			channel_code TEXT,
			app_type TEXT,
			account_uuid TEXT,
			campaign_code TEXT,
			utm_source TEXT,
			utm_medium TEXT,
			utm_campaign TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_activities (
			activity_uuid TEXT PRIMARY KEY,
			lead_uuid TEXT NOT NULL,
			tenant_uuid TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			payload TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_conversation_events (
			event_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			external_event_id TEXT NOT NULL,
			idempotency_key TEXT NOT NULL UNIQUE,
			conversation_id TEXT NOT NULL,
			actor_type TEXT NOT NULL,
			actor_id TEXT NOT NULL,
			direction TEXT NOT NULL,
			message_type TEXT NOT NULL,
			content_text TEXT,
			raw_payload TEXT,
			occurred_at DATETIME NOT NULL,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_conversation_bindings (
			binding_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			lead_uuid TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			bind_source TEXT NOT NULL,
			status TEXT NOT NULL,
			created_by TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_conv_bindings_active
			ON lead_capture_conversation_bindings (tenant_uuid, conversation_id, status);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_conversation_pending (
			pending_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			event_uuid TEXT NOT NULL UNIQUE,
			channel_account_uuid TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			reason TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME,
			resolved_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_realtime_projection (
			projection_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			lead_uuid TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			latest_message TEXT,
			latest_actor_type TEXT,
			latest_at DATETIME,
			unread_count INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME,
			created_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_realtime_projection
			ON lead_capture_realtime_projection (tenant_uuid, lead_uuid, conversation_id);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_channel_rules (
			rule_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			auto_create_lead_from_customer_dm BOOLEAN NOT NULL DEFAULT FALSE,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_channel_rules_tenant_channel_app
			ON lead_capture_channel_rules (tenant_uuid, channel, app_type);`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
