package contract

import (
	"bytes"
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

func TestLeadConversationAdminContract_ListAndBind(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"
	now := time.Now().UTC()

	db := openContractDB(t, "lead_conversation_admin_contract")
	require.NoError(t, db.Create(&leadmodel.Lead{LeadUUID: leadUUID, TenantUUID: tenantUUID, DisplayName: "Alice", Status: leadmodel.LeadStatusCaptured, CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&leadmodel.LeadConversationBinding{
		BindingUUID:        "20000000-0000-4000-8000-000000000001",
		TenantUUID:         tenantUUID,
		LeadUUID:           leadUUID,
		ConversationID:     "conv-001",
		ChannelAccountUUID: accountUUID,
		BindSource:         "manual",
		Status:             "active",
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error)
	require.NoError(t, db.Create(&leadmodel.LeadRealtimeProjection{
		ProjectionUUID:  "30000000-0000-4000-8000-000000000001",
		TenantUUID:      tenantUUID,
		LeadUUID:        leadUUID,
		ConversationID:  "conv-001",
		LatestMessage:   "hello",
		LatestActorType: "staff",
		LatestAt:        now,
		UnreadCount:     1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error)
	require.NoError(t, db.Create(&leadmodel.ConversationEvent{
		EventUUID:          "40000000-0000-4000-8000-000000000001",
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: accountUUID,
		ExternalEventID:    "evt-1",
		IdempotencyKey:     "idmp-1",
		ConversationID:     "conv-001",
		ActorType:          "staff",
		ActorID:            "staff-1",
		Direction:          "inbound",
		MessageType:        "text",
		ContentText:        "hello",
		OccurredAt:         now,
		CreatedAt:          now,
	}).Error)

	svc := leadsvc.NewConversationService(
		leadrepo.NewConversationEventRepository(db),
		leadrepo.NewLeadConversationBindingRepository(db),
		leadrepo.NewLeadConversationPendingRepository(db),
		leadrepo.NewLeadRealtimeProjectionRepository(db),
		nil,
		nil,
	).WithLeadRepository(leadrepo.NewLeadRepository(db))
	handler := httplead.NewConversationHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.GET("/api/v1/admin/leads/:lead_id/conversations", handler.ListLeadConversations)
	r.GET("/api/v1/admin/conversations/:conversation_id/events", handler.ListConversationEvents)
	r.POST("/api/v1/admin/leads/:lead_id/conversations/bind", handler.BindConversation)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/"+leadUUID+"/conversations", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)

	var listResp map[string]any
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResp))
	listData := listResp["data"].(map[string]any)
	conversations := listData["conversations"].([]any)
	require.Len(t, conversations, 1)

	eventsReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/conversations/conv-001/events?limit=20", nil)
	eventsRec := httptest.NewRecorder()
	r.ServeHTTP(eventsRec, eventsReq)
	require.Equal(t, http.StatusOK, eventsRec.Code)

	bindBody := bytes.NewBufferString(`{"conversation_id":"conv-002","channel_account_uuid":"` + accountUUID + `"}`)
	bindReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/"+leadUUID+"/conversations/bind", bindBody)
	bindReq.Header.Set("Content-Type", "application/json")
	bindRec := httptest.NewRecorder()
	r.ServeHTTP(bindRec, bindReq)
	require.Equal(t, http.StatusOK, bindRec.Code)

	listReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/"+leadUUID+"/conversations", nil)
	listRec2 := httptest.NewRecorder()
	r.ServeHTTP(listRec2, listReq2)
	require.Equal(t, http.StatusOK, listRec2.Code)

	var listResp2 map[string]any
	require.NoError(t, json.Unmarshal(listRec2.Body.Bytes(), &listResp2))
	listData2 := listResp2["data"].(map[string]any)
	conversations2 := listData2["conversations"].([]any)
	require.GreaterOrEqual(t, len(conversations2), 1)
}
