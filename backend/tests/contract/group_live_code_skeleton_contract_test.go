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
	httpwebhooks "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/webhooks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGroupLiveCodeSkeletonContract_ListAndWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000213"

	db := openContractDB(t, "group_live_code_skeleton_contract")
	require.NoError(t, ensureAcquisitionContractTables(db))
	seedGroupCodeForSkeletonContract(t, db, tenantUUID)

	r := setupGroupSkeletonContractRouter(db, tenantUUID)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/group-codes", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	var listResp map[string]any
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResp))
	items := listResp["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	require.Equal(t, "not_implemented", item["capability_status"])

	webhookBody := bytes.NewBufferString(`{"channel_account_uuid":"22222222-2222-4222-8222-222222222213","code_key":"group-code-001","external_event_id":"evt-001","event_type":"join","occurred_at":"2026-01-01T00:00:00Z"}`)
	webhookReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/channels/wechat/group-code-events", webhookBody)
	webhookReq.Header.Set("Content-Type", "application/json")
	webhookRec := httptest.NewRecorder()
	r.ServeHTTP(webhookRec, webhookReq)
	require.Equal(t, http.StatusOK, webhookRec.Code)
	var webhookResp map[string]any
	require.NoError(t, json.Unmarshal(webhookRec.Body.Bytes(), &webhookResp))
	require.Equal(t, true, webhookResp["success"])
	require.Equal(t, "not_implemented", webhookResp["data"].(map[string]any)["status"])
}

func setupGroupSkeletonContractRouter(db *gorm.DB, tenantUUID string) *gin.Engine {
	repos := acqrepo.NewBundle(db)
	groupHandler := httpacq.NewGroupLiveCodeHandler(acqsvc.NewGroupLiveCodeService(repos.GroupLiveCodes, repos.GroupChatSnapshots))
	webhookHandler := httpwebhooks.NewAcquisitionCodeEventsWebhookHandler()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/api/v1/webhooks/channels/wechat/group-code-events" {
			c.Next()
			return
		}
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	adminGroup := r.Group("/api/v1/admin/leads/acquisition")
	adminGroup.GET("/group-codes", groupHandler.List)

	webhookGroup := r.Group("/api/v1/webhooks")
	webhookGroup.POST("/channels/:channel/group-code-events", webhookHandler.IngestGroup)
	return r
}

func seedGroupCodeForSkeletonContract(t *testing.T, db *gorm.DB, tenantUUID string) {
	t.Helper()
	now := time.Now().UTC()
	item := &acqmodel.GroupLiveCode{
		GroupCodeUUID:      "44444444-4444-4444-8444-444444444213",
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "22222222-2222-4222-8222-222222222213",
		ActivityName:       "群活动",
		Status:             acqmodel.LiveCodeStatusDraft,
		CapabilityStatus:   "not_implemented",
		CreatedBy:          "system",
		UpdatedBy:          "system",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	require.NoError(t, db.Create(item).Error)
}
