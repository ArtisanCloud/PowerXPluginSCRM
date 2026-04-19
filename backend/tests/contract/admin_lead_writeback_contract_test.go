package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httplead "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLeadWritebackPolicyContract_GetAndUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	db := openContractDB(t, "lead_writeback_policy_contract")
	require.NoError(t, ensureContractSyncFoundationTables(db))

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	syncRepo := socialrepo.NewSyncFoundationRepository(db)
	svc := leadsvc.NewWeComSyncService(taskRepo, nil, nil).WithSyncFoundation(syncRepo)
	handler := httplead.NewWeComSyncHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.GET("/api/v1/admin/leads/wecom/writeback-policy", handler.GetWritebackPolicy)
	r.PUT("/api/v1/admin/leads/wecom/writeback-policy", handler.UpdateWritebackPolicy)

	recUnsupported := httptest.NewRecorder()
	reqUnsupported := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/wecom/writeback-policy?channel=dingtalk&app_type=dingtalk", nil)
	r.ServeHTTP(recUnsupported, reqUnsupported)
	require.Equal(t, http.StatusOK, recUnsupported.Code)
	var unsupported map[string]any
	require.NoError(t, json.Unmarshal(recUnsupported.Body.Bytes(), &unsupported))
	unsupportedData := unsupported["data"].(map[string]any)
	require.Equal(t, "not_supported", unsupportedData["capability_status"])

	updateBody := bytes.NewBufferString(`{
		"enabled":true,
		"overwrite_mode":"force",
		"mapping_rules":{"whitelist":["phone","status"]},
		"protected_fields":{"fields":["tenant_uuid"]}
	}`)
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/v1/admin/leads/wecom/writeback-policy", updateBody)
	reqUpdate.Header.Set("Content-Type", "application/json")
	recUpdate := httptest.NewRecorder()
	r.ServeHTTP(recUpdate, reqUpdate)
	require.Equal(t, http.StatusOK, recUpdate.Code)

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/wecom/writeback-policy", nil)
	r.ServeHTTP(recGet, reqGet)
	require.Equal(t, http.StatusOK, recGet.Code)

	var fetched map[string]any
	require.NoError(t, json.Unmarshal(recGet.Body.Bytes(), &fetched))
	data := fetched["data"].(map[string]any)
	require.Equal(t, "supported", data["capability_status"])
	require.Equal(t, "force", data["overwrite_mode"])
	require.Equal(t, true, data["enabled"])
}

func TestLeadWritebackDeadLetterContract_ListAndReplay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	deadLetterUUID := "90000000-0000-4000-8000-000000000111"
	jobUUID := "90000000-0000-4000-8000-000000000112"
	db := openContractDB(t, "lead_writeback_deadletter_contract")
	require.NoError(t, ensureContractSyncFoundationTables(db))
	now := time.Now().UTC().Format(time.RFC3339)

	require.NoError(t, db.Exec(`
		INSERT INTO social_sync_dead_letters (
			dead_letter_uuid, tenant_uuid, job_uuid, domain, direction, last_error_code,
			last_error_message, retry_exhausted_at, replay_status, replayed_by, payload, created_at, updated_at
		) VALUES (?, ?, ?, 'leads', 'push', 'WRITEBACK_FAILED', 'mock-failed', ?, 'pending', '', '{}', ?, ?)
	`, deadLetterUUID, tenantUUID, jobUUID, now, now, now).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	syncRepo := socialrepo.NewSyncFoundationRepository(db)
	svc := leadsvc.NewWeComSyncService(taskRepo, nil, nil).WithSyncFoundation(syncRepo)
	handler := httplead.NewWeComSyncHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.GET("/api/v1/admin/leads/wecom/writeback-dead-letters", handler.ListWritebackDeadLetters)
	r.POST("/api/v1/admin/leads/wecom/writeback-dead-letters/:dead_letter_uuid/replay", handler.ReplayWritebackDeadLetter)

	recList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/wecom/writeback-dead-letters", nil)
	r.ServeHTTP(recList, reqList)
	require.Equal(t, http.StatusOK, recList.Code)

	var listed map[string]any
	require.NoError(t, json.Unmarshal(recList.Body.Bytes(), &listed))
	items := listed["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)

	recReplay := httptest.NewRecorder()
	reqReplay := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/wecom/writeback-dead-letters/"+deadLetterUUID+"/replay", nil)
	r.ServeHTTP(recReplay, reqReplay)
	require.Equal(t, http.StatusOK, recReplay.Code)

	replayed, err := syncRepo.ReplayDeadLetter(context.Background(), tenantUUID, deadLetterUUID, "contract-check")
	require.NoError(t, err)
	require.Equal(t, "replayed", replayed.ReplayStatus)
}
