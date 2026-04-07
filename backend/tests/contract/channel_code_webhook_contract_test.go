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
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	webhooks "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/webhooks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChannelCodeWebhookContract_IngestAndIdempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000051"
	accountUUID := "11111111-1111-4111-8111-111111111151"

	db := openContractDB(t, "channel_code_webhook_contract")
	require.NoError(t, ensureChannelCodeEventContractTables(db))

	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-01",
		DisplayName:     "企微账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-01",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}).Error)

	codeUUID := uuid.NewString()
	require.NoError(t, db.Create(&domainmodel.ChannelCode{
		CodeUUID:           codeUUID,
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: accountUUID,
		CodeKey:            "wecom-us2-001",
		DisplayName:        "US2渠道码",
		TargetType:         "group",
		TargetID:           "group-001",
		Status:             domainmodel.ChannelCodeStatusActive,
		CreatedBy:          "system",
		UpdatedBy:          "system",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}).Error)

	repos := domainrepo.NewBundle(db)
	leadRepository := leadrepo.NewLeadRepository(db)
	attributionSvc := leadsvc.NewAttributionService(repos.Attributions, leadRepository, leadsvc.NewLeadService(leadRepository))
	eventSvc := leadsvc.NewChannelCodeEventService(repos.ChannelCodeEvents, repos.ChannelCodes, socialrepo.NewAccountRepository(db), attributionSvc, nil)
	handler := webhooks.NewChannelCodeEventsWebhookHandler(eventSvc)

	r := gin.New()
	r.POST("/api/v1/webhooks/channels/:channel/code-events", handler.Ingest)

	body := map[string]any{
		"channel_account_uuid": accountUUID,
		"code_key":             "wecom-us2-001",
		"external_event_id":    "evt-code-001",
		"event_type":           "join",
		"occurred_at":          "2026-03-24T10:00:00Z",
		"payload": map[string]any{
			"phone": "13800003333",
			"name":  "李四",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/channels/wechat/code-events", bytes.NewBuffer(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	var resp1 map[string]any
	require.NoError(t, json.Unmarshal(rec1.Body.Bytes(), &resp1))
	require.Equal(t, true, resp1["success"])
	data1 := resp1["data"].(map[string]any)
	require.Equal(t, true, data1["created"])
	require.Equal(t, false, data1["idempotent_hit"])
	require.NotEmpty(t, data1["lead_uuid"])

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/channels/wechat/code-events", bytes.NewBuffer(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)

	var resp2 map[string]any
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &resp2))
	require.Equal(t, true, resp2["success"])
	data2 := resp2["data"].(map[string]any)
	require.Equal(t, false, data2["created"])
	require.Equal(t, true, data2["idempotent_hit"])
}
