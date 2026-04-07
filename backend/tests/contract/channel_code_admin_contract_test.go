package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httplead "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestChannelCodeAdminContract_CreateAndList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000011"

	db := openContractDB(t, "channel_code_admin_contract")
	require.NoError(t, ensureChannelCodeContractTables(db))
	r := setupChannelCodeContractRouter(db, tenantUUID)

	createPayload := map[string]any{
		"channel":              "wechat",
		"app_type":             "wecom",
		"channel_account_uuid": "11111111-1111-4111-8111-111111111111",
		"code_key":             "wecom-sales-001",
		"display_name":         "销售群引流码",
		"target_type":          "group",
		"target_id":            "group-001",
	}
	createBody, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/channel-codes", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	require.Equal(t, true, createResp["success"])
	data, ok := createResp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "wecom-sales-001", data["code_key"])
	require.Equal(t, "draft", data["status"])

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/channel-codes?channel=wechat", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)

	var listResp map[string]any
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResp))
	require.Equal(t, true, listResp["success"])
	listData, ok := listResp["data"].(map[string]any)
	require.True(t, ok)
	items, ok := listData["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
}

func setupChannelCodeContractRouter(db *gorm.DB, tenantUUID string) *gin.Engine {
	repos := domainrepo.NewBundle(db)
	channelCodeHandler := httplead.NewChannelCodeHandler(leadsvc.NewChannelCodeService(repos.ChannelCodes, nil))
	welcomeHandler := httplead.NewWelcomeConfigHandler(leadsvc.NewWelcomeConfigService(repos.WelcomeConfigs, repos.ConfigChangeLogs, nil))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})

	group := r.Group("/api/v1/admin/leads")
	group.POST("/channel-codes", channelCodeHandler.Create)
	group.GET("/channel-codes", channelCodeHandler.List)
	group.PATCH("/channel-codes/:code_uuid/status", channelCodeHandler.UpdateStatus)
	group.PUT("/channel-codes/:code_uuid/welcome-config", welcomeHandler.Save)
	group.GET("/channel-codes/:code_uuid/welcome-config/history", welcomeHandler.ListHistory)
	return r
}

func ensureChannelCodeContractTables(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS lead_capture_channel_codes (
			code_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			code_key TEXT NOT NULL,
			display_name TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			status TEXT NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_channel_codes_tenant_channel_code
			ON lead_capture_channel_codes (tenant_uuid, channel, code_key);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_code_welcome_configs (
			config_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			code_uuid TEXT NOT NULL,
			welcome_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			message_content TEXT NOT NULL,
			sync_status TEXT NOT NULL,
			last_sync_error TEXT,
			last_synced_at DATETIME,
			version INTEGER NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_code_welcome_cfg_code
			ON lead_capture_code_welcome_configs (code_uuid);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_code_config_change_logs (
			change_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			code_uuid TEXT NOT NULL,
			config_uuid TEXT NOT NULL,
			version INTEGER NOT NULL,
			summary TEXT NOT NULL,
			changed_fields TEXT,
			previous_content TEXT,
			next_content TEXT,
			changed_by TEXT NOT NULL,
			created_at DATETIME
		);`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
