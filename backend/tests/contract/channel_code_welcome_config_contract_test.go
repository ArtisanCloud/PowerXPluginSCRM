package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelCodeWelcomeConfigContract_StatusSaveHistory(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000012"
	db := openContractDB(t, "channel_code_welcome_contract")
	require.NoError(t, ensureChannelCodeContractTables(db))
	r := setupChannelCodeContractRouter(db, tenantUUID)

	createPayload := map[string]any{
		"channel":              "wechat",
		"app_type":             "wecom",
		"channel_account_uuid": "11111111-1111-4111-8111-111111111112",
		"code_key":             "wecom-sales-002",
		"display_name":         "社群获客入口",
		"target_type":          "group",
		"target_id":            "group-002",
	}
	createBody, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/channel-codes", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createResp map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	codeUUID := createResp["data"].(map[string]any)["code_uuid"].(string)

	statusPayload := []byte(`{"status":"active"}`)
	statusReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/leads/channel-codes/"+codeUUID+"/status", bytes.NewBuffer(statusPayload))
	statusReq.Header.Set("Content-Type", "application/json")
	statusRec := httptest.NewRecorder()
	r.ServeHTTP(statusRec, statusReq)
	require.Equal(t, http.StatusOK, statusRec.Code)

	welcomePayload := []byte(`{"welcome_enabled":true,"message_content":{"text":"欢迎关注"}}`)
	welcomeReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/leads/channel-codes/"+codeUUID+"/welcome-config", bytes.NewBuffer(welcomePayload))
	welcomeReq.Header.Set("Content-Type", "application/json")
	welcomeRec := httptest.NewRecorder()
	r.ServeHTTP(welcomeRec, welcomeReq)
	require.Equal(t, http.StatusOK, welcomeRec.Code)

	var welcomeResp map[string]any
	require.NoError(t, json.Unmarshal(welcomeRec.Body.Bytes(), &welcomeResp))
	require.Equal(t, "pending", welcomeResp["data"].(map[string]any)["sync_status"])

	historyReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/channel-codes/"+codeUUID+"/welcome-config/history?limit=10", nil)
	historyRec := httptest.NewRecorder()
	r.ServeHTTP(historyRec, historyReq)
	require.Equal(t, http.StatusOK, historyRec.Code)

	var historyResp map[string]any
	require.NoError(t, json.Unmarshal(historyRec.Body.Bytes(), &historyResp))
	items := historyResp["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)
}
