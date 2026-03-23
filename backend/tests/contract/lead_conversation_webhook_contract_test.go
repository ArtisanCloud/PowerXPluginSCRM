package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	webhooks "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/webhooks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLeadConversationWebhookContract_IngestAndIdempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"

	db := openContractDB(t, "lead_conversation_webhook_contract")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-main",
		DisplayName:     "企微主账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)

	svc := leadsvc.NewConversationService(
		leadrepo.NewConversationEventRepository(db),
		leadrepo.NewLeadConversationBindingRepository(db),
		leadrepo.NewLeadConversationPendingRepository(db),
		leadrepo.NewLeadRealtimeProjectionRepository(db),
		nil,
		nil,
	).WithLeadRepository(leadrepo.NewLeadRepository(db))
	handler := webhooks.NewWeComConversationWebhookHandler(svc, socialrepo.NewAccountRepository(db))

	r := gin.New()
	r.POST("/api/v1/webhooks/wecom/conversations", handler.Ingest)

	body := `{
		"external_event_id":"evt-1001",
		"channel_account_uuid":"11111111-1111-4111-8111-111111111111",
		"conversation_id":"conv-001",
		"actor_type":"staff",
		"actor_id":"wecom-user-1",
		"direction":"inbound",
		"message_type":"text",
		"content_text":"客户咨询",
		"occurred_at":"2026-02-10T23:30:00Z",
		"raw_payload":{"source":"mock","phone":"13800000001"}
	}`

	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/wecom/conversations", bytes.NewBufferString(body))
	badReq.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	r.ServeHTTP(badRec, badReq)
	require.Equal(t, http.StatusBadRequest, badRec.Code)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/wecom/conversations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-WeCom-Signature", "mock-signature")
	req.Header.Set("X-WeCom-Timestamp", "1770700000")
	req.Header.Set("X-WeCom-Nonce", "nonce-001")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var first map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &first))
	require.Equal(t, true, first["success"])
	data := first["data"].(map[string]any)
	require.Equal(t, true, data["ok"])
	require.Equal(t, true, data["created"])

	reqDup := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/wecom/conversations", bytes.NewBufferString(body))
	reqDup.Header = req.Header.Clone()
	recDup := httptest.NewRecorder()
	r.ServeHTTP(recDup, reqDup)
	require.Equal(t, http.StatusOK, recDup.Code)

	var dup map[string]any
	require.NoError(t, json.Unmarshal(recDup.Body.Bytes(), &dup))
	require.Equal(t, true, dup["success"])
	dupData := dup["data"].(map[string]any)
	require.Equal(t, false, dupData["created"])
}
