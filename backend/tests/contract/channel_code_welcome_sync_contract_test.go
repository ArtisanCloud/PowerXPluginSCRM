package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	domainrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httplead "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestChannelCodeWelcomeSyncContract_PublishAndStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000081"

	db := openContractDB(t, "channel_code_welcome_sync_contract")
	require.NoError(t, ensureChannelCodeContractTables(db))
	require.NoError(t, ensureWelcomeSyncContractTables(db))
	codeUUID := seedWelcomeSyncContractData(t, db, tenantUUID)

	r := setupWelcomeSyncContractRouter(db, tenantUUID, []string{"tenant.admin"})

	triggerReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/channel-codes/"+codeUUID+"/welcome-config/sync", bytes.NewBuffer(nil))
	triggerRec := httptest.NewRecorder()
	r.ServeHTTP(triggerRec, triggerReq)
	require.Equal(t, http.StatusOK, triggerRec.Code)

	var triggerResp map[string]any
	require.NoError(t, json.Unmarshal(triggerRec.Body.Bytes(), &triggerResp))
	require.Equal(t, true, triggerResp["success"])
	triggerData := triggerResp["data"].(map[string]any)
	require.Equal(t, domainmodel.WelcomeSyncStatusManualRequired, triggerData["sync_status"])
	require.Equal(t, leadsvc.WelcomeSyncErrorChannelAuthInvalid, triggerData["error_code"])

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/channel-codes/"+codeUUID+"/welcome-config/sync-status", nil)
	statusRec := httptest.NewRecorder()
	r.ServeHTTP(statusRec, statusReq)
	require.Equal(t, http.StatusOK, statusRec.Code)

	var statusResp map[string]any
	require.NoError(t, json.Unmarshal(statusRec.Body.Bytes(), &statusResp))
	statusData := statusResp["data"].(map[string]any)
	require.Equal(t, domainmodel.WelcomeSyncStatusManualRequired, statusData["sync_status"])
	require.EqualValues(t, 3, statusData["latest_attempt_no"])
	require.Contains(t, statusData["last_sync_error"], leadsvc.WelcomeSyncErrorChannelAuthInvalid)
}

func TestChannelCodeWelcomeSyncContract_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000082"

	db := openContractDB(t, "channel_code_welcome_sync_contract_forbidden")
	require.NoError(t, ensureChannelCodeContractTables(db))
	require.NoError(t, ensureWelcomeSyncContractTables(db))
	codeUUID := seedWelcomeSyncContractData(t, db, tenantUUID)

	r := setupWelcomeSyncContractRouter(db, tenantUUID, []string{"viewer"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/channel-codes/"+codeUUID+"/welcome-config/sync", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, false, resp["success"])
	errObj := resp["error"].(map[string]any)
	require.Equal(t, "FORBIDDEN", errObj["code"])
}

func setupWelcomeSyncContractRouter(db *gorm.DB, tenantUUID string, roles []string) *gin.Engine {
	repos := domainrepo.NewBundle(db)
	svc := leadsvc.NewWelcomeSyncService(
		repos.ChannelCodes,
		repos.WelcomeConfigs,
		repos.WelcomeSyncAttempt,
		leadsvc.NewWeComWelcomeAdapter(),
		nil,
	).WithRetryPolicy([]time.Duration{0, 0, 0}, 3)
	handler := httplead.NewWelcomeSyncHandler(svc, leadsvc.NewWelcomeSyncAuthz())

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID, Roles: roles})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	group := r.Group("/api/v1/admin/leads")
	group.POST("/channel-codes/:code_uuid/welcome-config/sync", handler.TriggerSync)
	group.GET("/channel-codes/:code_uuid/welcome-config/sync-status", handler.GetSyncStatus)
	return r
}

func ensureWelcomeSyncContractTables(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS lead_capture_code_welcome_sync_attempts (
		attempt_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		code_uuid TEXT NOT NULL,
		config_version INTEGER NOT NULL,
		trigger_source TEXT NOT NULL,
		attempt_no INTEGER NOT NULL,
		result TEXT NOT NULL,
		error_code TEXT,
		error_message TEXT,
		started_at DATETIME,
		finished_at DATETIME,
		created_at DATETIME
	);`).Error
}

func seedWelcomeSyncContractData(t *testing.T, db *gorm.DB, tenantUUID string) string {
	t.Helper()
	codeUUID := uuid.NewString()
	now := time.Now().UTC()
	require.NoError(t, db.Create(&domainmodel.ChannelCode{
		CodeUUID:           codeUUID,
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "11111111-1111-4111-8111-111111111181",
		CodeKey:            "welcome-sync-contract",
		DisplayName:        "欢迎语同步合同",
		TargetType:         "group",
		TargetID:           "group-contract",
		Status:             domainmodel.ChannelCodeStatusActive,
		CreatedBy:          "system",
		UpdatedBy:          "system",
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error)
	require.NoError(t, db.Create(&domainmodel.CodeWelcomeConfig{
		ConfigUUID:     uuid.NewString(),
		TenantUUID:     tenantUUID,
		CodeUUID:       codeUUID,
		WelcomeEnabled: true,
		MessageContent: []byte(`{"text":"hello","mock_error_code":"CHANNEL_AUTH_INVALID"}`),
		SyncStatus:     domainmodel.WelcomeSyncStatusPending,
		Version:        1,
		CreatedBy:      "system",
		UpdatedBy:      "system",
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)
	return codeUUID
}
