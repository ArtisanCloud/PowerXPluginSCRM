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

func TestStaffWelcomeContract_SaveSyncAndStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000212"

	db := openContractDB(t, "staff_welcome_contract")
	require.NoError(t, ensureAcquisitionContractTables(db))
	staffCodeUUID := seedStaffCodeForWelcomeContract(t, db, tenantUUID)
	r := setupStaffWelcomeContractRouter(db, tenantUUID)

	saveBody := bytes.NewBufferString(`{"welcome_mode":"send","content_blocks":[{"type":"text","text":"hello"}]}`)
	saveReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/leads/acquisition/staff-codes/"+staffCodeUUID+"/welcome-config", saveBody)
	saveReq.Header.Set("Content-Type", "application/json")
	saveRec := httptest.NewRecorder()
	r.ServeHTTP(saveRec, saveReq)
	require.Equal(t, http.StatusOK, saveRec.Code)
	var saveResp map[string]any
	require.NoError(t, json.Unmarshal(saveRec.Body.Bytes(), &saveResp))
	require.Equal(t, true, saveResp["success"])
	require.Equal(t, "pending", saveResp["data"].(map[string]any)["sync_status"])

	syncReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/acquisition/staff-codes/"+staffCodeUUID+"/welcome-config/sync", nil)
	syncRec := httptest.NewRecorder()
	r.ServeHTTP(syncRec, syncReq)
	require.Equal(t, http.StatusOK, syncRec.Code)
	var syncResp map[string]any
	require.NoError(t, json.Unmarshal(syncRec.Body.Bytes(), &syncResp))
	require.Equal(t, true, syncResp["success"])
	syncData := syncResp["data"].(map[string]any)
	require.Equal(t, "success", syncData["sync_status"])
	require.EqualValues(t, 1, syncData["attempt_no"])
	require.Contains(t, syncData["message"], "welcome_code")

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/staff-codes/"+staffCodeUUID+"/welcome-config/sync-status", nil)
	statusRec := httptest.NewRecorder()
	r.ServeHTTP(statusRec, statusReq)
	require.Equal(t, http.StatusOK, statusRec.Code)
	var statusResp map[string]any
	require.NoError(t, json.Unmarshal(statusRec.Body.Bytes(), &statusResp))
	statusData := statusResp["data"].(map[string]any)
	require.Equal(t, "success", statusData["sync_status"])
	require.EqualValues(t, 1, statusData["latest_attempt_no"])
	require.Empty(t, statusData["last_sync_error"])
}

func setupStaffWelcomeContractRouter(db *gorm.DB, tenantUUID string) *gin.Engine {
	repos := acqrepo.NewBundle(db)
	handler := httpacq.NewStaffWelcomeHandler(acqsvc.NewStaffWelcomeService(
		repos.StaffLiveCodes,
		repos.StaffWelcomeConfigs,
		repos.StaffWelcomeAttempt,
	))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID, Roles: []string{"tenant.admin"}})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	group := r.Group("/api/v1/admin/leads/acquisition")
	group.PUT("/staff-codes/:staff_code_uuid/welcome-config", handler.Save)
	group.POST("/staff-codes/:staff_code_uuid/welcome-config/sync", handler.TriggerSync)
	group.GET("/staff-codes/:staff_code_uuid/welcome-config/sync-status", handler.GetSyncStatus)
	return r
}

func seedStaffCodeForWelcomeContract(t *testing.T, db *gorm.DB, tenantUUID string) string {
	t.Helper()
	now := time.Now().UTC()
	item := &acqmodel.StaffLiveCode{
		StaffCodeUUID:      "33333333-3333-4333-8333-333333333212",
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "22222222-2222-4222-8222-222222222212",
		ActivityName:       "欢迎语活动",
		CodeKey:            "welcome-staff-001",
		MemberUUIDs:        []string{"11111111-1111-4111-8111-111111111212"},
		CorpTagIDs:         []string{},
		RemarkEnabled:      false,
		Status:             acqmodel.LiveCodeStatusActive,
		CreatedBy:          "system",
		UpdatedBy:          "system",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	require.NoError(t, db.Create(item).Error)
	return item.StaffCodeUUID
}
