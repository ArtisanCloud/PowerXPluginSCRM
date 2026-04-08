package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	socialhttp "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/social_channel_governance"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOrgSyncConflictContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	db := openTagSyncContractDB(t, "org_sync_contract")
	seedTagConflict(t, db, tenantUUID, "org", "dept-001")

	syncRepo := socialrepo.NewSyncFoundationRepository(db)
	jobSvc := socialsvc.NewSyncJobService(
		syncRepo,
		socialsvc.NewIdempotencyService(),
		socialsvc.NewSyncScheduler(),
		socialsvc.NewCapabilityService(socialsvc.NewChannelFactory()),
	)
	jobHandler := socialhttp.NewSyncJobHandler(jobSvc, socialsvc.NewSyncOrchestrator(nil, socialsvc.NewSyncScheduler(), jobSvc), socialsvc.NewCapabilityService(socialsvc.NewChannelFactory()), socialsvc.NewConflictResolutionService(syncRepo))
	conflictHandler := socialhttp.NewConflictHandler(socialsvc.NewConflictResolutionService(syncRepo), socialsvc.NewRetryDeadletterService(syncRepo))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.POST("/api/v1/admin/social/openwork/foundation/sync/jobs", jobHandler.Create)
	r.GET("/api/v1/admin/social/openwork/foundation/sync/conflicts", conflictHandler.List)

	createPayload := map[string]any{
		"domain":    "org",
		"direction": "push",
		"mode":      "pushback",
		"channel":   "wechat",
		"app_type":  "wecom",
		"payload": map[string]any{
			"changes": []map[string]any{
				{"entity_type": "department", "entity_id": "dept-001", "action": "move"},
			},
		},
	}
	raw, err := json.Marshal(createPayload)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/social/openwork/foundation/sync/jobs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/social/openwork/foundation/sync/conflicts?domain=org", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &payload))
	require.Equal(t, true, payload["success"])
	data := payload["data"].(map[string]any)
	items := data["items"].([]any)
	require.GreaterOrEqual(t, len(items), 1)
}
