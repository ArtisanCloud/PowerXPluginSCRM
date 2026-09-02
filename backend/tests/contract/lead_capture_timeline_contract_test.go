package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httplead "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestLeadCaptureTimelineContract_ReturnsUnifiedPagedEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-4000-8000-000000000001"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	memberUUID := "30000000-0000-4000-8000-000000000001"
	db := openContractDB(t, "lead_capture_timeline_contract")
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS lead_capture_assignments (assignment_uuid TEXT PRIMARY KEY, tenant_uuid TEXT NOT NULL, lead_uuid TEXT NOT NULL, owner_user_uuid TEXT NOT NULL, reason TEXT, created_at DATETIME);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_status_history (history_uuid TEXT PRIMARY KEY, tenant_uuid TEXT NOT NULL, lead_uuid TEXT NOT NULL, from_status TEXT NOT NULL, to_status TEXT NOT NULL, changed_at DATETIME);`,
	} {
		require.NoError(t, db.Exec(statement).Error)
	}
	require.NoError(t, db.Create(&leadmodel.Lead{LeadUUID: leadUUID, TenantUUID: tenantUUID, DisplayName: "lead", Status: leadmodel.LeadStatusEngaging}).Error)
	require.NoError(t, db.Create(&leadmodel.LeadActivity{
		ActivityUUID: "20000000-0000-4000-8000-000000000001", TenantUUID: tenantUUID, LeadUUID: leadUUID,
		ActivityType: leadmodel.LeadActivityTypeManual,
		Payload:      datatypes.JSONMap{"stage_key": leadmodel.LeadStatusEngaging, "method": "wechat", "content": "确认需求", "actor_type": leadmodel.LeadAuditActorTypeMember, "operator_member_uuid": memberUUID},
	}).Error)

	handler := httplead.NewLeadHandler(leadsvc.NewLeadService(leadrepo.NewLeadRepository(db)))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		requestContext := authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID)
		c.Request = c.Request.WithContext(requestContext)
		c.Next()
	})
	router.GET("/api/v1/admin/leads/:lead_id/timeline", handler.ListTimeline)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/"+leadUUID+"/timeline?stage_key=engaging&event_type=manual_activity&page=1&page_size=10", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	data := payload["data"].(map[string]any)
	require.Equal(t, float64(1), data["total"])
	require.Equal(t, float64(1), data["page"])
	items := data["items"].([]any)
	require.Len(t, items, 1)
	event := items[0].(map[string]any)
	require.Equal(t, leadsvc.LeadTimelineEventManualActivity, event["event_type"])
	require.Equal(t, leadmodel.LeadAuditActorTypeMember, event["actor_type"])
	require.Equal(t, memberUUID, event["actor_member_uuid"])
}
