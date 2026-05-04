package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type groupChatContractRecord struct {
	ChatID              string    `json:"chat_id"`
	Name                string    `json:"name"`
	OwnerUserID         string    `json:"owner_userid"`
	MemberCount         int       `json:"member_count"`
	SourceGroupCodeUUID string    `json:"source_group_code_uuid,omitempty"`
	SourceConfigID      string    `json:"source_config_id,omitempty"`
	CreateTime          time.Time `json:"create_time"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func TestGroupChatSyncContract_ListAndDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000252"
	r := setupGroupChatSyncContractRouter(tenantUUID)

	syncBody := bytes.NewBufferString(`{"channel_account_uuid":"30aa4d9f-6768-4fdf-97c8-66844422a7a5","mode":"incremental"}`)
	syncReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/acquisition/group-chats/sync", syncBody)
	syncReq.Header.Set("Content-Type", "application/json")
	syncRec := httptest.NewRecorder()
	r.ServeHTTP(syncRec, syncReq)
	require.Equal(t, http.StatusOK, syncRec.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/group-chats?limit=20", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)

	var listResp map[string]any
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResp))
	items := listResp["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 2)
	first := items[0].(map[string]any)
	require.NotEmpty(t, first["chat_id"])

	chatID := first["chat_id"].(string)
	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/group-chats/"+chatID, nil)
	detailRec := httptest.NewRecorder()
	r.ServeHTTP(detailRec, detailReq)
	require.Equal(t, http.StatusOK, detailRec.Code)

	var detailResp map[string]any
	require.NoError(t, json.Unmarshal(detailRec.Body.Bytes(), &detailResp))
	detail := detailResp["data"].(map[string]any)
	require.Equal(t, chatID, detail["chat_id"])
	require.NotNil(t, detail["member_count"])
}

func setupGroupChatSyncContractRouter(tenantUUID string) *gin.Engine {
	store := map[string]*groupChatContractRecord{}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})

	group := r.Group("/api/v1/admin/leads/acquisition")
	group.POST("/group-chats/sync", func(c *gin.Context) {
		now := time.Now().UTC()
		store["chat-001"] = &groupChatContractRecord{
			ChatID:              "chat-001",
			Name:                "活动群A",
			OwnerUserID:         "zhangsan",
			MemberCount:         12,
			SourceGroupCodeUUID: "code-001",
			SourceConfigID:      "cfg-001",
			CreateTime:          now.Add(-2 * time.Hour),
			UpdatedAt:           now,
		}
		store["chat-002"] = &groupChatContractRecord{
			ChatID:      "chat-002",
			Name:        "活动群B",
			OwnerUserID: "lisi",
			MemberCount: 8,
			CreateTime:  now.Add(-1 * time.Hour),
			UpdatedAt:   now,
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"job_status": "success", "synced_count": len(store)}})
	})

	group.GET("/group-chats", func(c *gin.Context) {
		items := make([]*groupChatContractRecord, 0, len(store))
		for _, v := range store {
			items = append(items, v)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items}})
	})

	group.GET("/group-chats/:chat_id", func(c *gin.Context) {
		chatID := c.Param("chat_id")
		item, ok := store[chatID]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
	})

	return r
}
