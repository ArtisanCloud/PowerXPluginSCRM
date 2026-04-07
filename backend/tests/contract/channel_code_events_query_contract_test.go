package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	domainrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httplead "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestChannelCodeEventsQueryContract_ListAndStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000052"
	accountUUID := "11111111-1111-4111-8111-111111111152"

	db := openContractDB(t, "channel_code_events_query_contract")
	require.NoError(t, ensureChannelCodeEventContractTables(db))
	seedChannelCodeEventContractData(t, db, tenantUUID, accountUUID)

	repos := domainrepo.NewBundle(db)
	leadRepository := leadrepo.NewLeadRepository(db)
	attributionSvc := leadsvc.NewAttributionService(repos.Attributions, leadRepository, leadsvc.NewLeadService(leadRepository))
	eventSvc := leadsvc.NewChannelCodeEventService(repos.ChannelCodeEvents, repos.ChannelCodes, socialrepo.NewAccountRepository(db), attributionSvc, nil)
	adminHandler := httplead.NewChannelCodeEventsHandler(eventSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.GET("/api/v1/admin/leads/channel-codes/:code_uuid/events", adminHandler.List)

	var code domainmodel.ChannelCode
	require.NoError(t, db.Where("tenant_uuid = ?", tenantUUID).First(&code).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/channel-codes/"+code.CodeUUID+"/events?limit=20", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, true, resp["success"])
	data := resp["data"].(map[string]any)
	events := data["events"].([]any)
	require.Len(t, events, 1)
	stats := data["stats"].(map[string]any)
	require.EqualValues(t, 1, stats["touch_total"])
	require.EqualValues(t, 1, stats["intake_total"])
}

func seedChannelCodeEventContractData(t *testing.T, db *gorm.DB, tenantUUID, accountUUID string) {
	t.Helper()
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-query",
		DisplayName:     "企微查询账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-02",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}).Error)

	codeUUID := uuid.NewString()
	eventUUID := uuid.NewString()
	leadUUID := uuid.NewString()
	require.NoError(t, db.Create(&domainmodel.ChannelCode{
		CodeUUID:           codeUUID,
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: accountUUID,
		CodeKey:            "wecom-us2-query",
		DisplayName:        "US2查询",
		TargetType:         "group",
		TargetID:           "group-query",
		Status:             domainmodel.ChannelCodeStatusActive,
		CreatedBy:          "system",
		UpdatedBy:          "system",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}).Error)
	require.NoError(t, db.Exec(`INSERT INTO lead_capture_leads (lead_uuid, tenant_uuid, display_name, status, source_channel, source_app_type, source_account_uuid, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		leadUUID, tenantUUID, "合同测试线索", "new", "wechat", "wecom", accountUUID, time.Now().UTC(), time.Now().UTC()).Error)
	require.NoError(t, db.Exec(`INSERT INTO lead_capture_channel_code_events (event_uuid, tenant_uuid, channel, app_type, channel_account_uuid, code_uuid, external_event_id, event_type, idempotency_key, occurred_at, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		eventUUID, tenantUUID, "wechat", "wecom", accountUUID, codeUUID, "evt-query-001", "join", tenantUUID+":wechat:"+accountUUID+":evt-query-001", time.Now().UTC(), "{}", time.Now().UTC()).Error)
	require.NoError(t, db.Exec(`INSERT INTO lead_capture_lead_attribution_records (attribution_uuid, tenant_uuid, lead_uuid, code_uuid, event_uuid, is_primary, attribution_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), tenantUUID, leadUUID, codeUUID, eventUUID, true, "first_touch", time.Now().UTC()).Error)
}

func ensureChannelCodeEventContractTables(db *gorm.DB) error {
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
		`CREATE TABLE IF NOT EXISTS lead_capture_channel_code_events (
			event_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			code_uuid TEXT NOT NULL,
			external_event_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			idempotency_key TEXT NOT NULL UNIQUE,
			occurred_at DATETIME NOT NULL,
			payload TEXT NOT NULL,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_lead_attribution_records (
			attribution_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			lead_uuid TEXT NOT NULL,
			code_uuid TEXT NOT NULL,
			event_uuid TEXT NOT NULL,
			is_primary BOOLEAN NOT NULL DEFAULT FALSE,
			attribution_type TEXT NOT NULL,
			created_at DATETIME
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
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
