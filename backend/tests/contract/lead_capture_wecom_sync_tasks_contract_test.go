package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httplead "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLeadCaptureWeComSyncTasksContract_ListWithStatusFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"
	now := time.Now().UTC()

	db := openContractDB(t, "lead_capture_wecom_sync_tasks_contract")
	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	require.NoError(t, db.Create(&leadmodel.LeadSyncTask{
		TaskUUID:             "90000000-0000-4000-8000-000000000001",
		TenantUUID:           tenantUUID,
		Channel:              "wechat",
		AppType:              "wecom",
		ChannelAccountUUID:   accountUUID,
		AccountResolveSource: "default",
		TaskProvider:         "local_fallback",
		TriggerType:          "manual",
		Status:               "failed",
		ErrorMessage:         "provider timeout",
		StartedAt:            &now,
		FinishedAt:           &now,
	}).Error)
	require.NoError(t, db.Create(&leadmodel.LeadSyncTask{
		TaskUUID:             "90000000-0000-4000-8000-000000000002",
		TenantUUID:           tenantUUID,
		Channel:              "wechat",
		AppType:              "wecom",
		ChannelAccountUUID:   accountUUID,
		AccountResolveSource: "default",
		TaskProvider:         "local_fallback",
		TriggerType:          "manual",
		Status:               "success",
		StatsTotal:           3,
		StatsCreated:         2,
		StatsUpdated:         1,
		StartedAt:            &now,
		FinishedAt:           &now,
	}).Error)

	svc := leadsvc.NewWeComSyncService(taskRepo, nil, nil)
	handler := httplead.NewWeComSyncHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.GET("/api/v1/admin/leads/wecom/sync-tasks", handler.ListSyncTasks)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/wecom/sync-tasks?status=failed&limit=10", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)

	items, ok := data["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)

	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "90000000-0000-4000-8000-000000000001", first["task_uuid"])
	require.Equal(t, "failed", first["status"])
	require.Equal(t, "provider timeout", first["error_message"])
}
