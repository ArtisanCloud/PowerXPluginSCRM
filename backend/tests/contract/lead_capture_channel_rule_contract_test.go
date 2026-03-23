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

func TestLeadCaptureChannelRuleContract_WebhookAutoCreateLeadWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"

	db := openContractDB(t, "lead_capture_channel_rule_contract")
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
	_, err := leadrepo.NewChannelRuleRepository(db).UpsertAutoCreateLeadRule(
		t.Context(), tenantUUID, "wechat", "wecom", true,
	)
	require.NoError(t, err)

	leadRepository := leadrepo.NewLeadRepository(db)
	svc := leadsvc.NewConversationService(
		leadrepo.NewConversationEventRepository(db),
		leadrepo.NewLeadConversationBindingRepository(db),
		leadrepo.NewLeadConversationPendingRepository(db),
		leadrepo.NewLeadRealtimeProjectionRepository(db),
		nil,
		nil,
	).WithLeadRepository(leadRepository).
		WithLeadService(leadsvc.NewLeadService(leadRepository)).
		WithChannelRuleRepository(leadrepo.NewChannelRuleRepository(db))
	handler := webhooks.NewWeComConversationWebhookHandler(svc, socialrepo.NewAccountRepository(db))

	r := gin.New()
	r.POST("/api/v1/webhooks/wecom/conversations", handler.Ingest)

	body := `{
		"external_event_id":"evt-rule-contract-001",
		"channel_account_uuid":"11111111-1111-4111-8111-111111111111",
		"conversation_id":"conv-rule-001",
		"actor_type":"customer",
		"actor_id":"customer-1",
		"direction":"inbound",
		"message_type":"text",
		"content_text":"我想了解一下",
		"occurred_at":"2026-02-10T23:30:00Z",
		"raw_payload":{"phone":"13800001001","display_name":"客户A"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/wecom/conversations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-WeCom-Signature", "mock-signature")
	req.Header.Set("X-WeCom-Timestamp", "1770700000")
	req.Header.Set("X-WeCom-Nonce", "nonce-001")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])

	var leadCount int64
	require.NoError(t, db.Table("lead_capture_leads").Where("tenant_uuid = ?", tenantUUID).Count(&leadCount).Error)
	require.EqualValues(t, 1, leadCount)
}
